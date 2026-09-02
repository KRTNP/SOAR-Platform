// Package scheduler implements M6's rollback half
// (docs/10-roadmap/milestones.md: "controlled production pilot", gate
// "owner sign-off, rollback, KPI baseline") — enforcing
// actions.block_ttl_seconds automatically instead of relying on the manual
// POST /v1/actions/{id}/release path. See adr/0014-ttl-rollback-scheduler.md
// for why this is a ticker rather than a Temporal workflow, and why no new
// approval gate or row locking guards a concurrent manual release. KPI
// baseline reporting is a separate, later piece and is not implemented here.
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
)

const defaultBatchLimit = 100

// candidateQuery finds executed actions whose block TTL has elapsed.
// network_blocks is not queried directly — its status is kept in sync by
// net.Release inside actions.Release, the same as the manual release path
// already relies on.
const candidateQuery = `
	select id, tenant_id
	from actions
	where status = 'executed'
	  and executed_at + make_interval(secs => block_ttl_seconds) <= now()
	order by executed_at
	limit $1`

type candidate struct {
	id       string
	tenantID string
}

// Sweep releases every action whose block TTL has elapsed and has not yet
// been released, auditing each as an automatic (actor="scheduler") release.
// One candidate's release failing is logged and skipped, not fatal to the
// rest of the batch — each release runs in its own transaction. Returns a
// non-nil error only if the candidate query itself fails.
func Sweep(ctx context.Context, pool *pgxpool.Pool, net actions.NetworkConnector, logger *slog.Logger) (released int, err error) {
	candidates, err := findCandidates(ctx, pool)
	if err != nil {
		return 0, fmt.Errorf("selecting release candidates: %w", err)
	}

	for _, c := range candidates {
		if err := releaseOne(ctx, pool, net, c); err != nil {
			logger.Error("scheduler: releasing action failed", "action_id", c.id, "tenant_id", c.tenantID, "err", err)
			continue
		}
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
		if err := rows.Scan(&c.id, &c.tenantID); err != nil {
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

// Run calls Sweep every interval until ctx is canceled.
func Run(ctx context.Context, pool *pgxpool.Pool, net actions.NetworkConnector, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := Sweep(ctx, pool, net, logger); err != nil {
				logger.Error("scheduler: sweep failed", "err", err)
			}
		}
	}
}
