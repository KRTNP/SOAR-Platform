//go:build integration

// Package integration holds the M1 gate: "ingestion/replay tests" per
// docs/08-testing/replay-testing.md, run against a real Postgres via
// Testcontainers rather than a mock.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/audit"
	"soar-platform/services/api/cases"
	"soar-platform/services/api/connectors"
	"soar-platform/services/api/events"
	"soar-platform/services/api/evidence"
	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/network"
)

// mockNetworkConnector is the same mock-connector wiring services/api/main.go
// defaults to — tests exercise the approval gate against it, never the real
// services/api/opnsense connector (see adr/0013-real-opnsense-lab-connector.md).
func mockNetworkConnector() actions.NetworkConnector {
	return actions.NetworkConnector{
		New: func(tx pgx.Tx) connectors.Connector { return network.New(tx) },
		Release: func(ctx context.Context, tx pgx.Tx, actionID, _ string) error {
			return network.Release(ctx, tx, actionID)
		},
		Target:  "network_blocks",
		Version: "connector-mock-network-block@1.0.0",
	}
}

// setupPool starts a real Postgres container and applies every migration,
// for tests that only need direct DB access (no HTTP server).
func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"docker.io/library/postgres:17-alpine",
		tcpostgres.WithDatabase("soar_test"),
		tcpostgres.WithUsername("soar"),
		tcpostgres.WithPassword("soar_test_only"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pgContainer.Terminate(context.Background()) })

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	migrationsDir := filepath.Join("..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("reading migrations dir: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
		if err != nil {
			t.Fatalf("reading migration %s: %v", e.Name(), err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("applying migration %s: %v", e.Name(), err)
		}
	}
	return pool
}

// setupServer starts a real Postgres container, applies every migration,
// and returns an httptest server wired to it.
func setupServer(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()
	pool := setupPool(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpx.RequestIDMiddleware(), httpx.TenantStubMiddleware("lab"))
	eventsH := events.NewHandler(pool)
	casesH := cases.NewHandler(pool)
	actionsH := actions.NewHandler(pool, mockNetworkConnector())
	evidenceH := evidence.NewHandler(pool)
	auditH := audit.NewHandler(pool)
	v1 := r.Group("/v1")
	v1.POST("/events", eventsH.Ingest)
	v1.GET("/events", eventsH.List)
	v1.POST("/cases", casesH.Create)
	v1.GET("/cases", casesH.List)
	v1.GET("/cases/:caseId", casesH.Get)
	v1.PATCH("/cases/:caseId", casesH.Update)
	v1.GET("/cases/:caseId/timeline", casesH.Timeline)
	v1.GET("/cases/:caseId/evidence", casesH.Evidence)
	v1.POST("/cases/:caseId/export", casesH.Export)
	v1.POST("/actions/preview", actionsH.Preview)
	v1.GET("/actions/:actionId", actionsH.Get)
	v1.POST("/actions/:actionId/execute", actionsH.Execute)
	v1.GET("/approvals", actionsH.ListApprovals)
	v1.POST("/approvals/:approvalId/approve", actionsH.Approve)
	v1.POST("/approvals/:approvalId/reject", actionsH.Reject)
	v1.POST("/evidence/manifests", evidenceH.CreateManifest)
	v1.GET("/evidence/:evidenceId", evidenceH.Get)
	v1.GET("/audit", auditH.List)

	return httptest.NewServer(r), pool
}

func postEvent(t *testing.T, srv *httptest.Server, idempotencyKey string, body []byte) (int, genapi.AcceptedEvent) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/events", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("posting event: %v", err)
	}
	defer resp.Body.Close()

	var accepted genapi.AcceptedEvent
	_ = json.NewDecoder(resp.Body).Decode(&accepted)
	return resp.StatusCode, accepted
}

func countRows(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "select count(*) from "+table).Scan(&n); err != nil {
		t.Fatalf("counting %s: %v", table, err)
	}
	return n
}

func fixture(t *testing.T, sourceEventID string) []byte {
	t.Helper()
	payload := map[string]any{
		"source_event_id": sourceEventID,
		"event_class":     "authentication",
		"severity":        "medium",
		"confidence":      0.82,
	}
	body := map[string]any{
		"source":      "wazuh",
		"occurred_at": "2026-08-30T04:00:00Z",
		"payload":     payload,
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling fixture: %v", err)
	}
	return b
}

// TestIngestion_IdempotencyKeyReplay: the same Idempotency-Key twice returns
// accepted then duplicate, with identical event_id/alert_id, and exactly one
// row lands in events/alerts/cases/case_alerts.
func TestIngestion_IdempotencyKeyReplay(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()
	body := fixture(t, "replay-key-test-0001")

	status1, first := postEvent(t, srv, "idem-key-aaaa1111", body)
	if status1 != http.StatusAccepted || first.Status != genapi.AcceptedEventStatusAccepted {
		t.Fatalf("first post: status=%d body=%+v", status1, first)
	}

	status2, second := postEvent(t, srv, "idem-key-aaaa1111", body)
	if status2 != http.StatusAccepted || second.Status != genapi.AcceptedEventStatusDuplicate {
		t.Fatalf("second post: status=%d body=%+v", status2, second)
	}
	if first.EventId != second.EventId || *first.AlertId != *second.AlertId {
		t.Fatalf("event_id/alert_id changed on replay: first=%+v second=%+v", first, second)
	}

	for _, table := range []string{"events", "alerts", "cases", "case_alerts"} {
		if n := countRows(t, pool, table); n != 1 {
			t.Errorf("table %s: expected exactly 1 row, got %d", table, n)
		}
	}
}

// TestIngestion_NaturalKeyDedup: a different Idempotency-Key but the same
// underlying source_event_id is still treated as a duplicate.
func TestIngestion_NaturalKeyDedup(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()
	body := fixture(t, "replay-key-test-0002")

	status1, first := postEvent(t, srv, "idem-key-bbbb1111", body)
	if status1 != http.StatusAccepted || first.Status != genapi.AcceptedEventStatusAccepted {
		t.Fatalf("first post: status=%d body=%+v", status1, first)
	}

	status2, second := postEvent(t, srv, "idem-key-bbbb2222-different", body)
	if status2 != http.StatusAccepted || second.Status != genapi.AcceptedEventStatusDuplicate {
		t.Fatalf("second post (different key, same source_event_id): status=%d body=%+v", status2, second)
	}
	if first.EventId != second.EventId || *first.AlertId != *second.AlertId {
		t.Fatalf("event_id/alert_id changed on natural-key dedup: first=%+v second=%+v", first, second)
	}
	if n := countRows(t, pool, "events"); n != 1 {
		t.Errorf("expected exactly 1 event row, got %d", n)
	}
}

// TestIngestion_IdempotencyKeyReuseDifferentBody: reusing a key with a
// different body is rejected, never silently accepted.
func TestIngestion_IdempotencyKeyReuseDifferentBody(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	status1, _ := postEvent(t, srv, "idem-key-cccc1111", fixture(t, "replay-key-test-0003"))
	if status1 != http.StatusAccepted {
		t.Fatalf("first post: status=%d", status1)
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/events", strings.NewReader(string(fixture(t, "replay-key-test-0004"))))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-key-cccc1111")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("posting event: %v", err)
	}
	defer resp.Body.Close()

	var problem genapi.Problem
	_ = json.NewDecoder(resp.Body).Decode(&problem)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d (problem=%+v)", resp.StatusCode, problem)
	}
	if problem.Code != "idempotency_key_reuse" {
		t.Errorf("expected code=idempotency_key_reuse, got %q", problem.Code)
	}
}

// TestIngestion_FixtureReplay: replays the repo's own wazuh-alert.json-shaped
// payload and checks the normalized event lands with canonical fields.
func TestIngestion_FixtureReplay(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	payload := map[string]any{
		"source_event_id": "wazuh-alert-0001",
		"event_class":     "authentication",
		"severity":        "medium",
		"confidence":      0.82,
		"rule":            map[string]any{"id": "5710", "level": 10, "description": "Multiple authentication failures"},
		"agent":           map[string]any{"id": "001", "name": "lab-endpoint-01"},
		"observables": []any{
			map[string]any{
				"type": "ipv4", "value": "198.51.100.10", "normalization": "rfc5737-test-net",
				"first_seen": "2026-08-30T04:00:00Z", "last_seen": "2026-08-30T04:00:00Z",
				"confidence": 0.82, "sensitivity": "internal", "source_refs": []any{"wazuh-alert-0001"},
			},
			map[string]any{
				"type": "host", "value": "lab-endpoint-01", "normalization": "lowercase-hostname",
				"first_seen": "2026-08-30T04:00:00Z", "last_seen": "2026-08-30T04:00:00Z",
				"confidence": 1, "sensitivity": "internal", "source_refs": []any{"wazuh-alert-0001"},
			},
		},
	}
	body, _ := json.Marshal(map[string]any{"source": "wazuh", "occurred_at": "2026-08-30T04:00:00Z", "payload": payload})

	status, accepted := postEvent(t, srv, "idem-key-fixture-0001", body)
	if status != http.StatusAccepted || accepted.Status != genapi.AcceptedEventStatusAccepted {
		t.Fatalf("fixture replay: status=%d body=%+v", status, accepted)
	}

	var eventClass, severity string
	var confidence float64
	err := pool.QueryRow(context.Background(),
		"select event_class, severity, confidence from events where id = $1", accepted.EventId,
	).Scan(&eventClass, &severity, &confidence)
	if err != nil {
		t.Fatalf("reading back normalized event: %v", err)
	}
	if eventClass != "authentication" || severity != "medium" || confidence != 0.82 {
		t.Errorf("normalized fields mismatch: class=%s severity=%s confidence=%v", eventClass, severity, confidence)
	}
	if n := countRows(t, pool, "observables"); n != 2 {
		t.Errorf("expected 2 observables, got %d", n)
	}
}

func init() {
	// Fail fast with a clear message rather than a long testcontainers
	// timeout if no container runtime is reachable.
	if os.Getenv("DOCKER_HOST") == "" && os.Getenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE") == "" {
		fmt.Println("warning: DOCKER_HOST is not set; relying on the default docker/podman socket")
	}
}
