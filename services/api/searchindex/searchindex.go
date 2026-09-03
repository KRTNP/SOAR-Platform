// Package searchindex projects services/api/outbox rows into OpenSearch,
// implementing the "indexer worker" half of
// docs/02-architecture/data-architecture.md's already-decided
// "transactional outbox + indexer worker" design (adr/0022). It is a plain
// HTTP client against OpenSearch's document API (PUT
// {index}/_doc/{entity_id}) — no SDK dependency, matching this codebase's
// existing posture for wazuhpoll/opnsense (raw HTTP, not a vendor client
// library).
package searchindex

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is loaded from OPENSEARCH_* environment variables only, mirroring
// services/api/wazuhpoll's and services/api/opnsense's convention.
type Config struct {
	BaseURL             string
	InsecureSkipVerify  bool
	PollIntervalSeconds int
}

// Enabled is the master switch — default false, same posture ADR-0013/0017
// use for their own real-integration toggles.
func Enabled() bool {
	return os.Getenv("OPENSEARCH_ENABLE_REAL") == "true"
}

// LoadConfig reads OPENSEARCH_BASE_URL (required), plus optional
// OPENSEARCH_INSECURE_SKIP_VERIFY (default false) and
// SEARCHINDEX_POLL_INTERVAL_SECONDS (default 5 — outbox rows should reach
// search quickly, unlike wazuhpoll's slower external-alert cadence).
func LoadConfig() (Config, error) {
	baseURL := os.Getenv("OPENSEARCH_BASE_URL")
	if baseURL == "" {
		return Config{}, fmt.Errorf("OPENSEARCH_BASE_URL is required")
	}
	interval := 5
	if v := os.Getenv("SEARCHINDEX_POLL_INTERVAL_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("SEARCHINDEX_POLL_INTERVAL_SECONDS must be a positive integer: %q", v)
		}
		interval = n
	}
	return Config{
		BaseURL:             baseURL,
		InsecureSkipVerify:  os.Getenv("OPENSEARCH_INSECURE_SKIP_VERIFY") == "true",
		PollIntervalSeconds: interval,
	}, nil
}

// batchSize matches services/api/scheduler's and wazuhpoll's own hardcoded
// candidate limits.
const batchSize = 100

type row struct {
	id         string
	entityType string
	entityID   string
	payload    json.RawMessage
}

// indexNameFor maps an outbox entity_type onto
// data-architecture.md's versioned index names ("soar-<type>s-v1"). Only
// "case" and "event" are written today (see events.Handler.Ingest) —
// "alert"/"observable"/"audit" are named here for when a future write path
// starts producing them, not because anything does yet.
func indexNameFor(entityType string) string {
	return "soar-" + entityType + "s-v1"
}

// Run polls on a ticker until ctx is cancelled, logging and continuing past
// any single sweep's error — matching services/api/scheduler.Run's and
// wazuhpoll.Run's posture.
func Run(ctx context.Context, pool *pgxpool.Pool, cfg Config, logger *slog.Logger) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}, //nolint:gosec // lab-only, explicit opt-in
		},
	}
	ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			indexed, err := Sweep(ctx, pool, client, cfg.BaseURL, logger)
			if err != nil {
				logger.Error("searchindex: sweep failed", "err", err)
				continue
			}
			if indexed > 0 {
				logger.Info("searchindex: projected outbox rows", "count", indexed)
			}
		}
	}
}

// Sweep indexes every unindexed outbox_events row (up to batchSize),
// marking each indexed_at on success. One row's failure is logged and
// skipped — not fatal to the rest of the batch, retried on the next sweep.
func Sweep(ctx context.Context, pool *pgxpool.Pool, client *http.Client, baseURL string, logger *slog.Logger) (indexed int, err error) {
	rows, err := findUnindexed(ctx, pool)
	if err != nil {
		return 0, fmt.Errorf("selecting unindexed outbox rows: %w", err)
	}

	for _, r := range rows {
		if err := putDocument(ctx, client, baseURL, r); err != nil {
			logger.Error("searchindex: indexing outbox row failed", "id", r.id, "entity_type", r.entityType, "entity_id", r.entityID, "err", err)
			continue
		}
		if _, err := pool.Exec(ctx, `update outbox_events set indexed_at = now() where id = $1`, r.id); err != nil {
			logger.Error("searchindex: marking outbox row indexed failed", "id", r.id, "err", err)
			continue
		}
		indexed++
	}
	return indexed, nil
}

func findUnindexed(ctx context.Context, pool *pgxpool.Pool) ([]row, error) {
	dbRows, err := pool.Query(ctx, `
		select id, entity_type, entity_id, payload
		from outbox_events
		where indexed_at is null
		order by created_at
		limit $1`, batchSize)
	if err != nil {
		return nil, err
	}
	defer dbRows.Close()

	var out []row
	for dbRows.Next() {
		var r row
		if err := dbRows.Scan(&r.id, &r.entityType, &r.entityID, &r.payload); err != nil {
			return nil, fmt.Errorf("scanning outbox row: %w", err)
		}
		out = append(out, r)
	}
	return out, dbRows.Err()
}

func putDocument(ctx context.Context, client *http.Client, baseURL string, r row) error {
	url := fmt.Sprintf("%s/%s/_doc/%s", baseURL, indexNameFor(r.entityType), r.entityID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(r.payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("opensearch returned %d", resp.StatusCode)
	}
	return nil
}
