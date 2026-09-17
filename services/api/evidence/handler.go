package evidence

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/store"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// CreateManifest implements POST /v1/evidence/manifests.
func (h *Handler) CreateManifest(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	if len(c.GetHeader("Idempotency-Key")) < httpx.MinIdempotencyKeyLength {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_idempotency_key", "Idempotency-Key header must be at least 8 characters"))
		return
	}

	var req genapi.EvidenceManifestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}

	m := ManualEvidence{
		CaseID:            req.CaseId,
		ArtifactType:      req.ArtifactType,
		URI:               req.Uri,
		SHA256:            req.Sha256,
		Size:              req.Size,
		Source:            req.Source,
		AcquisitionMethod: req.AcquisitionMethod,
		Collector:         req.Collector,
		CollectedAt:       req.CollectedAt,
		ToolName:          req.ToolName,
		ToolVersion:       req.ToolVersion,
		CommandRef:        req.CommandRef,
		Analyst:           req.Analyst,
		Purpose:           req.Purpose,
		Sensitivity:       string(req.Sensitivity),
		RetentionClass:    req.RetentionClass,
	}
	if req.Limitations != nil {
		m.Limitations = *req.Limitations
	}
	if req.DerivedFrom != nil {
		m.DerivedFrom = *req.DerivedFrom
	}

	var ev Evidence
	err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `select exists(select 1 from cases where tenant_id=$1 and id=$2)`, tenant, m.CaseID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return httpx.ErrValidation("case_not_found", "case_id does not reference an existing case")
		}
		var registerErr error
		ev, registerErr = Register(ctx, tx, tenant, m)
		return registerErr
	})
	if err != nil {
		httpx.WriteProblem(c, mapError(err))
		return
	}
	c.JSON(http.StatusCreated, ToAPI(ev))
}

// Get implements GET /v1/evidence/{evidenceId} — the M4 gate: it re-verifies
// the recorded hash against the linked event's live payload rather than
// trusting the stored integrity_status.
func (h *Handler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)

	var ev *Evidence
	err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
		var getErr error
		ev, getErr = Get(ctx, tx, tenant, c.Param("evidenceId"))
		return getErr
	})
	if err != nil {
		httpx.WriteProblem(c, mapError(err))
		return
	}
	c.JSON(http.StatusOK, ToAPI(*ev))
}

func mapError(err error) error {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		return err
	}
	if errors.Is(err, ErrNotFound) {
		return httpx.ErrNotFound("evidence_not_found", "evidence does not exist")
	}
	if errors.Is(err, ErrValidation) {
		return httpx.ErrValidation("invalid_evidence_manifest", strings.TrimPrefix(err.Error(), ErrValidation.Error()+": "))
	}
	return httpx.ErrInternal("evidence_operation_failed", "evidence operation failed").WithCause(err)
}

func ToAPI(ev Evidence) genapi.Evidence {
	out := genapi.Evidence{
		Id:                ev.ID,
		CaseId:            ev.CaseID,
		ArtifactType:      ev.ArtifactType,
		Uri:               ev.URI,
		Sha256:            ev.SHA256,
		Size:              ev.Size,
		Source:            ev.Source,
		AcquisitionMethod: ev.AcquisitionMethod,
		Collector:         ev.Collector,
		CollectedAt:       ev.CollectedAt,
		CreatedAt:         ev.CreatedAt,
		ReceivedAt:        ev.ReceivedAt,
		Timezone:          ev.Timezone,
		ToolName:          ev.ToolName,
		ToolVersion:       ev.ToolVersion,
		CommandRef:        ev.CommandRef,
		Analyst:           ev.Analyst,
		Purpose:           ev.Purpose,
		Sensitivity:       genapi.EvidenceSensitivity(ev.Sensitivity),
		RetentionClass:    ev.RetentionClass,
		IntegrityStatus:   genapi.EvidenceIntegrityStatus(ev.IntegrityStatus),
	}
	if len(ev.Limitations) > 0 {
		out.Limitations = &ev.Limitations
	}
	if len(ev.DerivedFrom) > 0 {
		out.DerivedFrom = &ev.DerivedFrom
	}
	if len(ev.CustodyEvents) > 0 {
		items := make([]genapi.EvidenceCustodyEvent, 0, len(ev.CustodyEvents))
		for _, ce := range ev.CustodyEvents {
			items = append(items, genapi.EvidenceCustodyEvent{
				Event: ce.Event, Actor: ce.Actor, At: ce.At, FromHash: ce.FromHash, ToHash: ce.ToHash,
			})
		}
		out.CustodyEvents = &items
	}
	return out
}

// ToRedactedMap renders one evidence item as the entry cases.BuildManifest
// embeds per item. It is exported so the cases package can build the redacted
// export view without duplicating these field names, and `full` selects
// between the redacted set and the complete one.
//
// (The previous comment here named a RedactedManifestEntry type that no longer
// exists anywhere in the module, so this function was in effect undocumented.)
func ToRedactedMap(ev Evidence, full bool) map[string]any {
	m := map[string]any{
		"id":               ev.ID,
		"artifact_type":    ev.ArtifactType,
		"sha256":           ev.SHA256,
		"size":             ev.Size,
		"source":           ev.Source,
		"sensitivity":      ev.Sensitivity,
		"retention_class":  ev.RetentionClass,
		"integrity_status": ev.IntegrityStatus,
		"collected_at":     ev.CollectedAt,
	}
	if full {
		m["uri"] = ev.URI
		m["command_ref"] = ev.CommandRef
	}
	return m
}
