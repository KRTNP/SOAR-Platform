// Command backfill-search is a one-off tool (adr/0022's disclosed gap):
// services/api/outbox only started getting written the day the outbox
// table was added, so every case/event that existed before that has no
// OpenSearch document. This reads every existing cases and events row
// directly and writes an outbox_events row for each, using the identical
// payload shapes events.Handler.Ingest and cases.projectCaseBestEffort
// already write — services/api/searchindex's already-running poller then
// projects them into OpenSearch exactly as it does any other outbox row.
//
// Run once: DATABASE_URL=postgres://... go run ./cmd/backfill-search
//
// Idempotent as a catch-up tool: it only projects cases/events that have
// never had an outbox row written for them (by this tool or by the normal
// ingest/update paths), so re-running it is a no-op once the initial
// catch-up is done. It does not dedup against the outbox table's normal
// multi-row-per-entity behavior from ordinary mutations — that's intentional.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/outbox"
	"soar-platform/services/api/secrets"
)

// main exists only to turn run's error into an exit code. The work lives in
// run so that `defer pool.Close()` actually runs on the failure paths:
// log.Fatalf calls os.Exit, which skips every pending defer, so a failed
// backfill used to abandon the pool mid-flight instead of draining it.
func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	sp, err := secrets.New()
	if err != nil {
		return fmt.Errorf("selecting secrets backend: %w", err)
	}
	dbURL, err := sp.Get(ctx, "DATABASE_URL")
	if err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	now := time.Now().UTC()

	cases, err := backfillCases(ctx, pool, now)
	if err != nil {
		return fmt.Errorf("backfilling cases: %w", err)
	}
	events, err := backfillEvents(ctx, pool, now)
	if err != nil {
		return fmt.Errorf("backfilling events: %w", err)
	}

	fmt.Printf("wrote %d case and %d event outbox rows\n", cases, events)
	return nil
}

// alreadyProjected returns the entity ids that already have at least one
// outbox_events row for the given entity type, across all tenants — used
// so re-running this tool only ever projects entities it has never touched.
func alreadyProjected(ctx context.Context, pool *pgxpool.Pool, entityType string) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `select distinct entity_id from outbox_events where entity_type = $1`, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning projected entity id: %w", err)
		}
		seen[id] = true
	}
	return seen, rows.Err()
}

// backfillCases mirrors cases.projectCaseBestEffort's payload shape exactly
// — a document's field set must not depend on which write path produced it.
func backfillCases(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int, error) {
	done, err := alreadyProjected(ctx, pool, "case")
	if err != nil {
		return 0, fmt.Errorf("checking already-projected cases: %w", err)
	}

	rows, err := pool.Query(ctx, `
		select id, tenant_id, title, category, severity, status, repeat_count, correlation_key, updated_at
		from cases`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, tenantID, title, category, severity, status string
		var repeatCount int
		var correlationKey *string
		var updatedAt time.Time
		if err := rows.Scan(&id, &tenantID, &title, &category, &severity, &status, &repeatCount, &correlationKey, &updatedAt); err != nil {
			return count, fmt.Errorf("scanning case row: %w", err)
		}
		if done[id] {
			continue
		}
		key := ""
		if correlationKey != nil {
			key = *correlationKey
		}
		payload := map[string]any{
			"id": id, "tenant_id": tenantID, "title": title, "category": category,
			"severity": severity, "status": status, "repeat_count": repeatCount,
			"correlation_key": key, "updated_at": updatedAt,
		}
		if err := outbox.Write(ctx, pool, tenantID, "case", id, payload, now); err != nil {
			return count, fmt.Errorf("writing outbox row for case %s: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}

// backfillEvents mirrors events.Handler.Ingest's payload shape exactly,
// resolving each event's case via the same alert_events/alerts join
// events/handler.go relies on (an event may have no linked case yet, or
// more than one alert — this takes the first case_ref found, matching the
// 1:1 relationship every event ingested through this codebase actually has).
func backfillEvents(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int, error) {
	done, err := alreadyProjected(ctx, pool, "event")
	if err != nil {
		return 0, fmt.Errorf("checking already-projected events: %w", err)
	}

	rows, err := pool.Query(ctx, `
		select e.id, e.tenant_id, e.source, e.event_class, e.severity, e.confidence, e.observed_at,
		       (select a.case_ref from alert_events ae join alerts a on a.id = ae.alert_id
		        where ae.event_id = e.id and a.case_ref is not null limit 1)
		from events e`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, tenantID, source, eventClass, severity string
		var confidence float64
		var observedAt time.Time
		var caseID *string
		if err := rows.Scan(&id, &tenantID, &source, &eventClass, &severity, &confidence, &observedAt, &caseID); err != nil {
			return count, fmt.Errorf("scanning event row: %w", err)
		}
		if done[id] {
			continue
		}
		payload := map[string]any{
			"id": id, "tenant_id": tenantID, "case_id": caseID, "source": source,
			"event_class": eventClass, "severity": severity, "confidence": confidence,
			"occurred_at": observedAt,
		}
		if err := outbox.Write(ctx, pool, tenantID, "event", id, payload, now); err != nil {
			return count, fmt.Errorf("writing outbox row for event %s: %w", id, err)
		}
		count++
	}
	return count, rows.Err()
}
