//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"

	"soar-platform/services/api/connectors"
	"soar-platform/services/api/enrichment"
	"soar-platform/services/api/ticketing"
)

// TestEnrichmentConnectorContract is M2's gate: docs/10-roadmap/milestones.md
// names "connector contract tests" as what proves the mock enrichment
// connector before it's considered "open" per
// docs/05-integrations/connector-catalog.md.
func TestEnrichmentConnectorContract(t *testing.T) {
	pool := setupPool(t)
	ctx := context.Background()

	newConnector := func(t *testing.T) connectors.Connector {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		return enrichment.New(tx)
	}

	// A fresh observable per contract-test run, seeded to match a known-bad
	// indicator from the migration's lab-only ioc_indicators seed data.
	observableID := insertObservable(t, pool, "ipv4", "198.51.100.10")

	tc := connectors.ContractTestCase{
		Valid: connectors.Invocation{
			Intent: "lookup_indicator",
			Target: "ioc_indicators",
			Params: map[string]any{"observable_id": observableID, "type": "ipv4", "value": "198.51.100.10"},
		},
		Malformed: connectors.Invocation{
			Intent: "lookup_indicator",
			Target: "ioc_indicators",
			Params: map[string]any{},
		},
		OutsideAllowlistTarget: "external-ti-provider.example.com",
		CountEffects: func(t *testing.T) int {
			t.Helper()
			var n int
			if err := pool.QueryRow(ctx, `select count(*) from enrichment_results`).Scan(&n); err != nil {
				t.Fatalf("counting enrichment_results: %v", err)
			}
			return n
		},
	}
	connectors.RunContractTests(t, newConnector, tc)
}

// TestTicketingConnectorContract is the ticketing half of the same gate.
func TestTicketingConnectorContract(t *testing.T) {
	pool := setupPool(t)
	ctx := context.Background()

	newConnector := func(t *testing.T) connectors.Connector {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		return ticketing.New(tx)
	}

	caseID := insertCase(t, pool)

	tc := connectors.ContractTestCase{
		Valid: connectors.Invocation{
			Intent: "create_ticket",
			Target: "tickets",
			Params: map[string]any{"tenant_id": "lab", "case_id": caseID, "summary": "contract test case (medium severity)"},
		},
		Malformed: connectors.Invocation{
			Intent: "create_ticket",
			Target: "tickets",
			Params: map[string]any{"tenant_id": "lab"},
		},
		OutsideAllowlistTarget: "external-ticketing-provider.example.com",
		CountEffects: func(t *testing.T) int {
			t.Helper()
			var n int
			if err := pool.QueryRow(ctx, `select count(*) from tickets`).Scan(&n); err != nil {
				t.Fatalf("counting tickets: %v", err)
			}
			return n
		},
	}
	connectors.RunContractTests(t, newConnector, tc)
}

func insertObservable(t *testing.T, pool *pgxpool.Pool, obsType, value string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		insert into observables (tenant_id, type, value, normalization, first_seen, last_seen, confidence, sensitivity)
		values ('lab', $1, $2, 'raw', now(), now(), 0.8, 'internal')
		returning id`, obsType, value,
	).Scan(&id)
	if err != nil {
		t.Fatalf("inserting observable: %v", err)
	}
	return id
}

func insertCase(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := "case-" + ulid.Make().String()
	_, err := pool.Exec(context.Background(), `
		insert into cases (id, tenant_id, title, category, severity, confidence, status, owner)
		values ($1, 'lab', 'contract test case', 'authentication', 'medium', 0.7, 'new', 'system')`, id,
	)
	if err != nil {
		t.Fatalf("inserting case: %v", err)
	}
	return id
}
