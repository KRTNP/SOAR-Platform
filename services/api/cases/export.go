package cases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/oklog/ulid/v2"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/audit"
	"soar-platform/services/api/evidence"
	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/store"
	"soar-platform/services/api/ticketing"
)

// BuildManifest assembles the redacted case export manifest described in
// adr/0012-evidence-integrity-and-audit-export.md: "standard" (default)
// strips each evidence entry's uri/command_ref (internal system references);
// "full" includes them. include_evidence_references (already an existing
// request field) omits the evidence section entirely when false. Marshaling
// a map[string]any is deterministic — encoding/json always sorts map keys —
// so manifestSha256 is reproducible for the same inputs.
func BuildManifest(ctx context.Context, tx pgx.Tx, tenantID, caseID string, req genapi.CaseExportRequest) (manifest map[string]any, manifestSha256 string, profile string, err error) {
	profile = "standard"
	if req.RedactionProfile != nil && string(*req.RedactionProfile) != "" {
		profile = string(*req.RedactionProfile)
	}
	includeEvidence := req.IncludeEvidenceReferences == nil || *req.IncludeEvidenceReferences

	cs, err := Get(ctx, tx, tenantID, caseID)
	if err != nil {
		return nil, "", "", err
	}
	observables, err := GetObservables(ctx, tx, tenantID, caseID)
	if err != nil {
		return nil, "", "", err
	}
	ticket, err := ticketing.Get(ctx, tx, tenantID, caseID)
	if err != nil {
		return nil, "", "", err
	}
	action, approval, err := actions.GetLatestForCase(ctx, tx, tenantID, caseID)
	if err != nil {
		return nil, "", "", err
	}

	manifest = map[string]any{
		"case":              toAPICaseDetail(cs, observables, ticket, action, approval),
		"redaction_profile": profile,
		"exported_at":       time.Now().UTC(),
	}

	if includeEvidence {
		items, err := evidence.ListForCase(ctx, tx, tenantID, caseID)
		if err != nil {
			return nil, "", "", err
		}
		redacted := make([]map[string]any, 0, len(items))
		for _, ev := range items {
			redacted = append(redacted, evidence.ToRedactedMap(ev, profile == "full"))
		}
		manifest["evidence"] = redacted
	}

	bytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, "", "", err
	}
	sum := sha256.Sum256(bytes)
	return manifest, hex.EncodeToString(sum[:]), profile, nil
}

// Export implements POST /v1/cases/{caseId}/export. There is no export job
// queue yet (same "no Temporal" simplification as M1-M3), so the redacted
// manifest is built and returned synchronously.
func (h *Handler) Export(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := httpx.TenantID(c)
	caseID := c.Param("caseId")
	if len(c.GetHeader("Idempotency-Key")) < httpx.MinIdempotencyKeyLength {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_idempotency_key", "Idempotency-Key header must be at least 8 characters"))
		return
	}

	var req genapi.CaseExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_json", "request body is not valid JSON").WithCause(err))
		return
	}
	if req.Format == "csv" {
		httpx.WriteProblem(c, httpx.ErrUnsupported("csv_export_not_supported", "csv export is deferred; use format=json"))
		return
	}
	if req.Format != "json" {
		httpx.WriteProblem(c, httpx.ErrValidation("invalid_enum", "format must be json"))
		return
	}

	var manifest map[string]any
	var manifestSha256, profile string
	err := store.WithTx(ctx, h.pool, func(tx pgx.Tx) error {
		var buildErr error
		manifest, manifestSha256, profile, buildErr = BuildManifest(ctx, tx, tenant, caseID, req)
		if buildErr != nil {
			return buildErr
		}
		reason := exportReason(req)
		if err := AppendTimeline(ctx, tx, caseID, tenant, "case_exported", "analyst", map[string]any{
			"redaction_profile": profile, "manifest_sha256": manifestSha256,
		}); err != nil {
			return err
		}
		return auditAppend(ctx, tx, tenant, &caseID, "analyst", "analyst", "case.export", "completed", audit.TargetHash(caseID), &reason)
	})
	if err != nil {
		httpx.WriteProblem(c, mapCaseError(err))
		return
	}

	exportID := "exp-" + strings.ToLower(ulid.Make().String())
	c.JSON(http.StatusAccepted, genapi.CaseExportAccepted{
		ExportId:         exportID,
		Status:           genapi.CaseExportAcceptedStatusCompleted,
		ManifestSha256:   &manifestSha256,
		RedactionProfile: &profile,
		Manifest:         &manifest,
	})
}

func exportReason(req genapi.CaseExportRequest) string {
	purpose := "unspecified"
	if req.Purpose != nil && strings.TrimSpace(*req.Purpose) != "" {
		purpose = *req.Purpose
	}
	recipient := "unspecified"
	if req.Recipient != nil && strings.TrimSpace(*req.Recipient) != "" {
		recipient = *req.Recipient
	}
	return "purpose=" + purpose + "; recipient=" + recipient
}
