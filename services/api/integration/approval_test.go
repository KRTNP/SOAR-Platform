//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/genapi"
)

// This suite is M3's named gate (docs/10-roadmap/milestones.md: "no-action-
// before-approval test"): the network connector's effect table must gain
// zero rows whenever execute is attempted without a valid, unexpired,
// approved approval.

func getCaseDetail(t *testing.T, srv *httptest.Server, caseID string) genapi.Case {
	t.Helper()
	resp, err := http.Get(srv.URL + "/v1/cases/" + caseID)
	if err != nil {
		t.Fatalf("getting case detail: %v", err)
	}
	defer resp.Body.Close()
	var detail genapi.Case
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatalf("decoding case detail: %v", err)
	}
	return detail
}

func requestBlock(t *testing.T, srv *httptest.Server, caseID, targetValue string) genapi.Action {
	t.Helper()
	body := fmt.Sprintf(`{"operation":"request_block","target_value":%q}`, targetValue)
	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/v1/cases/"+caseID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH request_block: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 requesting a block, got %d", resp.StatusCode)
	}

	detail := getCaseDetail(t, srv, caseID)
	if detail.PendingAction == nil {
		t.Fatal("expected a pending_action after request_block")
	}
	return *detail.PendingAction
}

func findApprovalForAction(t *testing.T, srv *httptest.Server, actionID string) genapi.Approval {
	t.Helper()
	resp, err := http.Get(srv.URL + "/v1/approvals")
	if err != nil {
		t.Fatalf("listing approvals: %v", err)
	}
	defer resp.Body.Close()
	var page struct {
		Items []genapi.Approval `json:"items"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&page)
	for _, a := range page.Items {
		if a.ActionId == actionID {
			return a
		}
	}
	t.Fatalf("no approval found for action %s", actionID)
	return genapi.Approval{}
}

func executeAction(t *testing.T, srv *httptest.Server, actionID string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/actions/"+actionID+"/execute", nil)
	req.Header.Set("Idempotency-Key", "exec-key-"+actionID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST execute: %v", err)
	}
	return resp
}

func approveApproval(t *testing.T, srv *httptest.Server, approvalID string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/approvals/"+approvalID+"/approve", nil)
	req.Header.Set("Idempotency-Key", "approve-key-"+approvalID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST approve: %v", err)
	}
	return resp
}

func rejectApproval(t *testing.T, srv *httptest.Server, approvalID, reason string) *http.Response {
	t.Helper()
	body := fmt.Sprintf(`{"reason":%q}`, reason)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/approvals/"+approvalID+"/reject", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "reject-key-"+approvalID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST reject: %v", err)
	}
	return resp
}

// caseWithBlockTarget ingests an event carrying an ipv4 observable and
// returns its case id, so requestBlock has an observable to target.
func caseWithBlockTarget(t *testing.T, srv *httptest.Server, sourceEventID string) string {
	t.Helper()
	_, accepted := postEvent(t, srv, "idem-key-"+sourceEventID, fixtureWithObservables(t, sourceEventID))
	if accepted.AlertId == nil {
		t.Fatal("expected an alert_id from ingestion")
	}
	page := getCases(t, srv)
	return page[len(page)-1].Id
}

func getCases(t *testing.T, srv *httptest.Server) []genapi.Case {
	t.Helper()
	resp, err := http.Get(srv.URL + "/v1/cases")
	if err != nil {
		t.Fatalf("listing cases: %v", err)
	}
	defer resp.Body.Close()
	var page struct {
		Items []genapi.Case `json:"items"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&page)
	return page.Items
}

func TestExecute_DeniedWithoutApproval(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "approval-noapproval-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")

	before := countRows(t, pool, "network_blocks")
	resp := executeAction(t, srv, action.Id)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 executing without approval, got %d", resp.StatusCode)
	}
	if after := countRows(t, pool, "network_blocks"); after != before {
		t.Errorf("expected no network_blocks row from a denied execute, before=%d after=%d", before, after)
	}
}

func TestExecute_DeniedOnRejectedApproval(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "approval-rejected-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approval := findApprovalForAction(t, srv, action.Id)

	rejectResp := rejectApproval(t, srv, approval.Id, "not a real threat")
	rejectResp.Body.Close()
	if rejectResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 rejecting, got %d", rejectResp.StatusCode)
	}

	before := countRows(t, pool, "network_blocks")
	resp := executeAction(t, srv, action.Id)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 executing a rejected action, got %d", resp.StatusCode)
	}
	if after := countRows(t, pool, "network_blocks"); after != before {
		t.Errorf("expected no network_blocks row from a rejected-approval execute, before=%d after=%d", before, after)
	}
}

func TestExecute_DeniedOnExpiredApproval(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "approval-expired-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approval := findApprovalForAction(t, srv, action.Id)

	// No clock injection exists yet, so force expiry directly for this test.
	if _, err := pool.Exec(context.Background(),
		`update approvals set expires_at = now() - interval '1 minute' where id = $1`, approval.Id,
	); err != nil {
		t.Fatalf("forcing approval expiry: %v", err)
	}

	before := countRows(t, pool, "network_blocks")
	resp := executeAction(t, srv, action.Id)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 executing an expired approval, got %d", resp.StatusCode)
	}
	if after := countRows(t, pool, "network_blocks"); after != before {
		t.Errorf("expected no network_blocks row from an expired-approval execute, before=%d after=%d", before, after)
	}

	getResp, err := http.Get(srv.URL + "/v1/actions/" + action.Id)
	if err != nil {
		t.Fatalf("getting action: %v", err)
	}
	defer getResp.Body.Close()
	var got genapi.Action
	_ = json.NewDecoder(getResp.Body).Decode(&got)
	if got.Status != genapi.ActionStatusExpired {
		t.Errorf("expected action status expired after a denied execute past TTL, got %q", got.Status)
	}
}

func TestApprove_IsIdempotent(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "approval-idempotent-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approval := findApprovalForAction(t, srv, action.Id)

	first := approveApproval(t, srv, approval.Id)
	var firstBody genapi.Approval
	_ = json.NewDecoder(first.Body).Decode(&firstBody)
	first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on first approve, got %d", first.StatusCode)
	}

	second := approveApproval(t, srv, approval.Id)
	var secondBody genapi.Approval
	_ = json.NewDecoder(second.Body).Decode(&secondBody)
	second.Body.Close()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on repeat approve, got %d", second.StatusCode)
	}
	if firstBody.DecidedAt == nil || secondBody.DecidedAt == nil || !firstBody.DecidedAt.Equal(*secondBody.DecidedAt) {
		t.Errorf("expected the same decided_at on a repeat approve, got %v vs %v", firstBody.DecidedAt, secondBody.DecidedAt)
	}

	// Executing after the double-approve must still only produce one block.
	execResp := executeAction(t, srv, action.Id)
	execResp.Body.Close()
	if execResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 executing an approved action, got %d", execResp.StatusCode)
	}
	execResp2 := executeAction(t, srv, action.Id)
	execResp2.Body.Close()
	if execResp2.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 on a repeat execute, got %d", execResp2.StatusCode)
	}
	if n := countRows(t, pool, "network_blocks"); n != 1 {
		t.Errorf("expected exactly 1 network_blocks row after a double-approve and double-execute, got %d", n)
	}
}

func TestExecute_SucceedsAfterApproval(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "approval-success-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approval := findApprovalForAction(t, srv, action.Id)

	approveResp := approveApproval(t, srv, approval.Id)
	approveResp.Body.Close()
	if approveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 approving, got %d", approveResp.StatusCode)
	}

	before := countRows(t, pool, "network_blocks")
	execResp := executeAction(t, srv, action.Id)
	defer execResp.Body.Close()
	if execResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 executing an approved action, got %d", execResp.StatusCode)
	}
	if after := countRows(t, pool, "network_blocks"); after != before+1 {
		t.Errorf("expected exactly one new network_blocks row, before=%d after=%d", before, after)
	}

	detail := getCaseDetail(t, srv, caseID)
	if detail.PendingAction == nil || detail.PendingAction.Status != genapi.ActionStatusExecuted {
		t.Errorf("expected pending_action status executed, got %v", detail.PendingAction)
	}
}

func TestRelease_RollsBackBlock(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "approval-release-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approval := findApprovalForAction(t, srv, action.Id)

	approveApproval(t, srv, approval.Id).Body.Close()
	executeAction(t, srv, action.Id).Body.Close()

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	released, err := actions.Release(ctx, tx, "lab", action.Id)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if released.Status != "released" {
		t.Errorf("expected status released, got %q", released.Status)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var blockStatus string
	if err := pool.QueryRow(ctx, `select status from network_blocks where action_id = $1`, action.Id).Scan(&blockStatus); err != nil {
		t.Fatalf("checking network_blocks status: %v", err)
	}
	if blockStatus != "released" {
		t.Errorf("expected network_blocks status released, got %q", blockStatus)
	}
}
