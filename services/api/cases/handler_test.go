package cases

import (
	"testing"
	"time"

	"soar-platform/services/api/genapi"
	"soar-platform/services/api/ticketing"
)

func TestToAPICaseDetail_OmitsOptionalFieldsWhenAbsent(t *testing.T) {
	c := &Case{
		ID: "case-1", TenantID: "lab", Title: "t", Category: "c", Severity: "medium",
		Status: "new", OpenedAt: time.Now(), Owner: "system",
	}
	out := toAPICaseDetail(c, nil, nil)
	if out.Observables != nil {
		t.Error("expected no observables field when none were passed")
	}
	if out.TicketRef != nil || out.TicketStatus != nil {
		t.Error("expected no ticket fields when no ticket exists")
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

	out := toAPICaseDetail(c, observables, ticket)

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
}
