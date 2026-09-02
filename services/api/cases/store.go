package cases

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func Insert(ctx context.Context, tx pgx.Tx, c *Case) error {
	_, err := tx.Exec(ctx, `
		insert into cases (
			id, tenant_id, title, category, severity, confidence, status,
			opened_at, updated_at, owner, correlation_key, correlation_window_seconds, repeat_count
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		c.ID, c.TenantID, c.Title, c.Category, c.Severity, c.Confidence, c.Status,
		c.OpenedAt, c.UpdatedAt, c.Owner, c.CorrelationKey, c.CorrelationWindowSeconds, c.RepeatCount,
	)
	return err
}

// FindOpenByCorrelation returns the most recently opened, not-yet-closed
// case sharing correlationKey whose OpenedAt is still within windowSeconds
// of now — the read side of adr/0018's correlation slice. nil, nil (not
// ErrNotFound) when nothing matches, since "no existing case to attach to"
// is the normal, expected outcome for most calls, not an error.
func FindOpenByCorrelation(ctx context.Context, q querier, tenantID, correlationKey string, windowSeconds int, now time.Time) (*Case, error) {
	row := q.QueryRow(ctx, `
		select `+caseColumns+`
		from cases
		where tenant_id = $1
		  and correlation_key = $2
		  and status <> 'closed'
		  and opened_at + make_interval(secs => $3) >= $4
		order by opened_at desc
		limit 1`,
		tenantID, correlationKey, windowSeconds, now,
	)
	c, err := scanCase(row)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return c, err
}

// IncrementRepeatCount bumps repeat_count on an existing case that a new
// correlated alert just attached to, returning the count after the
// increment — what the SSH brute-force auto-trigger threshold check reads.
func IncrementRepeatCount(ctx context.Context, tx pgx.Tx, tenantID, caseID string, updatedAt time.Time) (int, error) {
	var count int
	err := tx.QueryRow(ctx, `
		update cases set repeat_count = repeat_count + 1, updated_at = $3
		where tenant_id = $1 and id = $2
		returning repeat_count`,
		tenantID, caseID, updatedAt,
	).Scan(&count)
	return count, err
}

func LinkAlert(ctx context.Context, tx pgx.Tx, caseID, alertID string) error {
	_, err := tx.Exec(ctx, `insert into case_alerts (case_id, alert_id) values ($1,$2)`, caseID, alertID)
	return err
}

// ExistingObservableValues returns "type:value" for every observable
// already linked to caseID — adr/0018's correlation attaches a new alert's
// observables onto an existing case, and without this check a case that
// correlates N events ends up with N near-identical observable rows all
// showing the same IP (services/api/events' Insert has no dedup of its
// own; this was invisible before correlation existed, since a case could
// only ever have one contributing event). The caller filters observableIDs
// against this before calling LinkObservables — the observable rows
// themselves still exist (harmless, just unlinked from this case), only
// the duplicate case_observables link and duplicate render (e.g. web's
// per-observable "Request block" button) are avoided.
func ExistingObservableValues(ctx context.Context, q querier, caseID string) (map[string]bool, error) {
	rows, err := q.Query(ctx, `
		select o.type, o.value
		from case_observables co join observables o on o.id = co.observable_id
		where co.case_id = $1`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var t, v string
		if err := rows.Scan(&t, &v); err != nil {
			return nil, err
		}
		out[t+":"+v] = true
	}
	return out, rows.Err()
}

func LinkObservables(ctx context.Context, tx pgx.Tx, caseID string, observableIDs []int64) error {
	for _, obsID := range observableIDs {
		if _, err := tx.Exec(ctx,
			`insert into case_observables (case_id, observable_id) values ($1,$2)`,
			caseID, obsID,
		); err != nil {
			return err
		}
	}
	return nil
}

func AppendTimeline(ctx context.Context, tx pgx.Tx, caseID, tenantID, kind, actor string, detail map[string]any) error {
	_, err := tx.Exec(ctx, `
		insert into case_timeline_entries (case_id, tenant_id, kind, actor, detail)
		values ($1,$2,$3,$4,$5)`,
		caseID, tenantID, kind, actor, detail,
	)
	return err
}

func scanCase(row pgx.Row) (*Case, error) {
	var c Case
	err := row.Scan(
		&c.ID, &c.TenantID, &c.Title, &c.Category, &c.Severity, &c.Confidence, &c.Status,
		&c.OpenedAt, &c.UpdatedAt, &c.ClosedAt, &c.Owner, &c.IncidentCommander, &c.ServiceOwner,
		&c.Hypothesis, &c.Facts, &c.Limitations, &c.ClosureReason,
		&c.CorrelationKey, &c.RootCause, &c.LessonsLearned,
		&c.CorrelationWindowSeconds, &c.RepeatCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

const caseColumns = `
	id, tenant_id, title, category, severity, confidence, status,
	opened_at, updated_at, closed_at, owner, incident_commander, service_owner,
	hypothesis, facts, limitations, closure_reason, correlation_key, root_cause, lessons_learned,
	correlation_window_seconds, repeat_count`

func Get(ctx context.Context, q querier, tenantID, caseID string) (*Case, error) {
	row := q.QueryRow(ctx, `select `+caseColumns+` from cases where tenant_id = $1 and id = $2`, tenantID, caseID)
	return scanCase(row)
}

// List returns up to limit cases for tenantID older than cursor (a case id),
// newest first. IDs are ULIDs, so id order is chronological.
func List(ctx context.Context, q querier, tenantID, cursor string, limit int) ([]Case, string, error) {
	var rows pgx.Rows
	var err error
	base := `select ` + caseColumns + ` from cases where tenant_id = $1`
	if cursor == "" {
		rows, err = q.Query(ctx, base+` order by id desc limit $2`, tenantID, limit+1)
	} else {
		rows, err = q.Query(ctx, base+` and id < $2 order by id desc limit $3`, tenantID, cursor, limit+1)
	}
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var out []Case
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(out) > limit {
		nextCursor = out[limit-1].ID
		out = out[:limit]
	}
	return out, nextCursor, nil
}

// UpdateFields applies the "update" PATCH operation: title and/or owner.
func UpdateFields(ctx context.Context, tx pgx.Tx, tenantID, caseID string, title, ownerID *string) error {
	tag, err := tx.Exec(ctx, `
		update cases set
			title = coalesce($3, title),
			owner = coalesce($4, owner),
			updated_at = now()
		where tenant_id = $1 and id = $2`,
		tenantID, caseID, title, ownerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Close applies the "close" PATCH operation: status -> closed, closed_at set,
// and the reason recorded permanently (unlike a transition's reason, which
// only lives in the timeline entry).
func Close(ctx context.Context, tx pgx.Tx, tenantID, caseID, closureReason string) error {
	tag, err := tx.Exec(ctx, `
		update cases set status = 'closed', closure_reason = $3, closed_at = now(), updated_at = now()
		where tenant_id = $1 and id = $2`,
		tenantID, caseID, closureReason,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Transition applies the "transition" PATCH operation: a status change.
// Transition applies a non-terminal status change. Reaching "closed" must go
// through Close, which enforces a mandatory closure_reason — see
// Handler.Update's rejection of transition-to-closed.
func Transition(ctx context.Context, tx pgx.Tx, tenantID, caseID, newStatus string) error {
	tag, err := tx.Exec(ctx, `
		update cases set status = $3, updated_at = now()
		where tenant_id = $1 and id = $2`,
		tenantID, caseID, newStatus,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func GetTimeline(ctx context.Context, q querier, tenantID, caseID string) ([]TimelineEntry, error) {
	if _, err := Get(ctx, q, tenantID, caseID); err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `
		select id, occurred_at, kind, actor, detail
		from case_timeline_entries
		where tenant_id = $1 and case_id = $2
		order by occurred_at asc, id asc`, tenantID, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TimelineEntry
	for rows.Next() {
		var e TimelineEntry
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Kind, &e.Actor, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ObservableRef is the minimal projection a case detail view needs — just
// enough to render an identifying chip, not the full observable record (see
// ADR-0009), plus the M2 mock-TI verdict when one has been recorded.
type ObservableRef struct {
	Type              string
	Value             string
	Verdict           *string
	TiConfidence      *float64
	SourceReliability *string
}

func GetObservables(ctx context.Context, q querier, tenantID, caseID string) ([]ObservableRef, error) {
	rows, err := q.Query(ctx, `
		select o.type, o.value, er.verdict, er.confidence, er.source_reliability
		from case_observables co
		join observables o on o.id = co.observable_id
		left join enrichment_results er on er.observable_id = o.id
		where co.case_id = $1 and o.tenant_id = $2
		order by o.id`, caseID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ObservableRef
	for rows.Next() {
		var ref ObservableRef
		if err := rows.Scan(&ref.Type, &ref.Value, &ref.Verdict, &ref.TiConfidence, &ref.SourceReliability); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}
