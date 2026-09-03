package events

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/alerts"
	"soar-platform/services/api/allowlist"
	"soar-platform/services/api/audit"
	"soar-platform/services/api/cases"
	"soar-platform/services/api/criticaltargets"
	"soar-platform/services/api/enrichment"
	"soar-platform/services/api/evidence"
	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/signatures"
	"soar-platform/services/api/store"
	"soar-platform/services/api/telemetry"
)

const ingestRoute = "ingestEvent"

// correlationKeyFor returns "" when eventClass/observables don't match any
// signatures.Registry entry — every unmatched event class keeps M1's
// original "always open a new case" behavior unchanged. adr/0021
// generalized this from adr/0018's single hardcoded
// authentication+ipv4 check into a registry lookup.
func correlationKeyFor(sig *signatures.Signature, eventClass string, observables []Observable) string {
	if sig == nil {
		return ""
	}
	for _, o := range observables {
		if o.Type == sig.ObservableType {
			return eventClass + ":" + o.Value
		}
	}
	return ""
}

// correlationValueFor mirrors correlationKeyFor's observable scan, for the
// allowlist check and the auto-trigger's block target — kept separate from
// the key itself so a future signature isn't forced to encode "the value to
// maybe block" into its key format.
func correlationValueFor(sig *signatures.Signature, observables []Observable) string {
	if sig == nil {
		return ""
	}
	for _, o := range observables {
		if o.Type == sig.ObservableType {
			return o.Value
		}
	}
	return ""
}

type Handler struct {
	pool *pgxpool.Pool
	rec  *telemetry.Recorder
}

// NewHandler wires the events API to a pool and a telemetry.Recorder — rec
// may be nil (e.g. in tests that don't care about metrics; every Recorder
// method is nil-receiver-safe).
func NewHandler(pool *pgxpool.Pool, rec *telemetry.Recorder) *Handler {
	return &Handler{pool: pool, rec: rec}
}

// Ingest implements POST /v1/events: normalize -> dedup -> alert -> case, all
// in one transaction, guarded by an Idempotency-Key.
func (h *Handler) Ingest(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)

	outcome := "rejected"
	defer func() { h.rec.IngestionOutcome(outcome) }()

	key := c.GetHeader("Idempotency-Key")
	if len(key) < httpx.MinIdempotencyKeyLength {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_idempotency_key",
			fmt.Sprintf("Idempotency-Key header must be at least %d characters", httpx.MinIdempotencyKeyLength)))
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_body", "could not read request body").WithCause(err))
		return
	}
	bodyHash := httpx.HashBody(bodyBytes)

	if existing, lookupErr := httpx.LookupIdempotency(ctx, h.pool, tenant, ingestRoute, key); lookupErr != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("idempotency_lookup_failed", "could not check Idempotency-Key").WithCause(lookupErr))
		return
	} else if existing != nil {
		if existing.RequestHash != bodyHash {
			httpx.WriteProblem(c, httpx.ErrConflict("idempotency_key_reuse", "Idempotency-Key was already used with a different request body"))
			return
		}
		var resp genapi.AcceptedEvent
		if err := json.Unmarshal(existing.Body, &resp); err == nil {
			// A retried ingest is always reported as a duplicate, even
			// though the original response said "accepted".
			resp.Status = genapi.AcceptedEventStatusDuplicate
			outcome = "duplicate"
			c.JSON(existing.StatusCode, resp)
			return
		}
	}

	var req genapi.EventIngestRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}
	if req.Source == "" {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "source is required"))
		return
	}
	if req.OccurredAt.IsZero() {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_field", "occurred_at is required"))
		return
	}

	now := time.Now().UTC()
	eventID := newID("evt")

	var resp genapi.AcceptedEvent
	txErr := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
		ev, err := Normalize(req, tenant, eventID, now)
		if err != nil {
			return err
		}

		existingEventID, duplicate, observableIDs, err := Insert(ctx, tx, ev)
		if err != nil {
			return err
		}

		if duplicate {
			outcome = "duplicate"
			alertID, _, err := alerts.FindByEventID(ctx, tx, existingEventID)
			if err != nil {
				return httpx.ErrInternal("dedup_lookup_failed", "could not resolve the existing alert for a duplicate event").WithCause(err)
			}
			resp = genapi.AcceptedEvent{
				EventId:   existingEventID,
				AlertId:   &alertID,
				RequestId: httpx.RequestID(c),
				Status:    genapi.AcceptedEventStatusDuplicate,
			}
			body, err := json.Marshal(resp)
			if err != nil {
				return err
			}
			return httpx.SaveIdempotency(ctx, tx, tenant, http.MethodPost, ingestRoute, key, bodyHash, http.StatusAccepted, body)
		}

		alertID := newID("alt")
		al := alerts.NewFromEvent(alertID, tenant, ev.ID, ev.Source, ev.Severity, ev.Confidence, now)
		if err := alerts.Insert(ctx, tx, al, observableIDs); err != nil {
			return err
		}
		if err := enrichment.LookupAll(ctx, tx, observableIDs); err != nil {
			return err
		}

		// adr/0018/adr/0021: an event whose class matches a registered
		// signature (services/api/signatures) attaches to the same
		// still-open case instead of each opening a new one — the
		// correlation step docs/03-domain-model already modeled
		// (cases.correlation_key) but M1 deliberately deferred. Every event
		// class with no matching signature keeps today's "always open a new
		// case" behavior.
		sig := signatures.MatchEventClass(ev.EventClass)
		correlationKey := correlationKeyFor(sig, ev.EventClass, ev.Observables)
		var caseID string
		var repeatCount int
		var existingCase *cases.Case
		if correlationKey != "" {
			existingCase, err = cases.FindOpenByCorrelation(ctx, tx, tenant, correlationKey, sig.WindowSeconds, now)
			if err != nil {
				return err
			}
		}
		var currentSeverity string
		if existingCase != nil {
			h.rec.CaseOutcome("correlated")
			caseID = existingCase.ID
			currentSeverity = existingCase.Severity
			repeatCount, err = cases.IncrementRepeatCount(ctx, tx, tenant, caseID, now)
			if err != nil {
				return err
			}
			if err := cases.AppendTimeline(ctx, tx, caseID, tenant, "alert_linked", "system", map[string]any{
				"alert_id": alertID, "event_id": ev.ID, "source": ev.Source, "repeat_count": repeatCount,
			}); err != nil {
				return err
			}
		} else {
			h.rec.CaseOutcome("created")
			caseID = newID("case")
			var windowPtr *int
			var keyPtr *string
			if correlationKey != "" {
				w := sig.WindowSeconds
				windowPtr = &w
				keyPtr = &correlationKey
			}
			cs := cases.NewFromAlert(caseID, tenant, ev.EventClass, ev.Severity, ev.Confidence, "system", now, keyPtr, windowPtr)
			if err := cases.Insert(ctx, tx, cs); err != nil {
				return err
			}
			repeatCount = cs.RepeatCount
			currentSeverity = cs.Severity
			if err := cases.AppendTimeline(ctx, tx, caseID, tenant, "case_opened", "system", map[string]any{
				"alert_id": alertID, "event_id": ev.ID, "source": ev.Source,
			}); err != nil {
				return err
			}
		}
		if err := cases.LinkAlert(ctx, tx, caseID, alertID); err != nil {
			return err
		}
		linkIDs := observableIDs
		if existingCase != nil {
			// Correlating onto a case that already has this exact
			// type:value observable — skip relinking it (see
			// ExistingObservableValues' comment for why this only matters
			// once a case can receive more than one event's observables).
			existingValues, err := cases.ExistingObservableValues(ctx, tx, caseID)
			if err != nil {
				return err
			}
			linkIDs = linkIDs[:0]
			for i, o := range ev.Observables {
				if !existingValues[o.Type+":"+o.Value] {
					linkIDs = append(linkIDs, observableIDs[i])
				}
			}
		}
		if err := cases.LinkObservables(ctx, tx, caseID, linkIDs); err != nil {
			return err
		}
		if err := alerts.SetCaseRef(ctx, tx, alertID, caseID); err != nil {
			return err
		}
		if _, err := evidence.RecordFromEvent(ctx, tx, tenant, caseID, ev.ID, ev.Source, ev.RawRef.URI, ev.ObservedAt); err != nil {
			return err
		}

		// Priority escalation (adr/0019): "Medium becomes higher priority
		// only when source targets a privileged user or critical asset" —
		// scoped to the same correlated signature as the auto-trigger below,
		// checked against any user/host observable this event carries.
		if correlationKey != "" {
			for _, o := range ev.Observables {
				if o.Type != "user" && o.Type != "host" {
					continue
				}
				critical, err := criticaltargets.IsCritical(ctx, tx, tenant, o.Value)
				if err != nil {
					return err
				}
				if critical && cases.SeverityRank["high"] > cases.SeverityRank[currentSeverity] {
					if err := cases.EscalateSeverity(ctx, tx, tenant, caseID, "high", now); err != nil {
						return err
					}
					if err := cases.AppendTimeline(ctx, tx, caseID, tenant, "priority_escalated", "system", map[string]any{
						"from": currentSeverity, "to": "high", "target": o.Value,
					}); err != nil {
						return err
					}
					currentSeverity = "high"
				}
			}
		}

		// Signature auto-trigger (adr/0018/adr/0021): once a registered
		// signature's threshold is hit for a correlated, non-allowlisted
		// source, request the block automatically instead of waiting for an
		// analyst to click "Request block" — everything downstream
		// (approval, execute, scheduled release) is the same already-built
		// path that manual click uses. Only signatures.Signature.ActionIntent
		// == "block_network_indicator" has a request path today
		// (actions.RequestBlock); no signature in the registry names
		// anything else yet.
		if correlationKey != "" && repeatCount >= sig.Threshold {
			sourceIP := correlationValueFor(sig, ev.Observables)
			trusted, err := allowlist.IsTrusted(ctx, tx, tenant, sourceIP, now)
			if err != nil {
				return err
			}
			if !trusted {
				a, _, created, err := actions.RequestBlock(ctx, tx, tenant, caseID, sig.ObservableType, sourceIP, "system")
				if err != nil {
					return err
				}
				if created {
					h.rec.ActionRequested(a.Intent)
					if err := cases.AppendTimeline(ctx, tx, caseID, tenant, "block_requested", "system", map[string]any{
						"action_id": a.ID, "target": sourceIP, "repeat_count": repeatCount,
					}); err != nil {
						return err
					}
					_, approvalRoles := actions.EvaluatePolicy(a.Intent, a.Risk)
					reason := "auto-triggered after " + fmt.Sprintf("%d", repeatCount) + " correlated events; requires approval from: " + strings.Join(approvalRoles, ", ")
					th := audit.TargetHash(sourceIP)
					if err := audit.Append(ctx, tx, audit.Record{
						TenantID: tenant, Actor: "soar-ingestion", ActorType: "system",
						CaseID: &caseID, Action: "action.request_block",
						TargetHash: &th, Decision: "completed", Reason: &reason,
					}); err != nil {
						return err
					}
				}
			}
		}
		requestID := httpx.RequestID(c)
		eventID := ev.ID
		targetHash := audit.TargetHash(ev.ID)
		if err := audit.Append(ctx, tx, audit.Record{
			TenantID:   tenant,
			EventID:    &eventID,
			RequestID:  &requestID,
			Actor:      "soar-ingestion",
			ActorType:  "system",
			CaseID:     &caseID,
			Action:     "event.ingest",
			TargetHash: &targetHash,
			Decision:   "completed",
		}); err != nil {
			return err
		}

		outcome = "accepted"
		resp = genapi.AcceptedEvent{
			EventId:   ev.ID,
			AlertId:   &alertID,
			RequestId: httpx.RequestID(c),
			Status:    genapi.AcceptedEventStatusAccepted,
		}
		body, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		return httpx.SaveIdempotency(ctx, tx, tenant, http.MethodPost, ingestRoute, key, bodyHash, http.StatusAccepted, body)
	})
	if txErr != nil {
		outcome = "error"
		httpx.WriteProblem(c, txErr)
		return
	}
	c.JSON(http.StatusAccepted, resp)
}

// List implements GET /v1/events.
func (h *Handler) List(c *gin.Context) {
	tenant := httpx.TenantID(c)
	limit := parseLimit(c.Query("limit"))
	cursor := c.Query("cursor")

	items, nextCursor, err := List(c.Request.Context(), h.pool, tenant, cursor, limit)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("list_events_failed", "could not list events").WithCause(err))
		return
	}

	page := gin.H{"items": items}
	if nextCursor != "" {
		page["next_cursor"] = nextCursor
	}
	c.JSON(http.StatusOK, page)
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

func newID(prefix string) string {
	return prefix + "-" + strings.ToLower(ulid.Make().String())
}
