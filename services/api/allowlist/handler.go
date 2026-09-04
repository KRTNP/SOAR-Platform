package allowlist

import (
	"errors"
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

// List implements GET /v1/allowlist.
func (h *Handler) List(c *gin.Context) {
	tenant := httpx.TenantID(c)
	sources, err := List(c.Request.Context(), h.pool, tenant)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("list_allowlist_failed", "could not list trusted sources").WithCause(err))
		return
	}
	items := make([]genapi.TrustedSource, 0, len(sources))
	for _, s := range sources {
		items = append(items, toAPI(s))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Add implements POST /v1/allowlist. No auth/RBAC yet (see adr/0011), so
// created_by is a fixed actor string, the same posture every other
// analyst-attributed write in this codebase already discloses.
func (h *Handler) Add(c *gin.Context) {
	var req genapi.TrustedSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}
	if strings.TrimSpace(req.Value) == "" || strings.TrimSpace(req.Reason) == "" {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "value and reason are required"))
		return
	}

	tenant := httpx.TenantID(c)
	t, err := Add(c.Request.Context(), h.pool, tenant, req.Value, req.Reason, "analyst", req.ExpiresAt, time.Now().UTC())
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("add_trusted_source_failed", "could not add trusted source").WithCause(err))
		return
	}
	c.JSON(http.StatusCreated, toAPI(*t))
}

// Delete implements DELETE /v1/allowlist/:id.
func (h *Handler) Delete(c *gin.Context) {
	tenant := httpx.TenantID(c)
	if err := Delete(c.Request.Context(), h.pool, tenant, c.Param("id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteProblem(c, httpx.ErrNotFound("trusted_source_not_found", "trusted source not found"))
			return
		}
		httpx.WriteProblem(c, httpx.ErrInternal("delete_trusted_source_failed", "could not delete trusted source").WithCause(err))
		return
	}
	c.Status(http.StatusNoContent)
}

func toAPI(t TrustedSource) genapi.TrustedSource {
	return genapi.TrustedSource{
		Id:        t.ID,
		Value:     t.Value,
		Reason:    t.Reason,
		CreatedBy: t.CreatedBy,
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
	}
}
