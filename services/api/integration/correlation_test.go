//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"soar-platform/services/api/genapi"
)

// authFixture is fixtureWithObservables (case_test.go) with a caller-chosen
// source IP instead of a fixed one, so a test can send several events that
// correlate onto the same case (adr/0018) by sharing an IP, or deliberately
// not correlate by using different ones.
func authFixture(t *testing.T, sourceEventID, sourceIP string) []byte {
	t.Helper()
	payload := map[string]any{
		"source_event_id": sourceEventID,
		"event_class":     "authentication",
		"severity":        "medium",
		"confidence":      0.82,
		"observables": []any{
			map[string]any{
				"type": "ipv4", "value": sourceIP, "normalization": "rfc5737-test-net",
				"first_seen": "2026-09-02T04:00:00Z", "last_seen": "2026-09-02T04:00:00Z",
				"confidence": 0.82, "sensitivity": "internal", "source_refs": []any{sourceEventID},
			},
		},
	}
	body := map[string]any{"source": "wazuh", "occurred_at": "2026-09-02T04:00:00Z", "payload": payload}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling fixture: %v", err)
	}
	return b
}

// authFixtureWithUser is authFixture plus a "user" observable, for
// adr/0019's priority-escalation check (which only fires for a user/host
// observable, never the IP alone).
func authFixtureWithUser(t *testing.T, sourceEventID, sourceIP, user string) []byte {
	t.Helper()
	payload := map[string]any{
		"source_event_id": sourceEventID,
		"event_class":     "authentication",
		"severity":        "medium",
		"confidence":      0.82,
		"observables": []any{
			map[string]any{
				"type": "ipv4", "value": sourceIP, "normalization": "rfc5737-test-net",
				"first_seen": "2026-09-02T04:00:00Z", "last_seen": "2026-09-02T04:00:00Z",
				"confidence": 0.82, "sensitivity": "internal", "source_refs": []any{sourceEventID},
			},
			map[string]any{
				"type": "user", "value": user, "normalization": "lowercase-username",
				"first_seen": "2026-09-02T04:00:00Z", "last_seen": "2026-09-02T04:00:00Z",
				"confidence": 0.82, "sensitivity": "internal", "source_refs": []any{sourceEventID},
			},
		},
	}
	body := map[string]any{"source": "wazuh", "occurred_at": "2026-09-02T04:00:00Z", "payload": payload}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling fixture: %v", err)
	}
	return b
}

// TestIngestion_CriticalTargetEscalatesPriority proves adr/0019's
// escalation rule: a correlated case's severity is raised from medium to
// high only once an event names a registered privileged user/critical
// asset — the same 5-event correlation as
// TestIngestion_FiveAuthFailuresFromOneIPAutoRequestsBlock, but with a
// critical-target user, checked from the very first event (no threshold —
// escalation is about *who* was targeted, not *how many times*).
func TestIngestion_CriticalTargetEscalatesPriority(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	addBody, err := json.Marshal(map[string]any{"value": "admin", "reason": "domain admin account"})
	if err != nil {
		t.Fatalf("marshaling critical-target body: %v", err)
	}
	addReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/critical-targets", strings.NewReader(string(addBody)))
	addReq.Header.Set("Content-Type", "application/json")
	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatalf("POST /v1/critical-targets: %v", err)
	}
	addResp.Body.Close()
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 adding a critical target, got %d", addResp.StatusCode)
	}

	const ip = "203.0.113.53"
	status, accepted := postEvent(t, srv, "idem-key-escalate-0001", authFixtureWithUser(t, "escalate-test-0001", ip, "admin"))
	if status != http.StatusAccepted || accepted.Status != genapi.AcceptedEventStatusAccepted {
		t.Fatalf("event: status=%d body=%+v", status, accepted)
	}

	listResp, err := http.Get(srv.URL + "/v1/cases")
	if err != nil {
		t.Fatalf("listing cases: %v", err)
	}
	defer listResp.Body.Close()
	var page struct {
		Items []genapi.Case `json:"items"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&page)
	if len(page.Items) == 0 {
		t.Fatal("expected at least one case")
	}

	c := getCaseDetail(t, srv, page.Items[0].Id)
	if c.Severity != "high" {
		t.Errorf("severity = %q, want high (targets a registered critical target)", c.Severity)
	}
}

// TestIngestion_FiveAuthFailuresFromOneIPAutoRequestsBlock proves adr/0018's
// correlation + auto-trigger slice end to end: 5 authentication events
// sharing a source IP land on one case (not five), and the 5th one alone
// crosses the threshold and requests a block without any client ever
// calling the request_block operation by hand.
func TestIngestion_FiveAuthFailuresFromOneIPAutoRequestsBlock(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	const ip = "203.0.113.50"
	var caseID string
	for i := 1; i <= 5; i++ {
		sourceEventID := "autoblock-test-" + string(rune('0'+i))
		status, accepted := postEvent(t, srv, "idem-key-autoblock-000"+string(rune('0'+i)), authFixture(t, sourceEventID, ip))
		if status != http.StatusAccepted || accepted.Status != genapi.AcceptedEventStatusAccepted {
			t.Fatalf("event %d: status=%d body=%+v", i, status, accepted)
		}
		var thisCase string
		if err := pool.QueryRow(t.Context(),
			"select case_ref from alerts where id = $1", *accepted.AlertId,
		).Scan(&thisCase); err != nil {
			t.Fatalf("looking up case for event %d: %v", i, err)
		}
		if i == 1 {
			caseID = thisCase
		} else if thisCase != caseID {
			t.Fatalf("event %d landed on case %s, want all 5 correlated onto %s", i, thisCase, caseID)
		}
	}

	var repeatCount int
	if err := pool.QueryRow(t.Context(), "select repeat_count from cases where id = $1", caseID).Scan(&repeatCount); err != nil {
		t.Fatalf("reading repeat_count: %v", err)
	}
	if repeatCount != 5 {
		t.Errorf("repeat_count = %d, want 5", repeatCount)
	}

	detail := getCaseDetail(t, srv, caseID)
	if detail.PendingAction == nil {
		t.Fatal("expected the 5th correlated event to auto-request a block, but pending_action is nil")
	}
	if detail.PendingAction.Target.Value != ip {
		t.Errorf("auto-requested block target = %q, want %q", detail.PendingAction.Target.Value, ip)
	}
	findApprovalForAction(t, srv, detail.PendingAction.Id) // fails the test if none exists
}

// TestIngestion_CorrelatedEventsDoNotDuplicateObservables proves the fix for
// what TestIngestion_FiveAuthFailuresFromOneIPAutoRequestsBlock's manual
// verification surfaced: a case that correlates 5 events sharing one IP
// must show that IP once, not 5 times (which also meant web's Network
// block card rendered 5 identical "Request block" buttons).
func TestIngestion_CorrelatedEventsDoNotDuplicateObservables(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	const ip = "203.0.113.52"
	var caseID string
	for i := 1; i <= 3; i++ {
		sourceEventID := "dedup-test-" + string(rune('0'+i))
		_, accepted := postEvent(t, srv, "idem-key-dedup-000"+string(rune('0'+i)), authFixture(t, sourceEventID, ip))
		if i == 1 {
			if err := pool.QueryRow(t.Context(),
				"select case_ref from alerts where id = $1", *accepted.AlertId,
			).Scan(&caseID); err != nil {
				t.Fatalf("looking up case: %v", err)
			}
		}
	}

	detail := getCaseDetail(t, srv, caseID)
	if detail.Observables == nil || len(*detail.Observables) != 1 {
		t.Fatalf("expected exactly 1 observable after 3 correlated events sharing an IP, got %v", detail.Observables)
	}
	if (*detail.Observables)[0].Value != ip {
		t.Errorf("observable value = %q, want %q", (*detail.Observables)[0].Value, ip)
	}
}

// TestIngestion_AllowlistedSourceNeverBlocked proves the negative test
// docs/08-testing/lab-scenario-ssh-bruteforce.md names: "Source IP is on
// trusted allowlist: no block action is created" — even past the same
// 5-event threshold the previous test crosses.
func TestIngestion_AllowlistedSourceNeverBlocked(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	const ip = "203.0.113.51"
	addBody, err := json.Marshal(map[string]any{"value": ip, "reason": "test management host"})
	if err != nil {
		t.Fatalf("marshaling allowlist body: %v", err)
	}
	addReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/allowlist", strings.NewReader(string(addBody)))
	addReq.Header.Set("Content-Type", "application/json")
	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatalf("POST /v1/allowlist: %v", err)
	}
	addResp.Body.Close()
	if addResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 adding a trusted source, got %d", addResp.StatusCode)
	}

	var caseID string
	for i := 1; i <= 5; i++ {
		sourceEventID := "allowlist-test-" + string(rune('0'+i))
		status, accepted := postEvent(t, srv, "idem-key-allowlist-000"+string(rune('0'+i)), authFixture(t, sourceEventID, ip))
		if status != http.StatusAccepted || accepted.Status != genapi.AcceptedEventStatusAccepted {
			t.Fatalf("event %d: status=%d body=%+v", i, status, accepted)
		}
		if i == 1 {
			if err := pool.QueryRow(t.Context(),
				"select case_ref from alerts where id = $1", *accepted.AlertId,
			).Scan(&caseID); err != nil {
				t.Fatalf("looking up case: %v", err)
			}
		}
	}

	detail := getCaseDetail(t, srv, caseID)
	if detail.PendingAction != nil {
		t.Errorf("expected no auto-requested block for an allowlisted source, got action %s", detail.PendingAction.Id)
	}
}
