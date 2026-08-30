// Package alerts owns the Alert aggregate (schemas/alert.schema.json). It
// takes plain values rather than an *events.Event so it stays decoupled from
// the events package — the ingestion handler (in events) is the only caller
// that knows about both.
package alerts

import "time"

type Alert struct {
	ID                       string
	TenantID                 string
	RuleID                   string
	RuleVersion              string
	DetectionSource          string
	Status                   string
	Severity                 string
	Confidence               float64
	CorrelationKey           *string
	CorrelationWindowSeconds *int
	CaseRef                  *string
	CreatedAt                time.Time
	EventIDs                 []string
}

// NewFromEvent builds the M1 alert for a single event: one event always
// produces exactly one alert. Grouping multiple events into one alert (e.g.
// "5 failed logins in 1 minute") is a rule/policy-engine concern for a later
// milestone — the RuleID below marks that this alert has no real detection
// rule behind it yet.
func NewFromEvent(id, tenantID, eventID, source, severity string, confidence float64, createdAt time.Time) *Alert {
	return &Alert{
		ID:              id,
		TenantID:        tenantID,
		RuleID:          "synthetic-ingest",
		RuleVersion:     "1.0",
		DetectionSource: source,
		Status:          "new",
		Severity:        severity,
		Confidence:      confidence,
		CreatedAt:       createdAt,
		EventIDs:        []string{eventID},
	}
}
