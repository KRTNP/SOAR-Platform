// Package opnsense implements the M5 real OPNsense connector
// (docs/05-integrations/lab-integration-profile.md's OPNsense contract): it
// calls the lab OPNsense VM's alias_util REST API to add/remove a source IP
// from the SOAR_TEMP_BLOCK alias, replacing services/api/network's mock for
// callers that opt into OPNSENSE_ENABLE_REAL. See
// adr/0013-real-opnsense-lab-connector.md for the manifest, the API quirks
// discovered against the real lab VM, and why the mock stays the default.
package opnsense

import (
	"bytes"
	"cmp"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"soar-platform/services/api/connectors"
)

const provider = "opnsense"

// manifest is hand-synced to examples/opnsense-connector.json, the checked-in
// schema-validated fixture — same disclosed hand-sync convention as
// adr/0010-mock-enrichment-and-ticketing-connectors.md and
// adr/0011-approval-gated-mock-network-block.md.
var manifest = connectors.Manifest{
	ID:                 "connector-opnsense-lab",
	Owner:              "security-operations",
	Provider:           provider,
	Intents:            []string{"preview_temp_block", "add_temp_block", "remove_temp_block", "health"},
	ErrorClasses:       []string{"authentication", "authorization", "timeout", "rate_limit", "provider_5xx", "postcondition_mismatch"},
	DryRun:             true,
	Idempotent:         true,
	DataClassification: "internal",
	TargetAllowlist:    []string{"opnsense-lab", "SOAR_TEMP_BLOCK"},
}

// Error classes beyond the shared connectors.ErrClass* set — these are
// specific to a connector with a real network target and match
// manifest.ErrorClasses above.
const (
	ErrClassAuthentication      connectors.ErrorClass = "authentication"
	ErrClassAuthorization       connectors.ErrorClass = "authorization"
	ErrClassTimeout             connectors.ErrorClass = "timeout"
	ErrClassRateLimit           connectors.ErrorClass = "rate_limit"
	ErrClassProvider5xx         connectors.ErrorClass = "provider_5xx"
	ErrClassPostconditionMismatch connectors.ErrorClass = "postcondition_mismatch"
)

// Config is loaded from OPNSENSE_* environment variables only — never from a
// committed file — matching AGENTS.md's "never commit credentials" rule.
type Config struct {
	BaseURL            string
	APIKey             string
	APISecret          string
	Alias              string
	Timeout            time.Duration
	InsecureSkipVerify bool
}

// Enabled reports whether the caller asked for the real connector.
// OPNSENSE_ENABLE_REAL is the master switch: even with valid credentials
// present, the mock connector stays wired in unless this is "true" — see
// adr/0013.
func Enabled() bool {
	return os.Getenv("OPNSENSE_ENABLE_REAL") == "true"
}

// LoadConfig reads OPNSENSE_BASE_URL, OPNSENSE_API_KEY and
// OPNSENSE_API_SECRET (all required), plus optional OPNSENSE_ALIAS (default
// SOAR_TEMP_BLOCK), OPNSENSE_TIMEOUT_SECONDS (default 15, matching the
// manifest) and OPNSENSE_INSECURE_SKIP_VERIFY (default false — the lab VM's
// self-signed cert requires opting into this explicitly).
func LoadConfig() (Config, error) {
	baseURL := os.Getenv("OPNSENSE_BASE_URL")
	apiKey := os.Getenv("OPNSENSE_API_KEY")
	apiSecret := os.Getenv("OPNSENSE_API_SECRET")
	if baseURL == "" || apiKey == "" || apiSecret == "" {
		return Config{}, fmt.Errorf("OPNSENSE_BASE_URL, OPNSENSE_API_KEY and OPNSENSE_API_SECRET are required")
	}

	timeoutSeconds := 15
	if v := os.Getenv("OPNSENSE_TIMEOUT_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("OPNSENSE_TIMEOUT_SECONDS must be a positive integer: %q", v)
		}
		timeoutSeconds = n
	}

	return Config{
		BaseURL:            strings.TrimRight(baseURL, "/"),
		APIKey:             apiKey,
		APISecret:          apiSecret,
		Alias:              cmp.Or(os.Getenv("OPNSENSE_ALIAS"), "SOAR_TEMP_BLOCK"),
		Timeout:            time.Duration(timeoutSeconds) * time.Second,
		InsecureSkipVerify: os.Getenv("OPNSENSE_INSECURE_SKIP_VERIFY") == "true",
	}, nil
}

// Connector is the opnsense side of the connectors.Connector contract. It
// holds no per-call transaction (unlike the mock connectors) — every method
// is a direct HTTP call to the lab VM's alias_util API, safe to share across
// calls.
type Connector struct {
	cfg    Config
	client *http.Client
}

// NewWithConfig builds a Connector that talks to the real lab VM. Callers
// should construct this once at startup and reuse it, not per-request.
func NewWithConfig(cfg Config) *Connector {
	return &Connector{
		cfg: cfg,
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}},
		},
	}
}

func (c *Connector) Manifest() connectors.Manifest { return manifest }

// HealthCheck lists the configured alias — read-only, satisfies
// schemas/connector.schema.json's health_check.read_only.
func (c *Connector) HealthCheck(ctx context.Context) error {
	_, err := c.list(ctx)
	return err
}

func inAllowlist(target string) bool {
	for _, t := range manifest.TargetAllowlist {
		if t == target {
			return true
		}
	}
	return false
}

// Invoke implements the connectors.Connector contract, performing
// add_temp_block: it adds target_value to the configured alias if not
// already present. Params must carry "tenant_id", "action_id" and
// "target_value" (string) — the same shape services/api/network.Invoke
// accepts, so services/api/actions can call either connector identically.
func (c *Connector) Invoke(ctx context.Context, inv connectors.Invocation) (connectors.Result, error) {
	if inv.Target != "" && !inAllowlist(inv.Target) {
		return connectors.Result{}, &connectors.ConnectorError{
			Class: connectors.ErrClassPolicyDenied, Message: "target not in allowlist",
		}
	}
	tenantID, _ := inv.Params["tenant_id"].(string)
	actionID, _ := inv.Params["action_id"].(string)
	targetValue, _ := inv.Params["target_value"].(string)
	if tenantID == "" || actionID == "" || targetValue == "" {
		return connectors.Result{}, &connectors.ConnectorError{
			Class: connectors.ErrClassValidation, Message: "tenant_id, action_id and target_value are required",
		}
	}

	present, err := c.contains(ctx, targetValue)
	if err != nil {
		return connectors.Result{}, err
	}
	if present {
		return connectors.Result{Effect: false}, nil
	}
	if err := c.add(ctx, targetValue); err != nil {
		return connectors.Result{}, err
	}
	return connectors.Result{Effect: true}, nil
}

// ReleaseTemp removes ip from the configured alias, matching
// services/api/network.Release's rollback role but keyed on the IP itself
// rather than a Postgres row, since OPNsense's alias is the only state.
// No-op-safe if the IP is already absent.
func (c *Connector) ReleaseTemp(ctx context.Context, ip string) error {
	present, err := c.contains(ctx, ip)
	if err != nil {
		return err
	}
	if !present {
		return nil
	}
	return c.remove(ctx, ip)
}

// PreviewResult is the read-only answer to "what would add_temp_block do
// right now", for the preview_temp_block intent.
type PreviewResult struct {
	AlreadyBlocked bool
	Alias          string
}

// Preview reports whether ip is already in the alias, without mutating
// anything — the preview_temp_block intent from
// docs/05-integrations/lab-integration-profile.md. Not yet wired into
// services/api/actions.Handler.Preview (see adr/0013).
func (c *Connector) Preview(ctx context.Context, ip string) (PreviewResult, error) {
	present, err := c.contains(ctx, ip)
	if err != nil {
		return PreviewResult{}, err
	}
	return PreviewResult{AlreadyBlocked: present, Alias: c.cfg.Alias}, nil
}

// listResponse mirrors OPNsense's confirmed alias_util/list/{alias} shape:
// {"total":0,"rowCount":0,"current":1,"rows":[{"ip":"x.x.x.x"}]}.
type listResponse struct {
	Rows []struct {
		IP string `json:"ip"`
	} `json:"rows"`
}

// statusResponse mirrors alias_util/add and /delete's confirmed shape:
// {"status":"done"} on success, {"status":"failed"} on failure.
type statusResponse struct {
	Status string `json:"status"`
}

func (c *Connector) list(ctx context.Context) (listResponse, error) {
	var out listResponse
	if err := c.do(ctx, http.MethodGet, "/api/firewall/alias_util/list/"+c.cfg.Alias, nil, &out); err != nil {
		return listResponse{}, err
	}
	return out, nil
}

func (c *Connector) contains(ctx context.Context, ip string) (bool, error) {
	rows, err := c.list(ctx)
	if err != nil {
		return false, err
	}
	for _, r := range rows.Rows {
		if r.IP == ip {
			return true, nil
		}
	}
	return false, nil
}

func (c *Connector) add(ctx context.Context, ip string) error {
	return c.mutate(ctx, "/api/firewall/alias_util/add/"+c.cfg.Alias, ip, true)
}

func (c *Connector) remove(ctx context.Context, ip string) error {
	return c.mutate(ctx, "/api/firewall/alias_util/delete/"+c.cfg.Alias, ip, false)
}

// mutate calls add/delete and verifies the alias actually reflects the
// change afterward — alias_util's own "status":"done" is not, by itself,
// proof the pf table changed (empirically, a bad path segment vs. a bad POST
// field both fail silently in different ways during this connector's
// development), so a genuine postcondition check is worth the extra list
// call for a low-volume, high-risk operation like this one.
func (c *Connector) mutate(ctx context.Context, path, ip string, wantPresent bool) error {
	var out statusResponse
	if err := c.do(ctx, http.MethodPost, path, map[string]string{"address": ip}, &out); err != nil {
		return err
	}
	if out.Status != "done" {
		return &connectors.ConnectorError{Class: connectors.ErrClassProviderError, Message: fmt.Sprintf("opnsense returned status %q", out.Status)}
	}
	present, err := c.contains(ctx, ip)
	if err != nil {
		return err
	}
	if present != wantPresent {
		return &connectors.ConnectorError{Class: ErrClassPostconditionMismatch, Message: "alias state did not match the requested change after mutation"}
	}
	return nil
}

func (c *Connector) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return &connectors.ConnectorError{Class: connectors.ErrClassValidation, Message: "encoding request body: " + err.Error()}
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, reqBody)
	if err != nil {
		return &connectors.ConnectorError{Class: connectors.ErrClassValidation, Message: "building request: " + err.Error()}
	}
	req.SetBasicAuth(c.cfg.APIKey, c.cfg.APISecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return &connectors.ConnectorError{Class: ErrClassTimeout, Message: "request timed out: " + err.Error()}
		}
		return &connectors.ConnectorError{Class: connectors.ErrClassProviderError, Message: "request failed: " + err.Error()}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return &connectors.ConnectorError{Class: ErrClassAuthentication, Message: "opnsense rejected the API key/secret"}
	case resp.StatusCode == http.StatusForbidden:
		return &connectors.ConnectorError{Class: ErrClassAuthorization, Message: "opnsense API user lacks the required privilege"}
	case resp.StatusCode == http.StatusTooManyRequests:
		return &connectors.ConnectorError{Class: ErrClassRateLimit, Message: "opnsense rate-limited this request"}
	case resp.StatusCode >= 500:
		return &connectors.ConnectorError{Class: ErrClassProvider5xx, Message: fmt.Sprintf("opnsense returned %d", resp.StatusCode)}
	case resp.StatusCode >= 400:
		return &connectors.ConnectorError{Class: connectors.ErrClassProviderError, Message: fmt.Sprintf("opnsense returned %d", resp.StatusCode)}
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &connectors.ConnectorError{Class: connectors.ErrClassProviderError, Message: "decoding opnsense response: " + err.Error()}
	}
	return nil
}
