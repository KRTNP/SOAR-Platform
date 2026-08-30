//go:build integration

package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"soar-platform/services/api/genapi"
)

// This suite is M4's named gate (docs/10-roadmap/milestones.md: "hash/
// redaction tests"): evidence.Get must re-verify (not trust) its recorded
// hash, and case export must actually redact per profile and report a hash
// that matches its own returned content.

func getCaseEvidence(t *testing.T, srv *httptest.Server, caseID string) []genapi.Evidence {
	t.Helper()
	resp, err := http.Get(srv.URL + "/v1/cases/" + caseID + "/evidence")
	if err != nil {
		t.Fatalf("listing case evidence: %v", err)
	}
	defer resp.Body.Close()
	var page struct {
		Items []genapi.Evidence `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decoding case evidence: %v", err)
	}
	return page.Items
}

func getEvidence(t *testing.T, srv *httptest.Server, id string) (int, genapi.Evidence) {
	t.Helper()
	resp, err := http.Get(srv.URL + "/v1/evidence/" + id)
	if err != nil {
		t.Fatalf("getting evidence: %v", err)
	}
	defer resp.Body.Close()
	var ev genapi.Evidence
	_ = json.NewDecoder(resp.Body).Decode(&ev)
	return resp.StatusCode, ev
}

func exportCase(t *testing.T, srv *httptest.Server, caseID, body string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/cases/"+caseID+"/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "export-key-"+caseID+"-"+body)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST export: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func getAudit(t *testing.T, srv *httptest.Server) []map[string]any {
	t.Helper()
	resp, err := http.Get(srv.URL + "/v1/audit")
	if err != nil {
		t.Fatalf("listing audit: %v", err)
	}
	defer resp.Body.Close()
	var page struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decoding audit page: %v", err)
	}
	return page.Items
}

func findAuditAction(items []map[string]any, action, decision string) map[string]any {
	for _, r := range items {
		if r["action"] == action && r["decision"] == decision {
			return r
		}
	}
	return nil
}

func TestEvidence_AutoCapturedOnIngest(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "evidence-autocapture-0001")

	items := getCaseEvidence(t, srv, caseID)
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 evidence entry after ingest, got %d", len(items))
	}
	ev := items[0]
	if ev.ArtifactType != "captured_event_payload" {
		t.Errorf("artifact_type = %q, want captured_event_payload", ev.ArtifactType)
	}
	if ev.Source != "wazuh" {
		t.Errorf("source = %q, want wazuh", ev.Source)
	}
	if ev.IntegrityStatus != genapi.EvidenceIntegrityStatusVerified {
		t.Errorf("integrity_status = %q, want verified", ev.IntegrityStatus)
	}
}

func TestEvidence_DetectsTamperedPayload(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "evidence-tamper-0001")
	items := getCaseEvidence(t, srv, caseID)
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 evidence entry, got %d", len(items))
	}
	id := items[0].Id

	status, ev := getEvidence(t, srv, id)
	if status != http.StatusOK || ev.IntegrityStatus != genapi.EvidenceIntegrityStatusVerified {
		t.Fatalf("expected verified before tampering, got status=%d integrity=%q", status, ev.IntegrityStatus)
	}

	if _, err := pool.Exec(context.Background(),
		`update events set raw_payload = '{"tampered":true}'::jsonb where id = (select source_event_id from evidence where id = $1)`,
		id,
	); err != nil {
		t.Fatalf("tampering with event payload: %v", err)
	}

	status, ev = getEvidence(t, srv, id)
	if status != http.StatusOK {
		t.Fatalf("expected 200 after tampering, got %d", status)
	}
	if ev.IntegrityStatus != genapi.EvidenceIntegrityStatusMismatch {
		t.Errorf("expected integrity_status mismatch after tampering, got %q", ev.IntegrityStatus)
	}

	// Persisted, not just returned once.
	status, ev = getEvidence(t, srv, id)
	if status != http.StatusOK || ev.IntegrityStatus != genapi.EvidenceIntegrityStatusMismatch {
		t.Errorf("expected mismatch to persist across a second GET, got status=%d integrity=%q", status, ev.IntegrityStatus)
	}
}

func TestEvidence_ManualManifestRequiresFields(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "evidence-manual-0001")

	body := `{"case_id":"` + caseID + `","artifact_type":"memory_image"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/evidence/manifests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "manual-evidence-key-0001")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST evidence manifest: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for a manifest missing required fields, got %d", resp.StatusCode)
	}
}

func TestExport_StandardProfileRedactsEvidenceReferences(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "export-standard-0001")

	status, out := exportCase(t, srv, caseID, `{"format":"json"}`)
	if status != http.StatusAccepted {
		t.Fatalf("expected 202 exporting, got %d", status)
	}
	manifest, _ := out["manifest"].(map[string]any)
	evList, _ := manifest["evidence"].([]any)
	if len(evList) != 1 {
		t.Fatalf("expected 1 evidence entry in manifest, got %d", len(evList))
	}
	entry := evList[0].(map[string]any)
	if _, hasURI := entry["uri"]; hasURI {
		t.Errorf("standard profile leaked uri: %v", entry["uri"])
	}
	if _, hasCommandRef := entry["command_ref"]; hasCommandRef {
		t.Errorf("standard profile leaked command_ref: %v", entry["command_ref"])
	}
}

func TestExport_FullProfileIncludesEvidenceReferences(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "export-full-0001")

	status, out := exportCase(t, srv, caseID, `{"format":"json","redaction_profile":"full"}`)
	if status != http.StatusAccepted {
		t.Fatalf("expected 202 exporting, got %d", status)
	}
	manifest, _ := out["manifest"].(map[string]any)
	evList, _ := manifest["evidence"].([]any)
	if len(evList) != 1 {
		t.Fatalf("expected 1 evidence entry in manifest, got %d", len(evList))
	}
	entry := evList[0].(map[string]any)
	uri, _ := entry["uri"].(string)
	if uri == "" {
		t.Fatal("full profile should include uri, got empty")
	}

	var wantURI string
	if err := pool.QueryRow(context.Background(), `select raw_ref_uri from events e join evidence ev on ev.source_event_id = e.id where ev.case_id = $1`, caseID).Scan(&wantURI); err != nil {
		t.Fatalf("reading expected raw_ref_uri: %v", err)
	}
	if uri != wantURI {
		t.Errorf("uri = %q, want %q", uri, wantURI)
	}
}

func manifestSHA256(t *testing.T, out map[string]any) string {
	t.Helper()
	manifest := out["manifest"]
	b, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("re-marshaling manifest: %v", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestExport_ManifestHashMatchesContent(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "export-hash-0001")

	statusStd, outStd := exportCase(t, srv, caseID, `{"format":"json"}`)
	statusFull, outFull := exportCase(t, srv, caseID, `{"format":"json","redaction_profile":"full"}`)
	if statusStd != http.StatusAccepted || statusFull != http.StatusAccepted {
		t.Fatalf("expected 202 for both exports, got std=%d full=%d", statusStd, statusFull)
	}

	stdHash, _ := outStd["manifest_sha256"].(string)
	fullHash, _ := outFull["manifest_sha256"].(string)
	if stdHash == "" || fullHash == "" {
		t.Fatalf("expected non-empty manifest_sha256 for both exports, got std=%q full=%q", stdHash, fullHash)
	}
	if stdHash != manifestSHA256(t, outStd) {
		t.Errorf("standard export's manifest_sha256 does not match its own content")
	}
	if fullHash != manifestSHA256(t, outFull) {
		t.Errorf("full export's manifest_sha256 does not match its own content")
	}
	if stdHash == fullHash {
		t.Errorf("expected different hashes for standard vs full redaction profiles, got the same: %s", stdHash)
	}
}

func TestExport_OmitsEvidenceWhenNotRequested(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "export-omit-0001")

	status, out := exportCase(t, srv, caseID, `{"format":"json","include_evidence_references":false}`)
	if status != http.StatusAccepted {
		t.Fatalf("expected 202 exporting, got %d", status)
	}
	manifest, _ := out["manifest"].(map[string]any)
	if _, ok := manifest["evidence"]; ok {
		t.Errorf("expected no evidence key when include_evidence_references is false, got %v", manifest["evidence"])
	}
}

func TestExport_RejectsCsvFormat(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "export-csv-0001")

	status, _ := exportCase(t, srv, caseID, `{"format":"csv"}`)
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for csv format, got %d", status)
	}
}

func TestAudit_RecordsLifecycleActions(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "audit-lifecycle-0001")

	transitionBody := `{"operation":"transition","status":"triage"}`
	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/v1/cases/"+caseID, strings.NewReader(transitionBody))
	req.Header.Set("Content-Type", "application/json")
	transResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH transition: %v", err)
	}
	transResp.Body.Close()
	if transResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 transitioning, got %d", transResp.StatusCode)
	}

	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approval := findApprovalForAction(t, srv, action.Id)
	approveApproval(t, srv, approval.Id).Body.Close()
	executeAction(t, srv, action.Id).Body.Close()

	items := getAudit(t, srv)
	for _, want := range []struct{ action, decision string }{
		{"event.ingest", "completed"},
		{"case.transition", "completed"},
		{"action.request_block", "completed"},
		{"approval.approve", "completed"},
		{"action.execute", "allowed"},
	} {
		r := findAuditAction(items, want.action, want.decision)
		if r == nil {
			t.Errorf("expected an audit record for action=%q decision=%q, none found", want.action, want.decision)
			continue
		}
		if verified, _ := r["verified"].(bool); !verified {
			t.Errorf("audit record %+v did not verify its own result_hash", r)
		}
		if targetHash, _ := r["target_hash"].(string); targetHash == "198.51.100.10" {
			t.Errorf("audit record leaked the raw target value instead of its hash: %+v", r)
		}
	}
}

func TestAudit_RecordsDeniedExecuteAttempt(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "audit-denied-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")

	before := countRows(t, pool, "network_blocks")
	resp := executeAction(t, srv, action.Id)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 executing without approval, got %d", resp.StatusCode)
	}
	if after := countRows(t, pool, "network_blocks"); after != before {
		t.Fatalf("expected no network_blocks row from a denied execute, before=%d after=%d", before, after)
	}

	items := getAudit(t, srv)
	r := findAuditAction(items, "action.execute", "denied")
	if r == nil {
		t.Fatalf("expected a denied action.execute audit record even though its transaction rolled back")
	}
	if reason, _ := r["reason"].(string); reason != "approval_required" {
		t.Errorf("expected reason=approval_required, got %q", reason)
	}
}
