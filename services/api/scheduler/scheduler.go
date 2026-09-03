// Package scheduler implements M6's rollback half
// (docs/10-roadmap/milestones.md: "controlled production pilot", gate
// "owner sign-off, rollback, KPI baseline") — enforcing
// actions.block_ttl_seconds automatically instead of relying on the manual
// POST /v1/actions/{id}/release path. See adr/0014-ttl-rollback-scheduler.md
// for why this is a ticker rather than a Temporal workflow, and why no new
// approval gate or row locking guards a concurrent manual release, and
// adr/0019 for the 5/15/30s backoff retry and escalation added after that
// ADR's initial "no backoff/alerting" disclosure. KPI baseline reporting is
// a separate, later piece and is not implemented here.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/audit"
	"soar-platform/services/api/store"
	"soar-platform/services/api/telemetry"
)

const defaultBatchLimit = 100

// retrySchedule is docs/08-testing/lab-scenario-ssh-bruteforce.md's "retry
// rollback at 5, 15 and 30 seconds" literally: retrySchedule[n] is how long
// to wait before the (n+1)th attempt after the nth one failed. Once every
// entry has been tried and failed, the action is escalated (see
// escalate) rather than retried again on the scheduler's own tick cadence.
var retrySchedule = []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}

// candidateQuery finds executed actions whose block TTL has elapsed,
// excluding ones already escalated (a human must act — see escalate) and
// ones still waiting out their own retry backoff. network_blocks is not
// queried directly — its status is kept in sync by net.Release inside
// actions.Release, the same as the manual release path already relies on.
const candidateQuery = `
	select id, tenant_id, release_attempts
	from actions
	where status = 'executed'
	  and executed_at + make_interval(secs => block_ttl_seconds) <= now()
	  and escalated = false
	  and (next_retry_at is null or next_retry_at <= now())
	order by executed_at
	limit $1`

type candidate struct {
	id              string
	tenantID        string
	releaseAttempts int
}

// Sweep releases every action whose block TTL has elapsed and has not yet
// been released, auditing each as an automatic (actor="scheduler") release.
// One candidate's release failing is logged, backed off or escalated (see
// recordFailure), and skipped — not fatal to the rest of the batch; each
// release attempt runs in its own transaction. Returns a non-nil error only
// if the candidate query itself fails.
func Sweep(ctx context.Context, pool *pgxpool.Pool, net actions.NetworkConnector, logger *slog.Logger, rec *telemetry.Recorder) (released int, err error) {
	candidates, err := findCandidates(ctx, pool)
	if err != nil {
		return 0, fmt.Errorf("selecting release candidates: %w", err)
	}

	for _, c := range candidates {
		if err := releaseOne(ctx, pool, net, c); err != nil {
			logger.Error("scheduler: releasing action failed", "action_id", c.id, "tenant_id", c.tenantID, "attempt", c.releaseAttempts+1, "err", err)
			if failErr := recordFailure(ctx, pool, c, logger, rec); failErr != nil {
				logger.Error("scheduler: recording release failure failed", "action_id", c.id, "err", failErr)
			}
			continue
		}
		rec.ActionReleased("scheduled")
		released++
	}
	return released, nil
}

func findCandidates(ctx context.Context, pool *pgxpool.Pool) ([]candidate, error) {
	rows, err := pool.Query(ctx, candidateQuery, defaultBatchLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.tenantID, &c.releaseAttempts); err != nil {
			return nil, fmt.Errorf("scanning release candidate: %w", err)
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating release candidates: %w", err)
	}
	return candidates, nil
}

func releaseOne(ctx context.Context, pool *pgxpool.Pool, net actions.NetworkConnector, c candidate) error {
	return store.WithTx(ctx, pool, func(tx pgx.Tx) error {
		a, err := actions.Release(ctx, tx, c.tenantID, c.id, net)
		if err != nil {
			return err
		}
		th := audit.TargetHash(a.ID)
		cv := net.Version
		return audit.Append(ctx, tx, audit.Record{
			TenantID: c.tenantID, Actor: "scheduler", ActorType: "system",
			CaseID: &a.CaseID, ExecutionID: &a.ID, Action: "action.release",
			TargetHash: &th, ConnectorVersion: &cv, Decision: "allowed",
		})
	})
}

// recordFailure runs as its own statement against the pool, not inside
// releaseOne's already-rolled-back transaction — the same posture
// services/api/actions/handler.go's auditDeniedExecute already uses for
// exactly this reason (a write made inside a transaction that failed and
// rolled back would vanish along with it). It advances release_attempts and
// either schedules the next backoff retry or, once retrySchedule is
// exhausted, marks the action escalated and writes an audit record — the
// block is retained (this never touches network_blocks or actions.status),
// matching the scenario's "retain the block and escalate if all retries
// fail, and never remove all firewall rules."
func recordFailure(ctx context.Context, pool *pgxpool.Pool, c candidate, logger *slog.Logger, rec *telemetry.Recorder) error {
	if c.releaseAttempts >= len(retrySchedule) {
		return escalate(ctx, pool, c, logger, rec)
	}
	nextRetryAt := time.Now().UTC().Add(retrySchedule[c.releaseAttempts])
	_, err := pool.Exec(ctx, `
		update actions set release_attempts = release_attempts + 1, next_retry_at = $2
		where id = $1`, c.id, nextRetryAt)
	return err
}

func escalate(ctx context.Context, pool *pgxpool.Pool, c candidate, logger *slog.Logger, rec *telemetry.Recorder) error {
	return store.WithTx(ctx, pool, func(tx pgx.Tx) error {
		var caseID string
		if err := tx.QueryRow(ctx, `
			update actions set release_attempts = release_attempts + 1, escalated = true
			where id = $1
			returning case_id`, c.id,
		).Scan(&caseID); err != nil {
			return err
		}
		rec.ActionEscalated()
		logger.Error("scheduler: release retries exhausted, escalating", "action_id", c.id, "case_id", caseID)
		reason := fmt.Sprintf("automatic rollback failed after %d attempts (5s/15s/30s backoff exhausted); block retained, needs manual release", len(retrySchedule))
		th := audit.TargetHash(c.id)
		// audit_log.decision is a closed enum (allowed/denied/completed) —
		// "escalated" isn't a decision on whether something was allowed to
		// happen, so this uses "completed" (the escalation itself
		// completed) with the actual detail carried in Reason, same as
		// every other system-notification-shaped audit record in this
		// codebase (e.g. approval.approve/reject).
		return audit.Append(ctx, tx, audit.Record{
			TenantID: c.tenantID, Actor: "scheduler", ActorType: "system",
			CaseID: &caseID, ExecutionID: &c.id, Action: "action.release_escalated",
			TargetHash: &th, Decision: "completed", Reason: &reason,
		})
	})
}

// Run calls Sweep every interval until ctx is canceled.
func Run(ctx context.Context, pool *pgxpool.Pool, net actions.NetworkConnector, interval time.Duration, logger *slog.Logger, rec *telemetry.Recorder) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			start := time.Now()
			_, err := Sweep(ctx, pool, net, logger, rec)
			rec.SchedulerSweep(time.Since(start))
			if err != nil {
				logger.Error("scheduler: sweep failed", "err", err)
			}
		}
	}
}
