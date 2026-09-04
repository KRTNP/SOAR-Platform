package actions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/audit"
	"soar-platform/services/api/auth"
	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/store"
	"soar-platform/services/api/telemetry"
)

type Handler struct {
	pool *pgxpool.Pool
	net  NetworkConnector
	rec  *telemetry.Recorder
}

// NewHandler wires the actions API to a pool and a NetworkConnector — the
// mock services/api/network by default, or the real services/api/opnsense
// when the caller (services/api/main.go) opts in via OPNSENSE_ENABLE_REAL.
// rec may be nil (every telemetry.Recorder method is nil-receiver-safe).
func NewHandler(pool *pgxpool.Pool, net NetworkConnector, rec *telemetry.Recorder) *Handler {
	return &Handler{pool: pool, net: net, rec: rec}
}

// Preview implements POST /v1/actions/preview — a dry-run description of
// what requesting this intent/target would do. It never writes to the
// database; the actual action+approval pair is created via the case's
// request_block operation (cases.Handler.Update). When the wired connector
// exposes a live state check (h.net.Preview — currently only the real
// services/api/opnsense connector, see adr/0013), the message also reports
// whether the target is already in effect; the mock connector has no such
// check, so its preview stays the generic message it always was.
func (h *Handler) Preview(c *gin.Context) {
	var req genapi.PreviewActionJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}
	if strings.TrimSpace(req.Intent) == "" || strings.TrimSpace(req.Target) == "" {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "intent and target are required"))
		return
	}
	if req.Intent != "block_network_indicator" {
		httpx.WriteProblem(c, httpx.ErrValidation("unsupported_intent", "only block_network_indicator has a preview in this milestone"))
		return
	}
	msg := fmt.Sprintf("dry_run: %s against %s would require an approved approval before executing; no state was changed", req.Intent, req.Target)
	if h.net.Preview != nil {
		alreadyInEffect, err := h.net.Preview(c.Request.Context(), req.Target)
		if err != nil {
			httpx.WriteProblem(c, httpx.ErrInternal("preview_failed", "could not check connector state").WithCause(err))
			return
		}
		if alreadyInEffect {
			msg = fmt.Sprintf("dry_run: %s target %s is already blocked; executing this action would have no additional effect", req.Intent, req.Target)
		}
	}
	c.JSON(http.StatusOK, genapi.StatusResponse{Status: "dry_run", Message: &msg})
}

// Get implements GET /v1/actions/{actionId}.
func (h *Handler) Get(c *gin.Context) {
	tenant := httpx.TenantID(c)
	a, ap, err := Get(c.Request.Context(), h.pool, tenant, c.Param("actionId"))
	if err != nil {
		httpx.WriteProblem(c, mapActionError(err))
		return
	}
	c.JSON(http.StatusOK, ToAPIAction(*a, *ap))
}

// Execute implements POST /v1/actions/{actionId}/execute — the M3 gate.
// Execute itself re-verifies the approval inside the transaction; this
// handler only translates the typed deny reasons to the 403 the openapi
// contract already declares for this route.
func (h *Handler) Execute(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	actionID := c.Param("actionId")
	if len(c.GetHeader("Idempotency-Key")) < httpx.MinIdempotencyKeyLength {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_idempotency_key", "Idempotency-Key header must be at least 8 characters"))
		return
	}

	actor, actorType := auth.ActorFrom(c, "incident-commander", "analyst")
	var a Action
	err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
		var err error
		a, err = Execute(ctx, tx, tenant, actionID, h.net)
		if err != nil {
			return err
		}
		th := audit.TargetHash(actionID)
		cv := h.net.Version
		return audit.Append(ctx, tx, audit.Record{
			TenantID: tenant, Actor: actor, ActorType: actorType,
			CaseID: &a.CaseID, ExecutionID: &a.ID, Action: "action.execute",
			TargetHash: &th, ConnectorVersion: &cv, Decision: "allowed",
		})
	})
	if err != nil {
		problem := mapExecuteError(err)
		var appErr *httpx.AppError
		if errors.As(problem, &appErr) && appErr.Status == http.StatusForbidden {
			h.rec.ActionExecuted("denied")
		} else {
			h.rec.ActionExecuted("error")
		}
		auditDeniedExecute(ctx, h.pool, tenant, actionID, problem, actor, actorType)
		httpx.WriteProblem(c, problem)
		return
	}
	h.rec.ActionExecuted("success")
	h.rec.ConnectorCall(h.net.Version, "success")
	c.JSON(http.StatusAccepted, genapi.ExecutionAccepted{
		ExecutionId: a.ID,
		Status:      genapi.ExecutionAcceptedStatusAccepted,
	})
}

// Release implements POST /v1/actions/{actionId}/release — the manual
// early-rollback path (see adr/0011, adr/0013's disclosed gap that this
// endpoint did not exist until now). Unlike a denied Execute, there is no
// approval gate to re-check here: releasing early is always allowed once
// something has actually been executed.
func (h *Handler) Release(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	actionID := c.Param("actionId")
	if len(c.GetHeader("Idempotency-Key")) < httpx.MinIdempotencyKeyLength {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_idempotency_key", "Idempotency-Key header must be at least 8 characters"))
		return
	}

	actor, actorType := auth.ActorFrom(c, "incident-commander", "analyst")
	var a Action
	err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
		var err error
		a, err = Release(ctx, tx, tenant, actionID, h.net)
		if err != nil {
			return err
		}
		th := audit.TargetHash(actionID)
		cv := h.net.Version
		return audit.Append(ctx, tx, audit.Record{
			TenantID: tenant, Actor: actor, ActorType: actorType,
			CaseID: &a.CaseID, ExecutionID: &a.ID, Action: "action.release",
			TargetHash: &th, ConnectorVersion: &cv, Decision: "allowed",
		})
	})
	if err != nil {
		httpx.WriteProblem(c, mapReleaseError(err))
		return
	}
	h.rec.ActionReleased("manual")
	c.JSON(http.StatusOK, genapi.ReleaseAccepted{
		ActionId: a.ID,
		Status:   genapi.ReleaseAcceptedStatusReleased,
	})
}

// auditDeniedExecute records a denied execute attempt as a second, independent
// statement against the pool rather than inside the transaction that Execute
// already rolled back — an audit insert made inside that transaction would
// vanish along with it. Best-effort: a logging failure here does not change
// the 403 already decided for the caller, same posture as WriteProblem's own
// internal-error logging.
func auditDeniedExecute(ctx context.Context, pool *pgxpool.Pool, tenant, actionID string, problem error, actor, actorType string) {
	var appErr *httpx.AppError
	if !errors.As(problem, &appErr) || appErr.Status != http.StatusForbidden {
		return
	}
	th := audit.TargetHash(actionID)
	reason := appErr.Code
	caseID := ""
	if a, _, err := Get(ctx, pool, tenant, actionID); err == nil {
		caseID = a.CaseID
	}
	var caseIDPtr *string
	if caseID != "" {
		caseIDPtr = &caseID
	}
	if err := audit.Append(ctx, pool, audit.Record{
		TenantID: tenant, Actor: actor, ActorType: actorType,
		CaseID: caseIDPtr, ExecutionID: &actionID, Action: "action.execute",
		TargetHash: &th, Decision: "denied", Reason: &reason,
	}); err != nil {
		slog.Default().Error("failed to record denied-execute audit entry", "action_id", actionID, "err", err)
	}
}

// ListApprovals implements GET /v1/approvals.
func (h *Handler) ListApprovals(c *gin.Context) {
	tenant := httpx.TenantID(c)
	views, err := ListApprovals(c.Request.Context(), h.pool, tenant)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("list_approvals_failed", "could not list approvals").WithCause(err))
		return
	}
	items := make([]genapi.Approval, 0, len(views))
	for _, v := range views {
		items = append(items, ToAPIApproval(v.Approval))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Approve implements POST /v1/approvals/{approvalId}/approve. When
// auth.Enabled(), the caller's real roles are checked against
// EvaluatePolicy's approvalRoles for the underlying action's intent before
// anything is decided (adr/0024) — with auth off, that check is skipped
// entirely, matching this endpoint's pre-adr/0024 "anyone can approve"
// behavior.
func (h *Handler) Approve(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	if len(c.GetHeader("Idempotency-Key")) < httpx.MinIdempotencyKeyLength {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_idempotency_key", "Idempotency-Key header must be at least 8 characters"))
		return
	}
	approvalID := c.Param("approvalId")
	if claims, ok := auth.FromContext(c); ok {
		pending, _, err := GetByApprovalID(ctx, h.pool, tenant, approvalID)
		if err != nil {
			httpx.WriteProblem(c, mapDecisionError(err))
			return
		}
		_, approvalRoles := EvaluatePolicy(pending.Intent, pending.Risk)
		if !claims.HasRole(approvalRoles...) {
			httpx.WriteProblem(c, httpx.ErrForbidden("insufficient_role", "caller's role cannot approve this action"))
			return
		}
	}

	actor, actorType := auth.ActorFrom(c, "incident-commander", "analyst")
	a, ap, err := Approve(ctx, h.pool, tenant, approvalID, actor)
	if err != nil {
		httpx.WriteProblem(c, mapDecisionError(err))
		return
	}
	th := audit.TargetHash(a.ID)
	if err := audit.Append(ctx, h.pool, audit.Record{
		TenantID: tenant, Actor: actor, ActorType: actorType,
		CaseID: &a.CaseID, ExecutionID: &a.ID, Action: "approval.approve",
		TargetHash: &th, Decision: "completed",
	}); err != nil {
		slog.Default().Error("failed to record approval audit entry", "approval_id", ap.ID, "err", err)
	}
	h.rec.ApprovalDecided("approved")
	c.JSON(http.StatusOK, ToAPIApproval(ap))
}

// Reject implements POST /v1/approvals/{approvalId}/reject.
func (h *Handler) Reject(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	if len(c.GetHeader("Idempotency-Key")) < httpx.MinIdempotencyKeyLength {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_idempotency_key", "Idempotency-Key header must be at least 8 characters"))
		return
	}
	var req genapi.RejectApprovalJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "reason is required to reject an approval"))
		return
	}
	approvalID := c.Param("approvalId")
	if claims, ok := auth.FromContext(c); ok {
		pending, _, err := GetByApprovalID(ctx, h.pool, tenant, approvalID)
		if err != nil {
			httpx.WriteProblem(c, mapDecisionError(err))
			return
		}
		_, approvalRoles := EvaluatePolicy(pending.Intent, pending.Risk)
		if !claims.HasRole(approvalRoles...) {
			httpx.WriteProblem(c, httpx.ErrForbidden("insufficient_role", "caller's role cannot reject this action"))
			return
		}
	}

	actor, actorType := auth.ActorFrom(c, "incident-commander", "analyst")
	a, ap, err := Reject(ctx, h.pool, tenant, approvalID, req.Reason)
	if err != nil {
		httpx.WriteProblem(c, mapDecisionError(err))
		return
	}
	th := audit.TargetHash(a.ID)
	if err := audit.Append(ctx, h.pool, audit.Record{
		TenantID: tenant, Actor: actor, ActorType: actorType,
		CaseID: &a.CaseID, ExecutionID: &a.ID, Action: "approval.reject",
		TargetHash: &th, Decision: "completed", Reason: &req.Reason,
	}); err != nil {
		slog.Default().Error("failed to record rejection audit entry", "approval_id", ap.ID, "err", err)
	}
	h.rec.ApprovalDecided("rejected")
	c.JSON(http.StatusOK, ToAPIApproval(ap))
}

func mapActionError(err error) error {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		return err
	}
	if errors.Is(err, ErrNotFound) {
		return httpx.ErrNotFound("action_not_found", "action does not exist")
	}
	return httpx.ErrInternal("action_operation_failed", "action operation failed").WithCause(err)
}

// mapExecuteError renders every approval-gate denial as 403, matching the
// executeAction 403 already declared in api/openapi.yaml — "no action
// before approval" is a permission failure, not a state conflict.
func mapExecuteError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.ErrNotFound("action_not_found", "action does not exist")
	case errors.Is(err, ErrApprovalRequired):
		return httpx.ErrForbidden("approval_required", "this action has not been approved yet")
	case errors.Is(err, ErrApprovalRejected):
		return httpx.ErrForbidden("approval_rejected", "this action's approval was rejected")
	case errors.Is(err, ErrApprovalExpired):
		return httpx.ErrForbidden("approval_expired", "this action's approval has expired")
	default:
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			return err
		}
		return httpx.ErrInternal("execute_action_failed", "could not execute action").WithCause(err)
	}
}

// mapReleaseError renders "nothing to release yet" as 409, matching
// releaseAction's declared 409 — releasing before execute is a state
// conflict, not a permission failure (there is no approval gate on release).
func mapReleaseError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.ErrNotFound("action_not_found", "action does not exist")
	case errors.Is(err, ErrNotExecuted):
		return httpx.ErrConflict("action_not_executed", "action has not been executed yet, nothing to release")
	default:
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			return err
		}
		return httpx.ErrInternal("release_action_failed", "could not release action").WithCause(err)
	}
}

// mapDecisionError renders a stale decision as 409, matching approveAction
// and rejectApproval's declared 409 — approving/rejecting is a state
// conflict, not a permission failure.
func mapDecisionError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.ErrNotFound("approval_not_found", "approval does not exist")
	case errors.Is(err, ErrApprovalExpired):
		return httpx.ErrConflict("approval_expired", "this approval has expired")
	case errors.Is(err, ErrAlreadyDecided):
		return httpx.ErrConflict("approval_already_decided", "this approval was already decided")
	default:
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			return err
		}
		return httpx.ErrInternal("approval_decision_failed", "could not record approval decision").WithCause(err)
	}
}

func ToAPIAction(a Action, ap Approval) genapi.Action {
	approvalID := ap.ID
	out := genapi.Action{
		Id:             a.ID,
		CaseId:         a.CaseID,
		ApprovalId:     &approvalID,
		Intent:         a.Intent,
		Risk:           genapi.ActionRisk(a.Risk),
		Mode:           genapi.ActionMode(a.Mode),
		Status:         genapi.ActionStatus(a.Status),
		RequestedBy:    a.RequestedBy,
		IdempotencyKey: a.IdempotencyKey,
		Reason:         ap.Reason,
	}
	out.Target.Type = a.TargetType
	out.Target.Value = a.TargetValue
	expiresAt := ap.ExpiresAt
	out.ExpiresAt = &expiresAt
	return out
}

func ToAPIApproval(ap Approval) genapi.Approval {
	return genapi.Approval{
		Id:         ap.ID,
		ActionId:   ap.ActionID,
		Status:     genapi.ApprovalStatus(ap.Status),
		ApprovedBy: ap.ApprovedBy,
		Reason:     ap.Reason,
		DecidedAt:  ap.DecidedAt,
		ExpiresAt:  ap.ExpiresAt,
	}
}
