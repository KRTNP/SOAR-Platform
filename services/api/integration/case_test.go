//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"soar-platform/services/api/genapi"
)

func fixtureWithObservables(t *testing.T, sourceEventID string) []byte {
	t.Helper()
	payload := map[string]any{
		"source_event_id": sourceEventID,
		"event_class":     "authentication",
		"severity":        "medium",
		"confidence":      0.82,
		"observables": []any{
			map[string]any{
				"type": "ipv4", "value": "198.51.100.10", "normalization": "rfc5737-test-net",
				"first_seen": "2026-08-30T04:00:00Z", "last_seen": "2026-08-30T04:00:00Z",
				"confidence": 0.82, "sensitivity": "internal", "source_refs": []any{sourceEventID},
			},
		},
	}
	body := map[string]any{"source": "wazuh", "occurred_at": "2026-08-30T04:00:00Z", "payload": payload}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling fixture: %v", err)
	}
	return b
}

// TestCaseDetail_IncludesObservables proves ADR-0009's Case response
// extension actually round-trips through the real API, not just the DB.
func TestCaseDetail_IncludesObservables(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	_, accepted := postEvent(t, srv, "idem-key-obs-0001", fixtureWithObservables(t, "obs-test-0001"))
	if accepted.AlertId == nil {
		t.Fatal("expected an alert_id from ingestion")
	}

	resp, err := http.Get(srv.URL + "/v1/cases")
	if err != nil {
		t.Fatalf("listing cases: %v", err)
	}
	var page struct {
		Items []genapi.Case `json:"items"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&page)
	resp.Body.Close()
	if len(page.Items) == 0 {
		t.Fatal("expected at least one case")
	}
	caseID := page.Items[len(page.Items)-1].Id

	detailResp, err := http.Get(srv.URL + "/v1/cases/" + caseID)
	if err != nil {
		t.Fatalf("getting case detail: %v", err)
	}
	defer detailResp.Body.Close()
	var detail genapi.Case
	if err := json.NewDecoder(detailResp.Body).Decode(&detail); err != nil {
		t.Fatalf("decoding case detail: %v", err)
	}

	if detail.Observables == nil || len(*detail.Observables) != 1 {
		t.Fatalf("expected exactly 1 observable on the case detail, got %+v", detail.Observables)
	}
	if (*detail.Observables)[0].Value != "198.51.100.10" {
		t.Errorf("expected observable value 198.51.100.10, got %q", (*detail.Observables)[0].Value)
	}
}

// TestCaseClose proves the "close" PATCH operation end-to-end: it requires a
// closure_reason, sets status/closed_at, and records a case_closed timeline
// entry.
func TestCaseClose(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	_, accepted := postEvent(t, srv, "idem-key-close-0001", fixture(t, "close-test-0001"))

	var caseID string
	err := pool.QueryRow(context.Background(),
		"select case_ref from alerts where id = $1", *accepted.AlertId,
	).Scan(&caseID)
	if err != nil {
		t.Fatalf("looking up case for alert: %v", err)
	}

	// Missing closure_reason must be rejected, not silently accepted.
	badReq, _ := http.NewRequest(http.MethodPatch, srv.URL+"/v1/cases/"+caseID, strings.NewReader(`{"operation":"close"}`))
	badReq.Header.Set("Content-Type", "application/json")
	badResp, err := http.DefaultClient.Do(badReq)
	if err != nil {
		t.Fatalf("PATCH without closure_reason: %v", err)
	}
	badResp.Body.Close()
	if badResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for a close without closure_reason, got %d", badResp.StatusCode)
	}

	closeReq, _ := http.NewRequest(http.MethodPatch, srv.URL+"/v1/cases/"+caseID,
		strings.NewReader(`{"operation":"close","closure_reason":"false positive, trusted source"}`))
	closeReq.Header.Set("Content-Type", "application/json")
	closeResp, err := http.DefaultClient.Do(closeReq)
	if err != nil {
		t.Fatalf("PATCH close: %v", err)
	}
	defer closeResp.Body.Close()
	if closeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 closing the case, got %d", closeResp.StatusCode)
	}
	var closed genapi.Case
	_ = json.NewDecoder(closeResp.Body).Decode(&closed)
	if closed.Status != genapi.CaseStatusClosed {
		t.Errorf("expected status closed, got %q", closed.Status)
	}
	if closed.ClosedAt == nil {
		t.Error("expected closed_at to be set")
	}
	if closed.ClosureReason == nil || *closed.ClosureReason != "false positive, trusted source" {
		t.Errorf("expected closure_reason to be recorded, got %v", closed.ClosureReason)
	}

	timelineResp, err := http.Get(srv.URL + "/v1/cases/" + caseID + "/timeline")
	if err != nil {
		t.Fatalf("getting timeline: %v", err)
	}
	defer timelineResp.Body.Close()
	var timeline struct {
		Items []struct {
			Kind string `json:"kind"`
		} `json:"items"`
	}
	_ = json.NewDecoder(timelineResp.Body).Decode(&timeline)
	found := false
	for _, e := range timeline.Items {
		if e.Kind == "case_closed" {
			found = true
		}
	}
	if !found {
		t.Error("expected a case_closed timeline entry after closing")
	}
}

// TestCaseTransition_RejectsDirectClose proves an analyst cannot bypass the
// mandatory closure_reason by transitioning status straight to "closed"
// instead of using the dedicated close operation.
func TestCaseTransition_RejectsDirectClose(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	_, accepted := postEvent(t, srv, "idem-key-noclose-0001", fixture(t, "noclose-test-0001"))

	var caseID string
	err := pool.QueryRow(context.Background(),
		"select case_ref from alerts where id = $1", *accepted.AlertId,
	).Scan(&caseID)
	if err != nil {
		t.Fatalf("looking up case for alert: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/v1/cases/"+caseID,
		strings.NewReader(`{"operation":"transition","status":"closed"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH transition to closed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 rejecting a direct transition to closed, got %d", resp.StatusCode)
	}

	detailResp, err := http.Get(srv.URL + "/v1/cases/" + caseID)
	if err != nil {
		t.Fatalf("getting case detail: %v", err)
	}
	defer detailResp.Body.Close()
	var detail genapi.Case
	_ = json.NewDecoder(detailResp.Body).Decode(&detail)
	if detail.Status == genapi.CaseStatusClosed {
		t.Error("case must not have been closed by the rejected transition")
	}
}
