package httpx

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
)

type ctxKey string

const (
	requestIDKey ctxKey = "request_id"
	tenantIDKey  ctxKey = "tenant_id"
)

// RequestIDMiddleware threads X-Request-ID through the request, generating a
// ULID when the caller didn't supply one, and echoes it back on the response.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = ulid.Make().String()
		}
		c.Set(string(requestIDKey), id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// RequestID returns the request id set by RequestIDMiddleware, or "" if the
// middleware hasn't run (should not happen outside tests).
func RequestID(c *gin.Context) string {
	v, _ := c.Get(string(requestIDKey))
	id, _ := v.(string)
	return id
}

// TenantKeyMiddleware resolves the caller's tenant from an optional
// X-Tenant-Key header via resolve (services/api/tenants.Resolve, bound to a
// pool by the caller — httpx stays dependency-free of any domain package).
// adr/0022: this is a lab-appropriate tenant *selector*, not authentication
// — no identity, no session, no RBAC. No header -> defaultTenantID (every
// existing curl/test/UI call that never sends the header keeps today's
// exact behavior, the same M1 auth-placeholder posture this replaces). A
// header present but unrecognized -> 401, never a silent fallback to the
// wrong tenant.
func TenantKeyMiddleware(defaultTenantID string, resolve func(ctx context.Context, key string) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Tenant-Key")
		if key == "" {
			c.Set(string(tenantIDKey), defaultTenantID)
			c.Next()
			return
		}
		tenantID, err := resolve(c.Request.Context(), key)
		if err != nil {
			WriteProblem(c, ErrUnauthorized("invalid_tenant_key", "X-Tenant-Key did not match any tenant"))
			c.Abort()
			return
		}
		c.Set(string(tenantIDKey), tenantID)
		c.Next()
	}
}

// TenantID returns the tenant id set by TenantKeyMiddleware.
func TenantID(c *gin.Context) string {
	v, _ := c.Get(string(tenantIDKey))
	id, _ := v.(string)
	return id
}

// RecoveryMiddleware converts a panic into a 500 Problem instead of a bare
// connection reset, and logs the panic with the request id attached.
func RecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered", "panic", r, "request_id", RequestID(c), "path", c.Request.URL.Path)
				WriteProblem(c, ErrInternal("internal_error", "An unexpected error occurred"))
			}
		}()
		c.Next()
	}
}

// AccessLogMiddleware logs one structured line per request.
func AccessLogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			"request_id", RequestID(c),
			"tenant_id", TenantID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}
