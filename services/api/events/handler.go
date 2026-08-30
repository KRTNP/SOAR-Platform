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

	"soar-platform/services/api/alerts"
	"soar-platform/services/api/audit"
	"soar-platform/services/api/cases"
	"soar-platform/services/api/enrichment"
	"soar-platform/services/api/evidence"
	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/store"
)

const ingestRoute = "ingestEvent"

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// Ingest implements POST /v1/events: normalize -> dedup -> alert -> case, all
// in one transaction, guarded by an Idempotency-Key.
func (h *Handler) Ingest(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)

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

		caseID := newID("case")
		cs := cases.NewFromAlert(caseID, tenant, ev.EventClass, ev.Severity, ev.Confidence, "system", now)
		if err := cases.Insert(ctx, tx, cs); err != nil {
			return err
		}
		if err := cases.LinkAlert(ctx, tx, caseID, alertID); err != nil {
			return err
		}
		if err := cases.LinkObservables(ctx, tx, caseID, observableIDs); err != nil {
			return err
		}
		if err := alerts.SetCaseRef(ctx, tx, alertID, caseID); err != nil {
			return err
		}
		if err := cases.AppendTimeline(ctx, tx, caseID, tenant, "case_opened", "system", map[string]any{
			"alert_id": alertID, "event_id": ev.ID, "source": ev.Source,
		}); err != nil {
			return err
		}
		if _, err := evidence.RecordFromEvent(ctx, tx, tenant, caseID, ev.ID, ev.Source, ev.RawRef.URI, ev.ObservedAt); err != nil {
			return err
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
