package audit

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/httpx"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// List implements GET /v1/audit. Each item carries a read-time recomputed
// "verified" flag (see List's doc comment) so a tampered row is visible to
// the caller rather than silently trusted.
func (h *Handler) List(c *gin.Context) {
	tenant := httpx.TenantID(c)
	records, err := List(c.Request.Context(), h.pool, tenant, 100)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("list_audit_failed", "could not list audit records").WithCause(err))
		return
	}
	items := make([]map[string]any, 0, len(records))
	for _, r := range records {
		items = append(items, map[string]any{
			"id":                r.ID,
			"event_id":          r.EventID,
			"request_id":        r.RequestID,
			"actor":             r.Actor,
			"actor_type":        r.ActorType,
			"case_id":           r.CaseID,
			"execution_id":      r.ExecutionID,
			"action":            r.Action,
			"target_hash":       r.TargetHash,
			"policy_version":    r.PolicyVersion,
			"connector_version": r.ConnectorVersion,
			"decision":          r.Decision,
			"reason":            r.Reason,
			"timestamp":         r.Timestamp,
			"result_hash":       r.ResultHash,
			"verified":          r.Verified,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
