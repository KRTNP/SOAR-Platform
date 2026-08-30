package alerts

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Insert writes al and its alert_events / alert_observables joins inside tx.
func Insert(ctx context.Context, tx pgx.Tx, al *Alert, observableIDs []int64) error {
	_, err := tx.Exec(ctx, `
		insert into alerts (
			id, tenant_id, rule_id, rule_version, detection_source, status,
			severity, confidence, correlation_key, correlation_window_seconds, case_ref, created_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		al.ID, al.TenantID, al.RuleID, al.RuleVersion, al.DetectionSource, al.Status,
		al.Severity, al.Confidence, al.CorrelationKey, al.CorrelationWindowSeconds, al.CaseRef, al.CreatedAt,
	)
	if err != nil {
		return err
	}
	for _, eventID := range al.EventIDs {
		if _, err := tx.Exec(ctx,
			`insert into alert_events (alert_id, event_id) values ($1,$2)`,
			al.ID, eventID,
		); err != nil {
			return err
		}
	}
	for _, obsID := range observableIDs {
		if _, err := tx.Exec(ctx,
			`insert into alert_observables (alert_id, observable_id) values ($1,$2)`,
			al.ID, obsID,
		); err != nil {
			return err
		}
	}
	return nil
}

// SetCaseRef records which case an alert belongs to, once that case exists.
func SetCaseRef(ctx context.Context, tx pgx.Tx, alertID, caseID string) error {
	_, err := tx.Exec(ctx, `update alerts set case_ref = $1 where id = $2`, caseID, alertID)
	return err
}

// FindByEventID returns the alert already linked to eventID, used on the
// duplicate-event path so a retried ingest returns the existing alert_id
// instead of creating a second alert for the same underlying event.
func FindByEventID(ctx context.Context, tx pgx.Tx, eventID string) (alertID string, caseRef *string, err error) {
	err = tx.QueryRow(ctx, `
		select a.id, a.case_ref
		from alerts a
		join alert_events ae on ae.alert_id = a.id
		where ae.event_id = $1
		limit 1`, eventID,
	).Scan(&alertID, &caseRef)
	return alertID, caseRef, err
}
