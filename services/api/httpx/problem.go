// Package httpx holds cross-cutting HTTP concerns shared by every domain
// package: Problem-shaped errors, request-id/tenant middleware, and the
// idempotency-key helpers. It owns no business logic.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"soar-platform/services/api/genapi"
)

// AppError is a domain error that already knows how to render itself as the
// OpenAPI-documented Problem shape (api/openapi.yaml#/components/schemas/Problem).
type AppError struct {
	Status    int
	Code      string
	Title     string
	Detail    string
	Retryable bool
	err       error
}

func (e *AppError) Error() string {
	if e.err != nil {
		return e.Title + ": " + e.err.Error()
	}
	return e.Title
}

func (e *AppError) Unwrap() error { return e.err }

func newError(status int, code, title string, retryable bool) *AppError {
	return &AppError{Status: status, Code: code, Title: title, Retryable: retryable}
}

// WithDetail attaches a human-readable, non-sensitive explanation.
func (e *AppError) WithDetail(detail string) *AppError {
	cp := *e
	cp.Detail = detail
	return &cp
}

// WithCause records the underlying error for logs without leaking it to the client.
func (e *AppError) WithCause(err error) *AppError {
	cp := *e
	cp.err = err
	return &cp
}

func ErrValidation(code, title string) *AppError {
	return newError(http.StatusBadRequest, code, title, false)
}

func ErrNotFound(code, title string) *AppError {
	return newError(http.StatusNotFound, code, title, false)
}

func ErrConflict(code, title string) *AppError {
	return newError(http.StatusConflict, code, title, false)
}

func ErrForbidden(code, title string) *AppError {
	return newError(http.StatusForbidden, code, title, false)
}

// ErrUnauthorized is for a caller-supplied credential (adr/0022's
// X-Tenant-Key) that doesn't resolve to anything — distinct from
// ErrForbidden, which is for an identified caller lacking permission.
func ErrUnauthorized(code, title string) *AppError {
	return newError(http.StatusUnauthorized, code, title, false)
}

func ErrUnsupported(code, title string) *AppError {
	return newError(http.StatusUnprocessableEntity, code, title, false)
}

func ErrInternal(code, title string) *AppError {
	return newError(http.StatusInternalServerError, code, title, true)
}

// WriteProblem renders err as application/problem+json. Any error that is not
// an *AppError is treated as an unclassified internal failure — the client
// never sees its raw message, only a stable code plus the request id needed
// to find it in logs.
func WriteProblem(c *gin.Context, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = ErrInternal("internal_error", "An unexpected error occurred").WithCause(err)
	}
	if appErr.Status >= 500 {
		slog.Default().Error("request failed", "code", appErr.Code, "request_id", RequestID(c), "cause", appErr.Unwrap())
	}

	problem := genapi.Problem{
		Type:      "https://soar.lab.local/problems/" + appErr.Code,
		Title:     appErr.Title,
		Status:    appErr.Status,
		Code:      appErr.Code,
		RequestId: RequestID(c),
		Retryable: appErr.Retryable,
	}
	if appErr.Detail != "" {
		problem.Detail = &appErr.Detail
	}

	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(appErr.Status, problem)
}
