package httpx

import (
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

// TenantStubMiddleware is the M1 auth placeholder: every request is treated
// as belonging to a single configured tenant. Replace with real OIDC/Keycloak
// token verification when multi-tenant auth becomes a requirement.
func TenantStubMiddleware(defaultTenantID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(tenantIDKey), defaultTenantID)
		c.Next()
	}
}

// TenantID returns the tenant id set by TenantStubMiddleware.
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
