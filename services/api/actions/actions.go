// Package actions implements the M3 approval/deny/expiry gate
// (docs/06-security/safe-response-controls.md, adr/0003-human-in-the-loop.md):
// an analyst-requested action never reaches the network connector until an
// approval bound to its exact id has been recorded as approved. See
// adr/0011-approval-gated-mock-network-block.md for the scope this
// implements (single approval, lazy TTL expiry, no scheduler yet) and what
// is deferred to later milestones.
package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"

	"soar-platform/services/api/connectors"
)

// NetworkConnector is the seam between this package's approval-gate logic
// and whichever connector actually performs block_network_indicator — the
// mock services/api/network (default) or the real services/api/opnsense
// (env-gated, see adr/0013-real-opnsense-lab-connector.md). The caller
// (services/api/main.go) picks the implementation once at startup; Execute
// and Release never import a concrete connector package directly, so
// swapping providers cannot silently change approval-gate behavior.
type NetworkConnector struct {
	// New builds a fresh connectors.Connector bound to tx for one Invoke call.
	New func(tx pgx.Tx) connectors.Connector
	// Release performs the rollback for actionID/targetValue outside of the
	// approval-gate transaction (matching services/api/network.Release's
	// signature plus the target value a real, stateless connector needs).
	Release func(ctx context.Context, tx pgx.Tx, actionID, targetValue string) error
	// Target is the connectors.Invocation.Target passed to Invoke — must be
	// present in the connector's own manifest target_allowlist.
	Target string
	// Version identifies which connector actually ran, for the audit trail
	// (services/api/actions/handler.go's Execute handler).
	Version string
	// Preview answers "would target already be in effect right now", for a
	// live dry-run in services/api/actions/handler.go's Preview handler. nil
	// when the connector has no live state to check (the mock, which has no
	// standalone existence query outside of an action row) — the handler
	// falls back to its existing generic message in that case.
	Preview func(ctx context.Context, target string) (alreadyInEffect bool, err error)
}

var (
	ErrNotFound         = errors.New("action not found")
	ErrApprovalRequired = errors.New("approval is required before this action can execute")
	ErrApprovalRejected = errors.New("approval was rejected")
	ErrApprovalExpired  = errors.New("approval has expired")
	ErrAlreadyDecided   = errors.New("approval was already decided")
	ErrNotExecuted      = errors.New("action has not been executed yet, nothing to release")
)

const (
	// approvalTTL matches the 15-minute analyst review SLA in
	// docs/08-testing/lab-scenario-ssh-bruteforce.md.
	approvalTTL = 15 * time.Minute
	// blockTTLSeconds matches the 10-minute block TTL from the same scenario
	// and ADR-0007. There is no scheduler yet to enforce it automatically —
	// see Release.
	blockTTLSeconds = 600
)

type Action struct {
	ID              string
	TenantID        string
	CaseID          string
	Intent          string
	TargetType      string
	TargetValue     string
	Risk            string
	Mode            string
	RequestedBy     string
	Status          string
	IdempotencyKey  string
	PolicyVersion   string
	BlockTTLSeconds int
	ExecutedAt      *time.Time
	ReleasedAt      *time.Time
	CreatedAt       time.Time
}

type Approval struct {
	ID         string
	ActionID   string
	Status     string
	ApprovedBy *string
	Reason     *string
	DecidedAt  *time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// nowUTC truncates to microsecond precision to match what a timestamptz
// column actually stores — keeping an in-memory struct's timestamp equal to
// what a later read of the same row would return.
func nowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

// idempotencyKey follows docs/03-domain-model/action-model.md's example
// format ("case-001:block:ipv4:198.51.100.10"). "block" stands in for the
// only intent this package supports so far.
func idempotencyKey(caseID, targetType, targetValue string) string {
	return fmt.Sprintf("%s:block:%s:%s", caseID, targetType, targetValue)
}

// RequestBlock creates a block_network_indicator action plus its pending
// approval, or returns the existing pair unchanged if the same case/target
// was already requested — idempotent per action-model.md's action hash.
// created reports whether this call actually created a new action.
func RequestBlock(ctx context.Context, q querier, tenantID, caseID, targetType, targetValue, requestedBy string) (a Action, ap Approval, created bool, err error) {
	key := idempotencyKey(caseID, targetType, targetValue)
	existingA, existingAp, err := getByIdempotencyKey(ctx, q, tenantID, key)
	if err != nil {
		return Action{}, Approval{}, false, err
	}
	if existingA != nil {
		if err := expireIfStale(ctx, q, existingA, existingAp); err != nil {
			return Action{}, Approval{}, false, err
		}
		return *existingA, *existingAp, false, nil
	}

	now := nowUTC()
	a = Action{
		ID:              "act-" + strings.ToLower(ulid.Make().String()),
		TenantID:        tenantID,
		CaseID:          caseID,
		Intent:          "block_network_indicator",
		TargetType:      targetType,
		TargetValue:     targetValue,
		Risk:            "high",
		Mode:            "execute",
		RequestedBy:     requestedBy,
		Status:          "pending_approval",
		IdempotencyKey:  key,
		PolicyVersion:   DefaultPolicyVersion,
		BlockTTLSeconds: blockTTLSeconds,
		CreatedAt:       now,
	}
	if _, err := q.Exec(ctx, `
		insert into actions (
			id, tenant_id, case_id, intent, target_type, target_value, risk, mode,
			requested_by, status, idempotency_key, policy_version, block_ttl_seconds
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		a.ID, a.TenantID, a.CaseID, a.Intent, a.TargetType, a.TargetValue, a.Risk, a.Mode,
		a.RequestedBy, a.Status, a.IdempotencyKey, a.PolicyVersion, a.BlockTTLSeconds,
	); err != nil {
		return Action{}, Approval{}, false, fmt.Errorf("creating action: %w", err)
	}

	ap = Approval{
		ID:        "apr-" + strings.ToLower(ulid.Make().String()),
		ActionID:  a.ID,
		Status:    "pending",
		ExpiresAt: now.Add(approvalTTL),
		CreatedAt: now,
	}
	if _, err := q.Exec(ctx, `
		insert into approvals (id, action_id, status, expires_at) values ($1,$2,$3,$4)`,
		ap.ID, ap.ActionID, ap.Status, ap.ExpiresAt,
	); err != nil {
		return Action{}, Approval{}, false, fmt.Errorf("creating approval: %w", err)
	}
	return a, ap, true, nil
}

// expireIfStale flips a still-pending approval (and its action) to expired
// once its TTL has passed, persisting the transition immediately. There is
// no scheduler yet (Temporal isn't built), so expiry is evaluated lazily on
// every read/decide path instead of pushed by a background timer.
func expireIfStale(ctx context.Context, q querier, a *Action, ap *Approval) error {
	if ap.Status != "pending" || !time.Now().UTC().After(ap.ExpiresAt) {
		return nil
	}
	if _, err := q.Exec(ctx, `update approvals set status = 'expired' where id = $1 and status = 'pending'`, ap.ID); err != nil {
		return fmt.Errorf("expiring approval: %w", err)
	}
	if _, err := q.Exec(ctx, `update actions set status = 'expired' where id = $1 and status = 'pending_approval'`, a.ID); err != nil {
		return fmt.Errorf("expiring action: %w", err)
	}
	ap.Status = "expired"
	a.Status = "expired"
	return nil
}

const selectActionApproval = `
	select a.id, a.tenant_id, a.case_id, a.intent, a.target_type, a.target_value, a.risk, a.mode,
	       a.requested_by, a.status, a.idempotency_key, a.policy_version, a.block_ttl_seconds,
	       a.executed_at, a.released_at, a.created_at,
	       ap.id, ap.status, ap.approved_by, ap.reason, ap.decided_at, ap.expires_at, ap.created_at
	from actions a join approvals ap on ap.action_id = a.id`

func scanActionApproval(row pgx.Row) (*Action, *Approval, error) {
	var a Action
	var ap Approval
	err := row.Scan(
		&a.ID, &a.TenantID, &a.CaseID, &a.Intent, &a.TargetType, &a.TargetValue, &a.Risk, &a.Mode,
		&a.RequestedBy, &a.Status, &a.IdempotencyKey, &a.PolicyVersion, &a.BlockTTLSeconds,
		&a.ExecutedAt, &a.ReleasedAt, &a.CreatedAt,
		&ap.ID, &ap.Status, &ap.ApprovedBy, &ap.Reason, &ap.DecidedAt, &ap.ExpiresAt, &ap.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("scanning action/approval: %w", err)
	}
	ap.ActionID = a.ID
	return &a, &ap, nil
}

func getByActionID(ctx context.Context, q querier, tenantID, actionID string) (*Action, *Approval, error) {
	row := q.QueryRow(ctx, selectActionApproval+` where a.tenant_id = $1 and a.id = $2`, tenantID, actionID)
	return scanActionApproval(row)
}

func getByApprovalID(ctx context.Context, q querier, tenantID, approvalID string) (*Action, *Approval, error) {
	row := q.QueryRow(ctx, selectActionApproval+` where a.tenant_id = $1 and ap.id = $2`, tenantID, approvalID)
	return scanActionApproval(row)
}

func getByIdempotencyKey(ctx context.Context, q querier, tenantID, key string) (*Action, *Approval, error) {
	row := q.QueryRow(ctx, selectActionApproval+` where a.tenant_id = $1 and a.idempotency_key = $2`, tenantID, key)
	return scanActionApproval(row)
}

// HasLiveAction reports whether caseID has any action still "executed" (a
// block genuinely in effect, not yet released) — adr/0019's close-gate:
// docs/08-testing/lab-scenario-ssh-bruteforce.md's "Case closes only after
// rollback succeeds." A merely pending_approval/approved action hasn't
// touched real infrastructure yet, so it does not block a close (an analyst
// rejecting-and-closing a case must still be able to); rejected/expired/
// released are terminal and never block one either.
func HasLiveAction(ctx context.Context, q querier, tenantID, caseID string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		select exists(
			select 1 from actions
			where tenant_id = $1 and case_id = $2 and status = 'executed'
		)`,
		tenantID, caseID,
	).Scan(&exists)
	return exists, err
}

// Get returns an action and its approval, applying lazy expiry first.
func Get(ctx context.Context, q querier, tenantID, actionID string) (*Action, *Approval, error) {
	a, ap, err := getByActionID(ctx, q, tenantID, actionID)
	if err != nil {
		return nil, nil, err
	}
	if a == nil {
		return nil, nil, ErrNotFound
	}
	if err := expireIfStale(ctx, q, a, ap); err != nil {
		return nil, nil, err
	}
	return a, ap, nil
}

// GetLatestForCase returns the most recently requested action for a case, or
// nil if none has ever been requested — used to project Case.pending_action.
func GetLatestForCase(ctx context.Context, q querier, tenantID, caseID string) (*Action, *Approval, error) {
	row := q.QueryRow(ctx, selectActionApproval+`
		where a.tenant_id = $1 and a.case_id = $2 order by a.created_at desc limit 1`, tenantID, caseID)
	a, ap, err := scanActionApproval(row)
	if err != nil || a == nil {
		return nil, nil, err
	}
	if err := expireIfStale(ctx, q, a, ap); err != nil {
		return nil, nil, err
	}
	return a, ap, nil
}

// ApprovalView pairs an approval with its action for the list endpoint.
type ApprovalView struct {
	Approval Approval
	Action   Action
}

// ListApprovals returns every approval for tenantID, pending ones first,
// newest first within each group.
func ListApprovals(ctx context.Context, q querier, tenantID string) ([]ApprovalView, error) {
	rows, err := q.Query(ctx, selectActionApproval+`
		where a.tenant_id = $1
		order by (ap.status = 'pending') desc, ap.created_at desc`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing approvals: %w", err)
	}
	defer rows.Close()

	var out []ApprovalView
	for rows.Next() {
		a, ap, err := scanActionApproval(rows)
		if err != nil {
			return nil, err
		}
		if err := expireIfStale(ctx, q, a, ap); err != nil {
			return nil, err
		}
		out = append(out, ApprovalView{Approval: *ap, Action: *a})
	}
	return out, rows.Err()
}

// Approve is idempotent: approving an already-approved approval returns it
// unchanged rather than re-deciding or re-executing anything.
func Approve(ctx context.Context, q querier, tenantID, approvalID, approvedBy string) (Action, Approval, error) {
	a, ap, err := getByApprovalID(ctx, q, tenantID, approvalID)
	if err != nil {
		return Action{}, Approval{}, err
	}
	if a == nil {
		return Action{}, Approval{}, ErrNotFound
	}
	if err := expireIfStale(ctx, q, a, ap); err != nil {
		return Action{}, Approval{}, err
	}

	switch ap.Status {
	case "approved":
		return *a, *ap, nil
	case "rejected":
		return Action{}, Approval{}, ErrAlreadyDecided
	case "expired":
		return Action{}, Approval{}, ErrApprovalExpired
	}

	now := nowUTC()
	if _, err := q.Exec(ctx, `update approvals set status = 'approved', approved_by = $2, decided_at = $3 where id = $1`,
		ap.ID, approvedBy, now); err != nil {
		return Action{}, Approval{}, fmt.Errorf("approving: %w", err)
	}
	if _, err := q.Exec(ctx, `update actions set status = 'approved' where id = $1`, a.ID); err != nil {
		return Action{}, Approval{}, fmt.Errorf("marking action approved: %w", err)
	}
	ap.Status, ap.ApprovedBy, ap.DecidedAt = "approved", &approvedBy, &now
	a.Status = "approved"
	return *a, *ap, nil
}

// Reject is idempotent for a repeated rejection, but returns ErrAlreadyDecided
// if the approval was already approved — a decision, once approved, cannot
// be reversed via reject.
func Reject(ctx context.Context, q querier, tenantID, approvalID, reason string) (Action, Approval, error) {
	a, ap, err := getByApprovalID(ctx, q, tenantID, approvalID)
	if err != nil {
		return Action{}, Approval{}, err
	}
	if a == nil {
		return Action{}, Approval{}, ErrNotFound
	}
	if err := expireIfStale(ctx, q, a, ap); err != nil {
		return Action{}, Approval{}, err
	}

	switch ap.Status {
	case "rejected":
		return *a, *ap, nil
	case "approved":
		return Action{}, Approval{}, ErrAlreadyDecided
	case "expired":
		return Action{}, Approval{}, ErrApprovalExpired
	}

	now := nowUTC()
	if _, err := q.Exec(ctx, `update approvals set status = 'rejected', reason = $2, decided_at = $3 where id = $1`,
		ap.ID, reason, now); err != nil {
		return Action{}, Approval{}, fmt.Errorf("rejecting: %w", err)
	}
	if _, err := q.Exec(ctx, `update actions set status = 'rejected' where id = $1`, a.ID); err != nil {
		return Action{}, Approval{}, fmt.Errorf("marking action rejected: %w", err)
	}
	ap.Status, ap.Reason, ap.DecidedAt = "rejected", &reason, &now
	a.Status = "rejected"
	return *a, *ap, nil
}

// Execute is the M3 gate: the network connector is invoked only when the
// freshly-reloaded approval is "approved" — never on the caller's say-so.
// Repeating execute on an already-executed action is a no-op success (the
// underlying connector call is separately idempotent per action id). net
// selects which connector actually runs (see NetworkConnector).
func Execute(ctx context.Context, tx pgx.Tx, tenantID, actionID string, net NetworkConnector) (Action, error) {
	a, ap, err := getByActionID(ctx, tx, tenantID, actionID)
	if err != nil {
		return Action{}, err
	}
	if a == nil {
		return Action{}, ErrNotFound
	}
	if err := expireIfStale(ctx, tx, a, ap); err != nil {
		return Action{}, err
	}

	if a.Status == "executed" || a.Status == "released" {
		return *a, nil
	}

	switch ap.Status {
	case "approved":
		// proceed below
	case "rejected":
		return Action{}, ErrApprovalRejected
	case "expired":
		return Action{}, ErrApprovalExpired
	default:
		return Action{}, ErrApprovalRequired
	}

	if _, err := net.New(tx).Invoke(ctx, connectors.Invocation{
		Intent: a.Intent,
		Target: net.Target,
		Params: map[string]any{
			"tenant_id":    tenantID,
			"action_id":    a.ID,
			"target_value": a.TargetValue,
		},
		IdempotencyKey: a.IdempotencyKey,
	}); err != nil {
		return Action{}, fmt.Errorf("invoking network connector: %w", err)
	}

	now := nowUTC()
	if _, err := tx.Exec(ctx, `update actions set status = 'executed', executed_at = $2 where id = $1`, a.ID, now); err != nil {
		return Action{}, fmt.Errorf("marking action executed: %w", err)
	}
	a.Status = "executed"
	a.ExecutedAt = &now
	return *a, nil
}

// Release performs the manual early-rollback path: it is safe to call before
// a scheduler exists to do this automatically at the block's TTL (deferred
// to M5/M6 — see adr/0011). No-op-safe if already released; returns
// ErrNotExecuted if called before the action has ever been executed (the
// connector never no-op-safely "releasing" something it never applied).
// net selects which connector performs the rollback (see NetworkConnector).
func Release(ctx context.Context, tx pgx.Tx, tenantID, actionID string, net NetworkConnector) (Action, error) {
	a, _, err := getByActionID(ctx, tx, tenantID, actionID)
	if err != nil {
		return Action{}, err
	}
	if a == nil {
		return Action{}, ErrNotFound
	}
	if a.Status == "released" {
		return *a, nil
	}
	if a.Status != "executed" {
		return Action{}, ErrNotExecuted
	}
	if err := net.Release(ctx, tx, a.ID, a.TargetValue); err != nil {
		return Action{}, err
	}
	now := nowUTC()
	if _, err := tx.Exec(ctx, `update actions set status = 'released', released_at = $2 where id = $1`, a.ID, now); err != nil {
		return Action{}, fmt.Errorf("marking action released: %w", err)
	}
	a.Status = "released"
	a.ReleasedAt = &now
	return *a, nil
}
