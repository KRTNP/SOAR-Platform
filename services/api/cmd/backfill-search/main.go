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
// Not idempotent at the outbox-row level — re-running inserts a fresh
// outbox row per case/event every time, though each still lands on the
// same OpenSearch document id (idempotent at that layer). Acceptable for a
// tool meant to run rarely, if ever more than once; not worth a dedup
// mechanism.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/outbox"
)

func main() {
	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	now := time.Now().UTC()

	cases, err := backfillCases(ctx, pool, now)
	if err != nil {
		log.Fatalf("backfilling cases: %v", err)
	}
	events, err := backfillEvents(ctx, pool, now)
	if err != nil {
		log.Fatalf("backfilling events: %v", err)
	}

	fmt.Printf("wrote %d case and %d event outbox rows\n", cases, events)
}

// backfillCases mirrors cases.projectCaseBestEffort's payload shape exactly
// — a document's field set must not depend on which write path produced it.
func backfillCases(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int, error) {
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
