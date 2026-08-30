package cases

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"

	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/store"
)

const createRoute = "createCase"

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// Create implements POST /v1/cases — manual case creation by an analyst.
// There is no auth yet, so owner is always "system"; replace once a real
// identity is available to attribute case ownership to.
func (h *Handler) Create(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)

	key := c.GetHeader("Idempotency-Key")
	if len(key) < httpx.MinIdempotencyKeyLength {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_idempotency_key", "Idempotency-Key header must be at least 8 characters"))
		return
	}
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_body", "could not read request body").WithCause(err))
		return
	}
	bodyHash := httpx.HashBody(bodyBytes)

	if existing, err := httpx.LookupIdempotency(ctx, h.pool, tenant, createRoute, key); err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("idempotency_lookup_failed", "could not check Idempotency-Key").WithCause(err))
		return
	} else if existing != nil {
		if existing.RequestHash != bodyHash {
			httpx.WriteProblem(c, httpx.ErrConflict("idempotency_key_reuse", "Idempotency-Key was already used with a different request body"))
			return
		}
		c.Data(existing.StatusCode, "application/json", existing.Body)
		return
	}

	var req genapi.CreateCaseRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "title is required"))
		return
	}
	if !req.Severity.Valid() {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_enum", "severity is not a recognized value"))
		return
	}
	category := "uncategorized"
	if req.Category != nil && *req.Category != "" {
		category = *req.Category
	}
	confidence := 0.5
	if req.Confidence != nil {
		confidence = float64(*req.Confidence)
	}

	caseID := "case-" + strings.ToLower(ulid.Make().String())
	cs := NewFromAlert(caseID, tenant, category, string(req.Severity), confidence, "system", time.Now().UTC())
	cs.Title = req.Title

	var apiCase genapi.Case
	txErr := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
		if err := Insert(ctx, tx, cs); err != nil {
			return err
		}
		if req.AlertIds != nil {
			for _, alertID := range *req.AlertIds {
				var exists bool
				if err := tx.QueryRow(ctx, `select exists(select 1 from alerts where tenant_id=$1 and id=$2)`, tenant, alertID).Scan(&exists); err != nil {
					return err
				}
				if !exists {
					return httpx.ErrValidation("alert_not_found", "alert_ids["+alertID+"] does not exist")
				}
				if err := LinkAlert(ctx, tx, caseID, alertID); err != nil {
					return err
				}
			}
		}
		if err := AppendTimeline(ctx, tx, caseID, tenant, "case_opened", "system", map[string]any{"origin": "manual"}); err != nil {
			return err
		}
		apiCase = toAPICase(cs)
		body, err := json.Marshal(apiCase)
		if err != nil {
			return err
		}
		return httpx.SaveIdempotency(ctx, tx, tenant, http.MethodPost, createRoute, key, bodyHash, http.StatusCreated, body)
	})
	if txErr != nil {
		httpx.WriteProblem(c, txErr)
		return
	}
	c.JSON(http.StatusCreated, apiCase)
}

// List implements GET /v1/cases.
func (h *Handler) List(c *gin.Context) {
	tenant := httpx.TenantID(c)
	limit := parseLimit(c.Query("limit"))
	cursor := c.Query("cursor")

	items, nextCursor, err := List(c.Request.Context(), h.pool, tenant, cursor, limit)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("list_cases_failed", "could not list cases").WithCause(err))
		return
	}
	apiItems := make([]genapi.Case, 0, len(items))
	for i := range items {
		apiItems = append(apiItems, toAPICase(&items[i]))
	}
	page := gin.H{"items": apiItems}
	if nextCursor != "" {
		page["next_cursor"] = nextCursor
	}
	c.JSON(http.StatusOK, page)
}

// Get implements GET /v1/cases/{caseId}. Unlike List, it also resolves the
// case's observables to human-readable {type, value} pairs (ADR-0009) — a
// per-row join isn't worth paying for every row in a list view.
func (h *Handler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	caseID := c.Param("caseId")

	cs, err := Get(ctx, h.pool, tenant, caseID)
	if err != nil {
		httpx.WriteProblem(c, mapCaseError(err))
		return
	}
	observables, err := GetObservables(ctx, h.pool, tenant, caseID)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("get_case_observables_failed", "could not load case observables").WithCause(err))
		return
	}
	c.JSON(http.StatusOK, toAPICaseDetail(cs, observables))
}

// Update implements PATCH /v1/cases/{caseId}.
func (h *Handler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	caseID := c.Param("caseId")

	var req genapi.CaseUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}

	switch req.Operation {
	case genapi.Update:
		var ownerID *string
		if req.OwnerId != nil {
			ownerID = req.OwnerId
		}
		err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
			return UpdateFields(ctx, tx, tenant, caseID, req.Title, ownerID)
		})
		if err != nil {
			httpx.WriteProblem(c, mapCaseError(err))
			return
		}
	case genapi.Transition:
		if req.Status == nil {
			httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "status is required for a transition operation"))
			return
		}
		newStatus := string(*req.Status)
		if !ValidStatus(newStatus) {
			httpx.WriteProblem(c, httpx.ErrValidation("invalid_enum", "status is not a recognized value"))
			return
		}
		if newStatus == "closed" {
			httpx.WriteProblem(c, httpx.ErrValidation("operation_not_supported",
				"transitioning directly to closed is not allowed; use the close operation with a closure_reason"))
			return
		}
		err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
			before, err := Get(ctx, tx, tenant, caseID)
			if err != nil {
				return err
			}
			if err := Transition(ctx, tx, tenant, caseID, newStatus); err != nil {
				return err
			}
			detail := map[string]any{"from": before.Status, "to": newStatus}
			if req.Reason != nil {
				detail["reason"] = *req.Reason
			}
			return AppendTimeline(ctx, tx, caseID, tenant, "status_changed", "analyst", detail)
		})
		if err != nil {
			httpx.WriteProblem(c, mapCaseError(err))
			return
		}
	case genapi.Close:
		if req.ClosureReason == nil || strings.TrimSpace(*req.ClosureReason) == "" {
			httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "closure_reason is required to close a case"))
			return
		}
		err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
			before, err := Get(ctx, tx, tenant, caseID)
			if err != nil {
				return err
			}
			if err := Close(ctx, tx, tenant, caseID, *req.ClosureReason); err != nil {
				return err
			}
			return AppendTimeline(ctx, tx, caseID, tenant, "case_closed", "analyst", map[string]any{
				"from": before.Status, "closure_reason": *req.ClosureReason,
			})
		})
		if err != nil {
			httpx.WriteProblem(c, mapCaseError(err))
			return
		}
	default:
		httpx.WriteProblem(c, httpx.ErrValidation("operation_not_supported",
			"operation "+string(req.Operation)+" has no case logic yet in this milestone"))
		return
	}

	cs, err := Get(ctx, h.pool, tenant, caseID)
	if err != nil {
		httpx.WriteProblem(c, mapCaseError(err))
		return
	}
	c.JSON(http.StatusOK, toAPICase(cs))
}

// Timeline implements GET /v1/cases/{caseId}/timeline.
func (h *Handler) Timeline(c *gin.Context) {
	tenant := httpx.TenantID(c)
	entries, err := GetTimeline(c.Request.Context(), h.pool, tenant, c.Param("caseId"))
	if err != nil {
		httpx.WriteProblem(c, mapCaseError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

func mapCaseError(err error) error {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		return err
	}
	if errors.Is(err, ErrNotFound) {
		return httpx.ErrNotFound("case_not_found", "case does not exist")
	}
	return httpx.ErrInternal("case_operation_failed", "case operation failed").WithCause(err)
}

func toAPICase(c *Case) genapi.Case {
	out := genapi.Case{
		Id:                c.ID,
		TenantId:          c.TenantID,
		Title:             c.Title,
		Category:          c.Category,
		Severity:          c.Severity,
		Confidence:        float32(c.Confidence),
		Status:            genapi.CaseStatus(c.Status),
		OpenedAt:          c.OpenedAt,
		Owner:             c.Owner,
		IncidentCommander: c.IncidentCommander,
		ServiceOwner:      c.ServiceOwner,
		ClosureReason:     c.ClosureReason,
		Hypothesis:        c.Hypothesis,
		CorrelationKey:    c.CorrelationKey,
		RootCause:         c.RootCause,
		LessonsLearned:    c.LessonsLearned,
	}
	if len(c.Facts) > 0 {
		out.Facts = &c.Facts
	}
	if len(c.Limitations) > 0 {
		out.Limitations = &c.Limitations
	}
	if !c.UpdatedAt.IsZero() {
		updated := c.UpdatedAt
		out.UpdatedAt = &updated
	}
	if c.ClosedAt != nil {
		out.ClosedAt = c.ClosedAt
	}
	return out
}

// toAPICaseDetail is toAPICase plus the resolved observables — only the
// single-case GET pays for that join (see Handler.Get).
func toAPICaseDetail(c *Case, observables []ObservableRef) genapi.Case {
	out := toAPICase(c)
	if len(observables) == 0 {
		return out
	}
	apiObs := make([]genapi.CaseObservable, 0, len(observables))
	for _, o := range observables {
		apiObs = append(apiObs, genapi.CaseObservable{Type: o.Type, Value: o.Value})
	}
	out.Observables = &apiObs
	return out
}

func parseLimit(raw string) int {
	if raw == "" {
		return 25
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 25
	}
	if n > 100 {
		return 100
	}
	return n
}
