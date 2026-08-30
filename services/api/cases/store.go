package cases

import (
	"context"
	"errors"

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
			opened_at, updated_at, owner
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		c.ID, c.TenantID, c.Title, c.Category, c.Severity, c.Confidence, c.Status,
		c.OpenedAt, c.UpdatedAt, c.Owner,
	)
	return err
}

func LinkAlert(ctx context.Context, tx pgx.Tx, caseID, alertID string) error {
	_, err := tx.Exec(ctx, `insert into case_alerts (case_id, alert_id) values ($1,$2)`, caseID, alertID)
	return err
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
	hypothesis, facts, limitations, closure_reason, correlation_key, root_cause, lessons_learned`

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
// ADR-0009).
type ObservableRef struct {
	Type  string
	Value string
}

func GetObservables(ctx context.Context, q querier, tenantID, caseID string) ([]ObservableRef, error) {
	rows, err := q.Query(ctx, `
		select o.type, o.value
		from case_observables co
		join observables o on o.id = co.observable_id
		where co.case_id = $1 and o.tenant_id = $2
		order by o.id`, caseID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ObservableRef
	for rows.Next() {
		var ref ObservableRef
		if err := rows.Scan(&ref.Type, &ref.Value); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}
