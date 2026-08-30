package cases

import (
	"testing"
	"time"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/genapi"
	"soar-platform/services/api/ticketing"
)

func TestToAPICaseDetail_OmitsOptionalFieldsWhenAbsent(t *testing.T) {
	c := &Case{
		ID: "case-1", TenantID: "lab", Title: "t", Category: "c", Severity: "medium",
		Status: "new", OpenedAt: time.Now(), Owner: "system",
	}
	out := toAPICaseDetail(c, nil, nil, nil, nil)
	if out.Observables != nil {
		t.Error("expected no observables field when none were passed")
	}
	if out.TicketRef != nil || out.TicketStatus != nil {
		t.Error("expected no ticket fields when no ticket exists")
	}
	if out.PendingAction != nil {
		t.Error("expected no pending_action field when no action exists")
	}
}

func TestToAPICaseDetail_PopulatesVerdictAndTicket(t *testing.T) {
	c := &Case{
		ID: "case-1", TenantID: "lab", Title: "t", Category: "c", Severity: "medium",
		Status: "new", OpenedAt: time.Now(), Owner: "system",
	}
	verdict := "malicious"
	confidence := 0.9
	reliability := "b"
	observables := []ObservableRef{
		{Type: "ipv4", Value: "198.51.100.10", Verdict: &verdict, TiConfidence: &confidence, SourceReliability: &reliability},
		{Type: "user", Value: "alice"}, // never enriched: no verdict
	}
	ticket := &ticketing.Ticket{ExternalRef: "TCK-ABC123", Status: "open"}
	action := &actions.Action{
		ID: "act-1", CaseID: "case-1", Intent: "block_network_indicator",
		TargetType: "ipv4", TargetValue: "198.51.100.10", Risk: "high", Mode: "execute",
		RequestedBy: "analyst", Status: "pending_approval", IdempotencyKey: "case-1:block:ipv4:198.51.100.10",
	}
	approval := &actions.Approval{ID: "apr-1", ActionID: "act-1", Status: "pending", ExpiresAt: time.Now().Add(15 * time.Minute)}

	out := toAPICaseDetail(c, observables, ticket, action, approval)

	if out.Observables == nil || len(*out.Observables) != 2 {
		t.Fatalf("expected 2 observables, got %v", out.Observables)
	}
	enriched := (*out.Observables)[0]
	if enriched.Verdict == nil || *enriched.Verdict != genapi.Malicious {
		t.Errorf("expected verdict malicious, got %v", enriched.Verdict)
	}
	if enriched.TiConfidence == nil || *enriched.TiConfidence != float32(0.9) {
		t.Errorf("expected ti_confidence 0.9, got %v", enriched.TiConfidence)
	}
	unenriched := (*out.Observables)[1]
	if unenriched.Verdict != nil {
		t.Errorf("expected no verdict for a never-enriched observable, got %v", unenriched.Verdict)
	}

	if out.TicketRef == nil || *out.TicketRef != "TCK-ABC123" {
		t.Errorf("expected ticket_ref TCK-ABC123, got %v", out.TicketRef)
	}
	if out.TicketStatus == nil || *out.TicketStatus != genapi.CaseTicketStatusOpen {
		t.Errorf("expected ticket_status open, got %v", out.TicketStatus)
	}

	if out.PendingAction == nil || out.PendingAction.Id != "act-1" {
		t.Fatalf("expected pending_action act-1, got %v", out.PendingAction)
	}
	if out.PendingAction.Status != genapi.ActionStatusPendingApproval {
		t.Errorf("expected pending_action status pending_approval, got %v", out.PendingAction.Status)
	}
}
