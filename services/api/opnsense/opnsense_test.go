package opnsense_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"soar-platform/services/api/connectors"
	"soar-platform/services/api/opnsense"
)

// fakeAlias stands in for OPNsense's SOAR_TEMP_BLOCK alias state, backed by
// the exact list/add/delete response shapes confirmed against the real lab
// VM (see adr/0013-real-opnsense-lab-connector.md) — this proves the
// connector's HTTP client and idempotency logic without any live
// infrastructure, safe to run in default CI.
type fakeAlias struct {
	mu  sync.Mutex
	ips map[string]bool
}

func newFakeServer(t *testing.T) (*fakeAlias, *httptest.Server) {
	t.Helper()
	fa := &fakeAlias{ips: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/firewall/alias_util/list/SOAR_TEMP_BLOCK", func(w http.ResponseWriter, r *http.Request) {
		fa.mu.Lock()
		rows := make([]map[string]string, 0, len(fa.ips))
		for ip := range fa.ips {
			rows = append(rows, map[string]string{"ip": ip})
		}
		fa.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": len(rows), "rowCount": len(rows), "current": 1, "rows": rows,
		})
	})
	mux.HandleFunc("/api/firewall/alias_util/add/SOAR_TEMP_BLOCK", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Address string `json:"address"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		fa.mu.Lock()
		fa.ips[body.Address] = true
		fa.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "done"})
	})
	mux.HandleFunc("/api/firewall/alias_util/delete/SOAR_TEMP_BLOCK", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Address string `json:"address"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		fa.mu.Lock()
		delete(fa.ips, body.Address)
		fa.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "done"})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return fa, ts
}

// TestOpnsenseConnectorContract is the real-connector half of M5's
// connector-contract-tests gate, mirroring
// services/api/integration/connector_contract_test.go's mock-connector
// coverage but against a fake HTTP server instead of a shared Postgres pool
// — this connector has no database of its own.
func TestOpnsenseConnectorContract(t *testing.T) {
	var current *fakeAlias

	newConnector := func(t *testing.T) connectors.Connector {
		t.Helper()
		fa, ts := newFakeServer(t)
		current = fa
		return opnsense.NewWithConfig(opnsense.Config{
			BaseURL:   ts.URL,
			APIKey:    "test-key",
			APISecret: "test-secret",
			Alias:     "SOAR_TEMP_BLOCK",
			Timeout:   5 * time.Second,
		})
	}

	tc := connectors.ContractTestCase{
		Valid: connectors.Invocation{
			Intent: "add_temp_block",
			Target: "SOAR_TEMP_BLOCK",
			Params: map[string]any{"tenant_id": "lab", "action_id": "act-1", "target_value": "198.51.100.10"},
		},
		Malformed: connectors.Invocation{
			Intent: "add_temp_block",
			Target: "SOAR_TEMP_BLOCK",
			Params: map[string]any{"tenant_id": "lab"},
		},
		OutsideAllowlistTarget: "external-firewall-provider.example.com",
		CountEffects: func(t *testing.T) int {
			t.Helper()
			current.mu.Lock()
			defer current.mu.Unlock()
			return len(current.ips)
		},
	}
	connectors.RunContractTests(t, newConnector, tc)
}

// TestReleaseTemp proves the rollback path independently, since
// RunContractTests only exercises Invoke/HealthCheck/Manifest (the shared
// connectors.Connector surface), not the package-specific Release helper
// every connector in this codebase also exposes (see
// services/api/network.Release).
func TestReleaseTemp(t *testing.T) {
	fa, ts := newFakeServer(t)
	c := opnsense.NewWithConfig(opnsense.Config{
		BaseURL: ts.URL, APIKey: "k", APISecret: "s", Alias: "SOAR_TEMP_BLOCK", Timeout: 5 * time.Second,
	})
	ctx := t.Context()

	// Releasing an IP that was never blocked is a no-op, not an error.
	if err := c.ReleaseTemp(ctx, "198.51.100.20"); err != nil {
		t.Fatalf("releasing an absent IP: %v", err)
	}

	fa.mu.Lock()
	fa.ips["198.51.100.20"] = true
	fa.mu.Unlock()

	if err := c.ReleaseTemp(ctx, "198.51.100.20"); err != nil {
		t.Fatalf("releasing a present IP: %v", err)
	}
	fa.mu.Lock()
	_, stillPresent := fa.ips["198.51.100.20"]
	fa.mu.Unlock()
	if stillPresent {
		t.Error("expected ReleaseTemp to remove the IP from the alias")
	}
}

// TestPreview proves the read-only preview_temp_block path never mutates
// the alias.
func TestPreview(t *testing.T) {
	fa, ts := newFakeServer(t)
	c := opnsense.NewWithConfig(opnsense.Config{
		BaseURL: ts.URL, APIKey: "k", APISecret: "s", Alias: "SOAR_TEMP_BLOCK", Timeout: 5 * time.Second,
	})
	ctx := t.Context()

	res, err := c.Preview(ctx, "198.51.100.30")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if res.AlreadyBlocked {
		t.Error("expected AlreadyBlocked to be false for an unblocked IP")
	}
	fa.mu.Lock()
	n := len(fa.ips)
	fa.mu.Unlock()
	if n != 0 {
		t.Errorf("Preview must not mutate the alias, got %d entries", n)
	}
}
