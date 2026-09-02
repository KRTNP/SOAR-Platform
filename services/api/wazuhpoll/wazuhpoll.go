// Package wazuhpoll implements the real half of the Wazuh integration
// docs/05-integrations/wazuh.md describes ("authenticated webhook/API →
// SOAR ingestion → normalize → deduplicate → ... → case"): poll a real
// Wazuh Indexer for new alerts and forward each one into this service's own
// POST /v1/events over loopback, the same path a manual curl or the
// existing integration tests already use. See adr/0017 for the design
// trade-offs (polling vs. a manager-side webhook, heuristic severity/
// confidence mapping) and adr/0018 for the checkpoint persistence this
// package originally shipped without.
package wazuhpoll

import (
	"bytes"
	"cmp"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is loaded from WAZUH_* environment variables only — mirrors
// services/api/opnsense's OPNSENSE_* convention (a master switch plus
// required connection fields, never a config file).
type Config struct {
	IndexerBaseURL      string
	IndexerUsername     string
	IndexerPassword     string
	InsecureSkipVerify  bool
	PollIntervalSeconds int
}

// Enabled is the master switch — default false, same posture ADR-0013 uses
// for OPNSENSE_ENABLE_REAL: an environment with no Wazuh credentials keeps
// this off rather than failing startup.
func Enabled() bool {
	return os.Getenv("WAZUH_POLL_ENABLE_REAL") == "true"
}

// LoadConfig reads WAZUH_INDEXER_BASE_URL, WAZUH_INDEXER_USERNAME and
// WAZUH_INDEXER_PASSWORD (all required), plus optional
// WAZUH_INDEXER_INSECURE_SKIP_VERIFY (default false — the lab indexer's
// self-signed cert needs this true) and WAZUH_POLL_INTERVAL_SECONDS
// (default 15).
func LoadConfig() (Config, error) {
	baseURL := os.Getenv("WAZUH_INDEXER_BASE_URL")
	username := os.Getenv("WAZUH_INDEXER_USERNAME")
	password := os.Getenv("WAZUH_INDEXER_PASSWORD")
	if baseURL == "" || username == "" || password == "" {
		return Config{}, fmt.Errorf("WAZUH_INDEXER_BASE_URL, WAZUH_INDEXER_USERNAME and WAZUH_INDEXER_PASSWORD are required")
	}

	interval := 15
	if v := os.Getenv("WAZUH_POLL_INTERVAL_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("WAZUH_POLL_INTERVAL_SECONDS must be a positive integer: %q", v)
		}
		interval = n
	}

	return Config{
		IndexerBaseURL:      baseURL,
		IndexerUsername:     username,
		IndexerPassword:     password,
		InsecureSkipVerify:  os.Getenv("WAZUH_INDEXER_INSECURE_SKIP_VERIFY") == "true",
		PollIntervalSeconds: interval,
	}, nil
}

// batchSize matches services/api/scheduler's own hardcoded candidate limit —
// a fixed constant rather than a config knob, revisit only if lab alert
// volume ever actually approaches it.
const batchSize = 50

type searchResponse struct {
	Hits struct {
		Hits []struct {
			ID     string          `json:"_id"`
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// alertSource is the subset of a wazuh-alerts-* document this package reads.
// Extra fields (agent, decoder, manager, location, full_log) are carried
// into the forwarded event's payload as-is, not modeled further here.
type alertSource struct {
	Timestamp string `json:"@timestamp"`
	Rule      struct {
		ID          string   `json:"id"`
		Level       int      `json:"level"`
		Description string   `json:"description"`
		Groups      []string `json:"groups"`
	} `json:"rule"`
	Agent struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"agent"`
	Manager struct {
		Name string `json:"name"`
	} `json:"manager"`
	Decoder struct {
		Name string `json:"name"`
	} `json:"decoder"`
	Location string         `json:"location"`
	FullLog  string         `json:"full_log"`
	Data     map[string]any `json:"data"`
}

// Run polls on a ticker until ctx is cancelled, logging and continuing past
// any single Poll error (matching services/api/scheduler.Run's posture —
// a transient indexer or ingest failure should not kill the whole loop).
func Run(ctx context.Context, pool *pgxpool.Pool, cfg Config, ingestBaseURL string, logger *slog.Logger) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}, //nolint:gosec // lab-only, explicit opt-in
		},
	}
	since, err := LoadCheckpoint(ctx, pool)
	if err != nil {
		logger.Error("loading wazuh poll checkpoint, starting from now instead", "err", err)
		since = time.Now().UTC()
	}
	ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			newSince, forwarded, err := Poll(ctx, client, cfg, ingestBaseURL, since, logger)
			if err != nil {
				logger.Error("wazuh poll failed", "err", err)
				continue
			}
			if err := SaveCheckpoint(ctx, pool, newSince); err != nil {
				logger.Error("saving wazuh poll checkpoint", "err", err)
			}
			since = newSince
			if forwarded > 0 {
				logger.Info("wazuh poll forwarded alerts", "count", forwarded, "checkpoint", since)
			}
		}
	}
}

// LoadCheckpoint reads the single wazuhpoll_checkpoint row (adr/0018). No
// row yet (first-ever run on a fresh database) is not an error — it returns
// the zero time and lets Run fall back to time.Now(), the exact behavior
// this package originally always had.
func LoadCheckpoint(ctx context.Context, pool *pgxpool.Pool) (time.Time, error) {
	var t time.Time
	err := pool.QueryRow(ctx, `select last_polled_at from wazuhpoll_checkpoint where id`).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Now().UTC(), nil
	}
	return t, err
}

// SaveCheckpoint upserts the single wazuhpoll_checkpoint row.
func SaveCheckpoint(ctx context.Context, pool *pgxpool.Pool, t time.Time) error {
	_, err := pool.Exec(ctx, `
		insert into wazuhpoll_checkpoint (id, last_polled_at) values (true, $1)
		on conflict (id) do update set last_polled_at = excluded.last_polled_at`, t)
	return err
}

// Poll runs one pass: fetch alerts newer than since (ascending order, up to
// batchSize), forward each to POST {ingestBaseURL}/v1/events, and return the
// checkpoint to resume from next time. The checkpoint advances to the last
// alert in the fetched batch regardless of whether any single one failed to
// forward — a failed alert is logged and skipped, not retried, so this
// package makes forward progress even if one alert is permanently
// malformed. See adr/0017's Consequences for why that trade-off (matching
// this codebase's existing scheduler and connector "log and skip" idiom)
// was chosen over blocking the whole batch on one bad item.
func Poll(ctx context.Context, client *http.Client, cfg Config, ingestBaseURL string, since time.Time, logger *slog.Logger) (newSince time.Time, forwarded int, err error) {
	hits, err := search(ctx, client, cfg, since)
	if err != nil {
		return since, 0, fmt.Errorf("querying wazuh indexer: %w", err)
	}
	if len(hits) == 0 {
		return since, 0, nil
	}

	newSince = since
	for _, hit := range hits {
		var src alertSource
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			logger.Error("wazuh alert has unparseable _source, skipping", "id", hit.ID, "err", err)
			continue
		}
		ts, err := time.Parse(time.RFC3339, src.Timestamp)
		if err != nil {
			logger.Error("wazuh alert has unparseable @timestamp, skipping", "id", hit.ID, "err", err)
			continue
		}
		if ts.After(newSince) {
			newSince = ts
		}

		body := toIngestRequest(hit.ID, src)
		if err := forward(ctx, client, ingestBaseURL, hit.ID, body); err != nil {
			logger.Error("forwarding wazuh alert failed, skipping", "id", hit.ID, "err", err)
			continue
		}
		forwarded++
	}
	return newSince, forwarded, nil
}

func search(ctx context.Context, client *http.Client, cfg Config, since time.Time) ([]struct {
	ID     string          `json:"_id"`
	Source json.RawMessage `json:"_source"`
}, error) {
	query := map[string]any{
		"size": batchSize,
		"sort": []map[string]string{{"@timestamp": "asc"}},
		"query": map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]string{"gt": since.Format(time.RFC3339Nano)},
			},
		},
	}
	b, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	url := cfg.IndexerBaseURL + "/wazuh-alerts-*/_search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(cfg.IndexerUsername, cfg.IndexerPassword)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("indexer search returned %d", resp.StatusCode)
	}

	var out searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decoding indexer response: %w", err)
	}
	return out.Hits.Hits, nil
}

// severityFromLevel buckets Wazuh's 0-15 rule.level into this platform's
// severity enum. This mapping is a documented heuristic (adr/0017), not an
// authoritative Wazuh convention — Wazuh itself only labels level ranges
// loosely in its own docs.
func severityFromLevel(level int) string {
	switch {
	case level >= 13:
		return "critical"
	case level >= 10:
		return "high"
	case level >= 7:
		return "medium"
	case level >= 4:
		return "low"
	default:
		return "informational"
	}
}

// confidenceFromLevel normalizes level onto [0,1] — Wazuh has no native
// confidence concept, so this is a placeholder signal, disclosed as such in
// adr/0017, not a calibrated probability.
func confidenceFromLevel(level int) float64 {
	c := float64(level) / 15.0
	if c > 1 {
		c = 1
	}
	if c < 0 {
		c = 0
	}
	return c
}

// eventClassFromGroups maps Wazuh's open-ended rule.groups taxonomy onto
// event.schema.json's closed event_class enum. Discovered against the real
// lab indexer: naively passing groups[0] through (e.g. "ossec", the
// manager's own internal-log group) is rejected by that enum, so this is a
// deliberate many-to-one mapping with "other" as the catch-all — see
// adr/0017's Consequences for the disclosed lossiness this implies (e.g.
// "web"/"firewall"/"ids" alerts all collapse into "network").
func eventClassFromGroups(groups []string) string {
	joined := strings.ToLower(strings.Join(groups, ","))
	switch {
	case strings.Contains(joined, "authentication") || strings.Contains(joined, "pam") || strings.Contains(joined, "sshd"):
		return "authentication"
	case strings.Contains(joined, "syscheck"):
		return "file"
	case strings.Contains(joined, "rootcheck") || strings.Contains(joined, "malware") || strings.Contains(joined, "rootkit") || strings.Contains(joined, "virus"):
		return "malware"
	case strings.Contains(joined, "vulnerability"):
		return "vulnerability"
	case strings.Contains(joined, "amazon") || strings.Contains(joined, "azure") || strings.Contains(joined, "gcp") || strings.Contains(joined, "office365") || strings.Contains(joined, "github"):
		return "cloud"
	case strings.Contains(joined, "mail") || strings.Contains(joined, "postfix"):
		return "email"
	case strings.Contains(joined, "web") || strings.Contains(joined, "firewall") || strings.Contains(joined, "ids") || strings.Contains(joined, "attack") || strings.Contains(joined, "recon"):
		return "network"
	case strings.Contains(joined, "process") || strings.Contains(joined, "audit_command"):
		return "process"
	default:
		return "other"
	}
}

// toIngestRequest builds the same JSON body POST /v1/events already accepts
// (docs/05-integrations/wazuh.md's mapping list) — source_event_id is the
// indexer document id so a re-polled or replayed alert naturally dedupes via
// services/api/events' existing natural-key path rather than needing this
// package to track its own dedup state.
func toIngestRequest(hitID string, src alertSource) map[string]any {
	payload := map[string]any{
		"source_event_id": hitID,
		"event_class":     eventClassFromGroups(src.Rule.Groups),
		"severity":        severityFromLevel(src.Rule.Level),
		"confidence":      confidenceFromLevel(src.Rule.Level),
		"rule": map[string]any{
			"id":          src.Rule.ID,
			"level":       src.Rule.Level,
			"description": src.Rule.Description,
		},
		"agent":    map[string]any{"id": src.Agent.ID, "name": src.Agent.Name},
		"manager":  map[string]any{"name": src.Manager.Name},
		"decoder":  map[string]any{"name": src.Decoder.Name},
		"location": src.Location,
		"full_log": src.FullLog,
	}

	if observables := extractObservables(hitID, src); len(observables) > 0 {
		payload["observables"] = observables
	}

	return map[string]any{
		"source":      "wazuh",
		"occurred_at": src.Timestamp,
		"payload":     payload,
	}
}

// extractObservables reads the handful of Wazuh "data" fields
// docs/05-integrations/wazuh.md names (source/destination IP, user) when
// present — most alert types (like a manager health log) carry none, and
// this deliberately returns an empty slice rather than guessing.
func extractObservables(hitID string, src alertSource) []map[string]any {
	var out []map[string]any
	add := func(obsType, value string) {
		if value == "" {
			return
		}
		conf := confidenceFromLevel(src.Rule.Level)
		out = append(out, map[string]any{
			"type":          obsType,
			"value":         value,
			"normalization": "wazuh-data-field",
			"first_seen":    src.Timestamp,
			"last_seen":     src.Timestamp,
			"confidence":    conf,
			"sensitivity":   "internal",
			"source_refs":   []string{hitID},
		})
	}
	add("ipv4", stringField(src.Data, "srcip"))
	add("ipv4", stringField(src.Data, "dstip"))
	add("user", stringField(src.Data, "srcuser"))
	return out
}

func stringField(data map[string]any, key string) string {
	v, _ := data[key].(string)
	return v
}

func forward(ctx context.Context, client *http.Client, ingestBaseURL, hitID string, body map[string]any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ingestBaseURL+"/v1/events", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", cmp.Or("wazuh-"+hitID, "wazuh-alert"))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("ingest returned %d", resp.StatusCode)
	}
	return nil
}
