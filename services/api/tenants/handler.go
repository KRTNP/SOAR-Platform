package tenants

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

// List implements GET /v1/tenants. No auth gate yet (adr/0011's disclosed
// posture), same as /v1/allowlist and /v1/critical-targets.
func (h *Handler) List(c *gin.Context) {
	list, err := List(c.Request.Context(), h.pool)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("list_tenants_failed", "could not list tenants").WithCause(err))
		return
	}
	items := make([]genapi.Tenant, 0, len(list))
	for _, t := range list {
		items = append(items, toAPI(t, ""))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Create implements POST /v1/tenants. The plaintext api_key is returned
// exactly once, in this response — only its hash is ever persisted.
func (h *Handler) Create(c *gin.Context) {
	var req genapi.TenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "name is required"))
		return
	}

	t, key, err := Create(c.Request.Context(), h.pool, req.Name, time.Now().UTC())
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("create_tenant_failed", "could not create tenant").WithCause(err))
		return
	}
	c.JSON(http.StatusCreated, toAPI(t, key))
}

func toAPI(t Tenant, apiKey string) genapi.Tenant {
	out := genapi.Tenant{
		Id:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
	}
	if apiKey != "" {
		out.ApiKey = &apiKey
	}
	return out
}
