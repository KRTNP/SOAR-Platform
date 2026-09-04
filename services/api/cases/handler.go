package cases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/audit"
	"soar-platform/services/api/auth"
	"soar-platform/services/api/evidence"
	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/outbox"
	"soar-platform/services/api/signatures"
	"soar-platform/services/api/store"
	"soar-platform/services/api/telemetry"
	"soar-platform/services/api/ticketing"
)

const createRoute = "createCase"

type Handler struct {
	pool *pgxpool.Pool
	rec  *telemetry.Recorder
}

// NewHandler wires the cases API to a pool and a telemetry.Recorder — rec
// may be nil (every telemetry.Recorder method is nil-receiver-safe).
func NewHandler(pool *pgxpool.Pool, rec *telemetry.Recorder) *Handler {
	return &Handler{pool: pool, rec: rec}
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
	cs := NewFromAlert(caseID, tenant, category, string(req.Severity), confidence, "system", time.Now().UTC(), nil, nil)
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
		if err := auditAppend(ctx, tx, tenant, &caseID, "system", "system", "case.create", "completed", audit.TargetHash(caseID), nil); err != nil {
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
	ticket, err := ticketing.Get(ctx, h.pool, tenant, caseID)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("get_case_ticket_failed", "could not load case ticket").WithCause(err))
		return
	}
	action, approval, err := actions.GetLatestForCase(ctx, h.pool, tenant, caseID)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("get_case_action_failed", "could not load case action").WithCause(err))
		return
	}
	c.JSON(http.StatusOK, toAPICaseDetail(cs, observables, ticket, action, approval))
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
			if before.Status == "closed" {
				return httpx.ErrConflict("case_already_closed", "a closed case cannot be transitioned; it is a terminal state")
			}
			if err := Transition(ctx, tx, tenant, caseID, newStatus); err != nil {
				return err
			}
			detail := map[string]any{"from": before.Status, "to": newStatus}
			if req.Reason != nil {
				detail["reason"] = *req.Reason
			}
			if err := AppendTimeline(ctx, tx, caseID, tenant, "status_changed", "analyst", detail); err != nil {
				return err
			}
			return auditAppend(ctx, tx, tenant, &caseID, "analyst", "analyst", "case.transition", "completed", audit.TargetHash(caseID), req.Reason)
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
			if live, err := actions.HasLiveAction(ctx, tx, tenant, caseID); err != nil {
				return err
			} else if live {
				return httpx.ErrConflict("case_has_live_action", "cannot close a case with a block still in effect; release it first")
			}
			// adr/0019's remaining close-gate clause: "no repeated alert is
			// present." Scoped to a case that actually reached its matching
			// signature's auto-trigger threshold (services/api/signatures,
			// adr/0021 — cases may import signatures, it has no dependency
			// on cases or events) — not every merely-correlated case. A case
			// with 1-2 low-signal correlated events that never escalated to
			// an auto-requested block is not "an active attack pattern" in
			// the sense this clause means; gating those too broke closing
			// them almost immediately after creation, which is normal,
			// expected usage (found via
			// TestCaseClose_RejectsWhileActionExecuted's own regression).
			// eventClass, "" if CorrelationKey has no ":" — should not
			// happen since events.correlationKeyFor always writes
			// "eventClass:value", but an unmatched signature (nil) falls
			// back to 5, adr/0018's original SSH brute-force threshold.
			autoBlockThreshold := 5
			if before.CorrelationKey != nil {
				if eventClass, _, found := strings.Cut(*before.CorrelationKey, ":"); found {
					if sig := signatures.MatchEventClass(eventClass); sig != nil {
						autoBlockThreshold = sig.Threshold
					}
				}
			}
			if before.CorrelationKey != nil && before.CorrelationWindowSeconds != nil && before.RepeatCount >= autoBlockThreshold {
				windowEnd := before.UpdatedAt.Add(time.Duration(*before.CorrelationWindowSeconds) * time.Second)
				if time.Now().UTC().Before(windowEnd) {
					return httpx.ErrConflict("case_correlation_window_open",
						"cannot close while this case's correlation window is still open; a repeated alert may still arrive")
				}
			}
			if err := Close(ctx, tx, tenant, caseID, *req.ClosureReason); err != nil {
				return err
			}
			if err := ticketing.Close(ctx, tx, tenant, caseID, *req.ClosureReason); err != nil {
				return err
			}
			if err := AppendTimeline(ctx, tx, caseID, tenant, "case_closed", "analyst", map[string]any{
				"from": before.Status, "closure_reason": *req.ClosureReason,
			}); err != nil {
				return err
			}
			return auditAppend(ctx, tx, tenant, &caseID, "analyst", "analyst", "case.close", "completed", audit.TargetHash(caseID), req.ClosureReason)
		})
		if err != nil {
			httpx.WriteProblem(c, mapCaseError(err))
			return
		}
	case genapi.OpenTicket:
		err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
			cs, err := Get(ctx, tx, tenant, caseID)
			if err != nil {
				return err
			}
			observables, err := GetObservables(ctx, tx, tenant, caseID)
			if err != nil {
				return err
			}
			evidenceRefs := make([]string, 0, len(observables))
			for _, o := range observables {
				evidenceRefs = append(evidenceRefs, o.Type+":"+o.Value)
			}
			summary := fmt.Sprintf("%s (%s severity, owner: %s)", cs.Title, cs.Severity, cs.Owner)
			ticket, created, err := ticketing.Open(ctx, tx, tenant, caseID, summary, evidenceRefs)
			if err != nil {
				return err
			}
			if !created {
				return nil
			}
			if err := AppendTimeline(ctx, tx, caseID, tenant, "ticket_opened", "analyst", map[string]any{
				"ticket_ref": ticket.ExternalRef,
			}); err != nil {
				return err
			}
			return auditAppend(ctx, tx, tenant, &caseID, "analyst", "analyst", "case.open_ticket", "completed", audit.TargetHash(ticket.ExternalRef), nil)
		})
		if err != nil {
			httpx.WriteProblem(c, mapCaseError(err))
			return
		}
	case genapi.RequestBlock:
		if req.TargetValue == nil || strings.TrimSpace(*req.TargetValue) == "" {
			httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "target_value is required to request a block"))
			return
		}
		targetValue := *req.TargetValue
		err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
			observables, err := GetObservables(ctx, tx, tenant, caseID)
			if err != nil {
				return err
			}
			targetType := ""
			for _, o := range observables {
				if o.Value == targetValue {
					targetType = o.Type
					break
				}
			}
			if targetType == "" {
				return httpx.ErrValidation("target_not_on_case", "target_value must match an observable already on this case")
			}
			actor, actorType := auth.ActorFrom(c, "analyst", "analyst")
			a, _, created, err := actions.RequestBlock(ctx, tx, tenant, caseID, targetType, targetValue, actor)
			if err != nil {
				return err
			}
			if !created {
				return nil
			}
			h.rec.ActionRequested(a.Intent)
			if err := AppendTimeline(ctx, tx, caseID, tenant, "block_requested", actor, map[string]any{
				"action_id": a.ID, "target": targetValue,
			}); err != nil {
				return err
			}
			// adr/0016/adr/0024: the approval_roles this intent's policy rule
			// names is recorded on the audit trail for visibility; the actual
			// enforcement against a caller's roles happens later, when
			// someone approves it (actions.Handler.Approve).
			_, approvalRoles := actions.EvaluatePolicy(a.Intent, a.Risk)
			reason := "requires approval from: " + strings.Join(approvalRoles, ", ")
			return auditAppend(ctx, tx, tenant, &caseID, actor, actorType, "action.request_block", "completed", audit.TargetHash(targetValue), &reason)
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
	projectCaseBestEffort(ctx, h.pool, tenant, cs)
	c.JSON(http.StatusOK, toAPICase(cs))
}

// projectCaseBestEffort re-projects a case into the OpenSearch outbox
// (adr/0022) after any of Update's five operations mutate it — without
// this, a case's search document goes stale the moment it's transitioned,
// closed, or otherwise changed outside of event ingestion (the only other
// writer, events.Handler.Ingest). Best-effort/log-only, the same posture
// actions/handler.go's auditDeniedExecute already uses: a search-projection
// failure must never fail a case mutation that already succeeded. Runs
// against pool directly, not the mutation's own transaction (already
// committed by the time this is called) — outbox.Write's querier interface
// only needs Exec, which *pgxpool.Pool already satisfies.
func projectCaseBestEffort(ctx context.Context, pool *pgxpool.Pool, tenant string, cs *Case) {
	key := ""
	if cs.CorrelationKey != nil {
		key = *cs.CorrelationKey
	}
	payload := map[string]any{
		"id": cs.ID, "tenant_id": tenant, "title": cs.Title, "category": cs.Category,
		"severity": cs.Severity, "status": cs.Status, "repeat_count": cs.RepeatCount,
		"correlation_key": key, "updated_at": cs.UpdatedAt,
	}
	if err := outbox.Write(ctx, pool, tenant, "case", cs.ID, payload, time.Now().UTC()); err != nil {
		slog.Default().Error("failed to project case update to search outbox", "case_id", cs.ID, "err", err)
	}
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

// Evidence implements GET /v1/cases/{caseId}/evidence. It does not re-verify
// each row's hash — that belongs to the single-item evidence.Get gate; see
// adr/0012-evidence-integrity-and-audit-export.md.
func (h *Handler) Evidence(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	caseID := c.Param("caseId")

	if _, err := Get(ctx, h.pool, tenant, caseID); err != nil {
		httpx.WriteProblem(c, mapCaseError(err))
		return
	}
	items, err := evidence.ListForCase(ctx, h.pool, tenant, caseID)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("list_case_evidence_failed", "could not list case evidence").WithCause(err))
		return
	}
	apiItems := make([]genapi.Evidence, 0, len(items))
	for _, ev := range items {
		apiItems = append(apiItems, evidence.ToAPI(ev))
	}
	c.JSON(http.StatusOK, gin.H{"items": apiItems})
}

// auditAppend is a thin convenience wrapper so every case-mutation branch
// above stays a one-line addition next to its existing AppendTimeline call.
func auditAppend(ctx context.Context, tx pgx.Tx, tenant string, caseID *string, actor, actorType, action, decision, targetHash string, reason *string) error {
	th := targetHash
	return audit.Append(ctx, tx, audit.Record{
		TenantID:   tenant,
		Actor:      actor,
		ActorType:  actorType,
		CaseID:     caseID,
		Action:     action,
		TargetHash: &th,
		Decision:   decision,
		Reason:     reason,
	})
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

// toAPICaseDetail is toAPICase plus the resolved observables, ticket and
// most recent action — only the single-case GET pays for those joins (see
// Handler.Get).
func toAPICaseDetail(c *Case, observables []ObservableRef, ticket *ticketing.Ticket, action *actions.Action, approval *actions.Approval) genapi.Case {
	out := toAPICase(c)
	if len(observables) > 0 {
		apiObs := make([]genapi.CaseObservable, 0, len(observables))
		for _, o := range observables {
			apiObs = append(apiObs, genapi.CaseObservable{
				Type:              o.Type,
				Value:             o.Value,
				Verdict:           verdictPtr(o.Verdict),
				TiConfidence:      float32Ptr(o.TiConfidence),
				SourceReliability: o.SourceReliability,
			})
		}
		out.Observables = &apiObs
	}
	if ticket != nil {
		out.TicketRef = &ticket.ExternalRef
		status := genapi.CaseTicketStatus(ticket.Status)
		out.TicketStatus = &status
	}
	if action != nil {
		pending := actions.ToAPIAction(*action, *approval)
		out.PendingAction = &pending
	}
	return out
}

func verdictPtr(v *string) *genapi.CaseObservableVerdict {
	if v == nil {
		return nil
	}
	verdict := genapi.CaseObservableVerdict(*v)
	return &verdict
}

func float32Ptr(v *float64) *float32 {
	if v == nil {
		return nil
	}
	f := float32(*v)
	return &f
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
