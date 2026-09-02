// Package allowlist implements the trusted/management source list
// docs/08-testing/lab-scenario-ssh-bruteforce.md requires ("Trusted sources:
// Management/admin allowlist cannot be blocked; allowlist changes are
// audited and expire") — the negative-test guard for adr/0018's SSH
// brute-force auto-trigger. It has no relationship to approval or
// connector logic; IsTrusted is a plain read the auto-trigger checks before
// ever calling actions.RequestBlock.
package allowlist

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
)

// querier is satisfied by both *pgxpool.Pool and pgx.Tx, matching every
// other package in this codebase (e.g. services/api/actions.querier).
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type TrustedSource struct {
	ID        string
	TenantID  string
	Value     string
	Reason    string
	CreatedBy string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

// IsTrusted reports whether value (e.g. a source IP) is currently on the
// allowlist for tenantID — expired entries (expires_at in the past) count
// as not trusted, matching the scenario's "allowlist changes ... expire".
func IsTrusted(ctx context.Context, q querier, tenantID, value string, now time.Time) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		select exists(
			select 1 from trusted_sources
			where tenant_id = $1 and value = $2
			  and (expires_at is null or expires_at > $3)
		)`,
		tenantID, strings.TrimSpace(value), now,
	).Scan(&exists)
	return exists, err
}

// Add inserts a trusted source, or updates reason/expires_at if the value
// already exists for this tenant (the unique index on (tenant_id, value) is
// what makes this an upsert-by-value rather than accumulating duplicates for
// the same repeatedly-re-added source).
func Add(ctx context.Context, q querier, tenantID, value, reason, createdBy string, expiresAt *time.Time, now time.Time) (*TrustedSource, error) {
	id := "trust-" + strings.ToLower(ulid.Make().String())
	row := q.QueryRow(ctx, `
		insert into trusted_sources (id, tenant_id, value, reason, created_by, created_at, expires_at)
		values ($1,$2,$3,$4,$5,$6,$7)
		on conflict (tenant_id, value) do update
			set reason = excluded.reason, expires_at = excluded.expires_at
		returning id, tenant_id, value, reason, created_by, created_at, expires_at`,
		id, tenantID, strings.TrimSpace(value), reason, createdBy, now, expiresAt,
	)
	var t TrustedSource
	if err := row.Scan(&t.ID, &t.TenantID, &t.Value, &t.Reason, &t.CreatedBy, &t.CreatedAt, &t.ExpiresAt); err != nil {
		return nil, err
	}
	return &t, nil
}

// List returns every trusted source for tenantID, newest first — lab scale
// only, no pagination (matching services/api/audit.List's own posture).
func List(ctx context.Context, q querier, tenantID string) ([]TrustedSource, error) {
	rows, err := q.Query(ctx, `
		select id, tenant_id, value, reason, created_by, created_at, expires_at
		from trusted_sources where tenant_id = $1 order by created_at desc`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TrustedSource
	for rows.Next() {
		var t TrustedSource
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Value, &t.Reason, &t.CreatedBy, &t.CreatedAt, &t.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
