// Package criticaltargets implements the "privileged user or critical
// asset" concept docs/08-testing/lab-scenario-ssh-bruteforce.md's escalation
// rule needs ("Medium becomes higher priority only when source targets a
// privileged user or critical asset"). It has no relationship to
// services/api/allowlist — one says "never block this source," the other
// says "this destination deserves higher priority" — they share a CRUD
// shape (adr/0019) but are unrelated concerns.
package criticaltargets

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
)

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type CriticalTarget struct {
	ID        string
	TenantID  string
	Value     string
	Reason    string
	CreatedBy string
	CreatedAt time.Time
}

// IsCritical reports whether value (a user or host observable's value) is a
// registered privileged user or critical asset for tenantID.
func IsCritical(ctx context.Context, q querier, tenantID, value string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		select exists(select 1 from critical_targets where tenant_id = $1 and value = $2)`,
		tenantID, strings.TrimSpace(value),
	).Scan(&exists)
	return exists, err
}

// Add inserts a critical target, or updates its reason if the value already
// exists for this tenant (upsert-by-value, same as allowlist.Add).
func Add(ctx context.Context, q querier, tenantID, value, reason, createdBy string, now time.Time) (*CriticalTarget, error) {
	id := "crit-" + strings.ToLower(ulid.Make().String())
	row := q.QueryRow(ctx, `
		insert into critical_targets (id, tenant_id, value, reason, created_by, created_at)
		values ($1,$2,$3,$4,$5,$6)
		on conflict (tenant_id, value) do update set reason = excluded.reason
		returning id, tenant_id, value, reason, created_by, created_at`,
		id, tenantID, strings.TrimSpace(value), reason, createdBy, now,
	)
	var c CriticalTarget
	if err := row.Scan(&c.ID, &c.TenantID, &c.Value, &c.Reason, &c.CreatedBy, &c.CreatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// List returns every critical target for tenantID, newest first.
func List(ctx context.Context, q querier, tenantID string) ([]CriticalTarget, error) {
	rows, err := q.Query(ctx, `
		select id, tenant_id, value, reason, created_by, created_at
		from critical_targets where tenant_id = $1 order by created_at desc`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CriticalTarget
	for rows.Next() {
		var c CriticalTarget
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Value, &c.Reason, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
