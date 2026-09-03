package criticaltargets

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// List implements GET /v1/critical-targets.
func (h *Handler) List(c *gin.Context) {
	tenant := httpx.TenantID(c)
	targets, err := List(c.Request.Context(), h.pool, tenant)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("list_critical_targets_failed", "could not list critical targets").WithCause(err))
		return
	}
	items := make([]genapi.CriticalTarget, 0, len(targets))
	for _, t := range targets {
		items = append(items, toAPI(t))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Add implements POST /v1/critical-targets. No auth/RBAC yet (see
// adr/0011), so created_by is a fixed actor string, the same posture every
// other analyst-attributed write in this codebase already discloses.
func (h *Handler) Add(c *gin.Context) {
	var req genapi.CriticalTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}
	if strings.TrimSpace(req.Value) == "" || strings.TrimSpace(req.Reason) == "" {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "value and reason are required"))
		return
	}

	tenant := httpx.TenantID(c)
	t, err := Add(c.Request.Context(), h.pool, tenant, req.Value, req.Reason, "analyst", time.Now().UTC())
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("add_critical_target_failed", "could not add critical target").WithCause(err))
		return
	}
	c.JSON(http.StatusCreated, toAPI(*t))
}

func toAPI(t CriticalTarget) genapi.CriticalTarget {
	return genapi.CriticalTarget{
		Id:        t.ID,
		Value:     t.Value,
		Reason:    t.Reason,
		CreatedBy: t.CreatedBy,
		CreatedAt: t.CreatedAt,
	}
}
