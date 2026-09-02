// Package metrics computes M6's "KPI baseline" gate
// (docs/10-roadmap/milestones.md, docs/07-operations/metrics-and-kpis.md)
// directly from data already recorded by audit_log/actions/cases/evidence —
// no new tables, no time-series store. See adr/0015-kpi-snapshot.md for
// which of metrics-and-kpis.md's full list this does and doesn't cover.
package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/httpx"
)

// ConnectorOutcome is one connector_version's execute allow/deny tally.
type ConnectorOutcome struct {
	ConnectorVersion string `json:"connector_version"`
	Allowed          int    `json:"allowed"`
	Denied           int    `json:"denied"`
}

// Snapshot is a point-in-time read of every KPI this system can currently
// compute from its own recorded data. It is not a time-series or a
// historical baseline — just "what the numbers say right now".
type Snapshot struct {
	ComputedAt                   time.Time          `json:"computed_at"`
	ConnectorReliability         []ConnectorOutcome `json:"connector_reliability"`
	AnalystOverrideCount         int                `json:"analyst_override_count"`
	AutoReleaseCount             int                `json:"auto_release_count"`
	ManualReleaseCount           int                `json:"manual_release_count"`
	MeanTimeToContainmentSeconds *float64           `json:"mean_time_to_containment_seconds,omitempty"`
	CasesWithVerifiedEvidence    int                `json:"cases_with_verified_evidence"`
	TotalCases                   int                `json:"total_cases"`
}

// Compute reads audit_log, actions and cases/evidence for tenantID and
// returns a Snapshot. Every query is a simple aggregate over whatever this
// tenant has recorded so far — with no usage yet, most fields are just 0.
func Compute(ctx context.Context, pool *pgxpool.Pool, tenantID string) (Snapshot, error) {
	s := Snapshot{ComputedAt: time.Now().UTC()}

	rows, err := pool.Query(ctx, `
		select connector_version, decision, count(*)
		from audit_log
		where tenant_id = $1 and action = 'action.execute' and connector_version is not null
		group by connector_version, decision`, tenantID)
	if err != nil {
		return Snapshot{}, err
	}
	outcomes := map[string]*ConnectorOutcome{}
	for rows.Next() {
		var version, decision string
		var count int
		if err := rows.Scan(&version, &decision, &count); err != nil {
			rows.Close()
			return Snapshot{}, err
		}
		o, ok := outcomes[version]
		if !ok {
			o = &ConnectorOutcome{ConnectorVersion: version}
			outcomes[version] = o
		}
		switch decision {
		case "allowed":
			o.Allowed = count
		case "denied":
			o.Denied = count
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Snapshot{}, err
	}
	rows.Close()
	for _, o := range outcomes {
		s.ConnectorReliability = append(s.ConnectorReliability, *o)
	}

	if err := pool.QueryRow(ctx,
		`select count(*) from audit_log where tenant_id = $1 and action = 'approval.reject'`, tenantID,
	).Scan(&s.AnalystOverrideCount); err != nil {
		return Snapshot{}, err
	}

	if err := pool.QueryRow(ctx,
		`select count(*) from audit_log where tenant_id = $1 and action = 'action.release' and actor = 'scheduler'`, tenantID,
	).Scan(&s.AutoReleaseCount); err != nil {
		return Snapshot{}, err
	}
	if err := pool.QueryRow(ctx,
		`select count(*) from audit_log where tenant_id = $1 and action = 'action.release' and actor != 'scheduler'`, tenantID,
	).Scan(&s.ManualReleaseCount); err != nil {
		return Snapshot{}, err
	}

	if err := pool.QueryRow(ctx, `
		select avg(extract(epoch from (al."timestamp" - c.opened_at)))
		from audit_log al
		join cases c on c.id = al.case_id
		where al.tenant_id = $1 and al.action = 'action.execute' and al.decision = 'allowed'`, tenantID,
	).Scan(&s.MeanTimeToContainmentSeconds); err != nil {
		return Snapshot{}, err
	}

	if err := pool.QueryRow(ctx, `
		select
			count(*) filter (where exists (
				select 1 from evidence e where e.case_id = c.id and e.integrity_status = 'verified'
			)),
			count(*)
		from cases c
		where c.tenant_id = $1`, tenantID,
	).Scan(&s.CasesWithVerifiedEvidence, &s.TotalCases); err != nil {
		return Snapshot{}, err
	}

	return s, nil
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// Get implements GET /v1/metrics.
func (h *Handler) Get(c *gin.Context) {
	tenant := httpx.TenantID(c)
	snap, err := Compute(c.Request.Context(), h.pool, tenant)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("compute_metrics_failed", "could not compute metrics").WithCause(err))
		return
	}
	if snap.ConnectorReliability == nil {
		snap.ConnectorReliability = []ConnectorOutcome{}
	}
	c.JSON(http.StatusOK, snap)
}
