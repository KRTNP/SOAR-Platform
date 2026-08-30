package events

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Insert writes ev inside tx. If ev's natural dedup key (tenant, source,
// source_event_id) was already used by an earlier event, no new row is
// written and duplicate=true is returned with the existing event's id — the
// caller reuses that event's alert/case instead of creating a new chain.
// observableIDs are the newly-inserted observables.id values, in the same
// order as ev.Observables, for callers that need to link them to an alert.
func Insert(ctx context.Context, tx pgx.Tx, ev *Event) (existingEventID string, duplicate bool, observableIDs []int64, err error) {
	// The insert runs inside a savepoint: a unique-violation aborts whatever
	// transaction it ran in, and pgx has no way to run further statements on
	// an aborted transaction. Rolling back to the savepoint (not the whole
	// outer tx) lets the natural-key lookup below still run in the same
	// transaction as the caller's other writes.
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return "", false, nil, err
	}
	_, err = savepoint.Exec(ctx, `
		insert into events (
			id, tenant_id, source, source_event_id, observed_at, received_at,
			event_class, severity, confidence, raw_payload, raw_ref_uri, raw_ref_sha256,
			labels, extensions, schema_version
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		ev.ID, ev.TenantID, ev.Source, ev.SourceEventID, ev.ObservedAt, ev.ReceivedAt,
		ev.EventClass, ev.Severity, ev.Confidence, ev.RawPayload, ev.RawRef.URI, ev.RawRef.SHA256,
		ev.Labels, ev.Extensions, ev.SchemaVersion,
	)
	if err != nil {
		_ = savepoint.Rollback(ctx)
		if isUniqueViolation(err, "events_tenant_source_event_uq") {
			existing, lookupErr := findByNaturalKey(ctx, tx, ev.TenantID, ev.Source, ev.SourceEventID)
			if lookupErr != nil {
				return "", false, nil, lookupErr
			}
			return existing, true, nil, nil
		}
		return "", false, nil, err
	}
	if err := savepoint.Commit(ctx); err != nil {
		return "", false, nil, err
	}

	observableIDs = make([]int64, 0, len(ev.Observables))
	for _, o := range ev.Observables {
		var obsID int64
		err = tx.QueryRow(ctx, `
			insert into observables (
				tenant_id, type, value, algorithm, normalization,
				first_seen, last_seen, confidence, sensitivity, labels
			) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			returning id`,
			ev.TenantID, o.Type, o.Value, o.Algorithm, o.Normalization,
			o.FirstSeen, o.LastSeen, o.Confidence, o.Sensitivity, o.Labels,
		).Scan(&obsID)
		if err != nil {
			return "", false, nil, err
		}
		if _, err = tx.Exec(ctx,
			`insert into event_observables (event_id, observable_id) values ($1,$2)`,
			ev.ID, obsID,
		); err != nil {
			return "", false, nil, err
		}
		for _, ref := range o.SourceRefs {
			if _, err = tx.Exec(ctx,
				`insert into observable_source_refs (observable_id, source_ref) values ($1,$2) on conflict do nothing`,
				obsID, ref,
			); err != nil {
				return "", false, nil, err
			}
		}
		observableIDs = append(observableIDs, obsID)
	}
	return ev.ID, false, observableIDs, nil
}

func findByNaturalKey(ctx context.Context, tx pgx.Tx, tenantID, source string, sourceEventID *string) (string, error) {
	var id string
	err := tx.QueryRow(ctx,
		`select id from events where tenant_id = $1 and source = $2 and source_event_id = $3`,
		tenantID, source, sourceEventID,
	).Scan(&id)
	return id, err
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && (constraint == "" || pgErr.ConstraintName == constraint)
}
