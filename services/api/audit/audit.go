// Package audit is the append-only audit log docs/06-security/audit-and-nonrepudiation.md
// requires for every state transition. Only Append and List exist — there is
// no update or delete function in this package at all, which is how
// append-only is enforced (no DB-role enforcement exists yet, the same
// disclosed gap M3 already used for RBAC). Every record carries a
// result_hash self-hash so a reader can detect a tampered individual row;
// chaining records together needs a single-writer guarantee this lab does
// not have yet and is deferred, per adr/0012-evidence-integrity-and-audit-export.md.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Record mirrors audit-and-nonrepudiation.md's field list exactly.
// TargetHash must be a hash of the affected target, never the raw value —
// see TargetHash below.
type Record struct {
	ID               string
	TenantID         string
	EventID          *string
	RequestID        *string
	Actor            string
	ActorType        string
	CaseID           *string
	ExecutionID      *string
	Action           string
	TargetHash       *string
	PolicyVersion    string
	ConnectorVersion *string
	Decision         string
	Reason           *string
	Timestamp        time.Time
	ResultHash       string
	Verified         bool // set only by List, a read-time check; never persisted
}

// querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// TargetHash hashes a target identifier so raw target values (IPs,
// observable values, ticket refs) never enter the audit trail directly.
func TargetHash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func resultHash(r Record) string {
	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	s := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		r.TenantID, deref(r.CaseID), r.Actor, r.ActorType, r.Action, deref(r.TargetHash),
		r.PolicyVersion, deref(r.ConnectorVersion), r.Decision, r.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Append writes r as a new row, computing and storing its result_hash.
// PolicyVersion defaults to "policy-1.0" (the same default actions.Action
// uses) when unset. Timestamp defaults to now (UTC, microsecond-truncated —
// the same class of Postgres-precision fix M3's nowUTC applied) when zero.
func Append(ctx context.Context, q querier, r Record) error {
	if r.PolicyVersion == "" {
		r.PolicyVersion = "policy-1.0"
	}
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now().UTC().Truncate(time.Microsecond)
	} else {
		r.Timestamp = r.Timestamp.UTC().Truncate(time.Microsecond)
	}
	r.ResultHash = resultHash(r)

	_, err := q.Exec(ctx, `
		insert into audit_log (
			tenant_id, event_id, request_id, actor, actor_type, case_id, execution_id,
			action, target_hash, policy_version, connector_version, decision, reason,
			"timestamp", result_hash
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		r.TenantID, r.EventID, r.RequestID, r.Actor, r.ActorType, r.CaseID, r.ExecutionID,
		r.Action, r.TargetHash, r.PolicyVersion, r.ConnectorVersion, r.Decision, r.Reason,
		r.Timestamp, r.ResultHash,
	)
	if err != nil {
		return fmt.Errorf("appending audit record: %w", err)
	}
	return nil
}

// List returns the tenant's audit records newest-first, each with Verified
// set from a read-time recomputation of its result_hash — a tampered row is
// visible to the caller without this package ever "fixing" it silently.
func List(ctx context.Context, q querier, tenantID string, limit int) ([]Record, error) {
	rows, err := q.Query(ctx, `
		select id, event_id, request_id, actor, actor_type, case_id, execution_id,
		       action, target_hash, policy_version, connector_version, decision, reason,
		       "timestamp", result_hash
		from audit_log where tenant_id = $1 order by "timestamp" desc, id desc limit $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing audit records: %w", err)
	}
	defer rows.Close()

	var out []Record
	for rows.Next() {
		var r Record
		var id int64
		if err := rows.Scan(
			&id, &r.EventID, &r.RequestID, &r.Actor, &r.ActorType, &r.CaseID, &r.ExecutionID,
			&r.Action, &r.TargetHash, &r.PolicyVersion, &r.ConnectorVersion, &r.Decision, &r.Reason,
			&r.Timestamp, &r.ResultHash,
		); err != nil {
			return nil, fmt.Errorf("scanning audit record: %w", err)
		}
		r.ID = strconv.FormatInt(id, 10)
		r.TenantID = tenantID
		r.Verified = resultHash(r) == r.ResultHash
		out = append(out, r)
	}
	return out, rows.Err()
}
