// Package cases owns the Case aggregate (schemas/case.schema.json): creation
// from an alert, listing, detail, timeline, and the limited PATCH operations
// M1 supports (update, transition). Merge/split are part of the schema but
// have no domain logic yet — UnsupportedOperation is returned honestly
// instead of silently accepting them.
package cases

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("case not found")

var ValidStatuses = []string{
	"new", "triage", "investigating", "contained", "eradication",
	"recovery", "escalated", "dismissed", "closed",
}

type Case struct {
	ID                string
	TenantID          string
	Title             string
	Category          string
	Severity          string
	Confidence        float64
	Status            string
	OpenedAt          time.Time
	UpdatedAt         time.Time
	ClosedAt          *time.Time
	Owner             string
	IncidentCommander *string
	ServiceOwner      *string
	Hypothesis        *string
	Facts             []string
	Limitations       []string
	ClosureReason     *string
	CorrelationKey    *string
	RootCause         *string
	LessonsLearned    *string
	// CorrelationWindowSeconds and RepeatCount back adr/0018's correlation
	// slice: an event carrying the same CorrelationKey as this case, within
	// this window of OpenedAt, attaches to this case instead of opening a
	// new one, incrementing RepeatCount. nil/1 (the default) for a case
	// opened outside that path — most event classes still open a fresh case
	// per event, unchanged from M1.
	CorrelationWindowSeconds *int
	RepeatCount              int
}

// NewFromAlert builds the case for a newly created alert. correlationKey/
// correlationWindowSeconds are nil for a category this session's
// correlation slice (adr/0018) doesn't cover, or for a manually-created case
// (services/api/cases/handler.go's direct POST /v1/cases path) — both keep
// today's "always open a new one" behavior unchanged. A non-nil
// correlationKey is what services/api/events/handler.go's Ingest checks via
// FindOpenByCorrelation before calling this at all, so by the time this
// constructor runs for one, no existing open case matched it yet.
func NewFromAlert(id, tenantID, category, severity string, confidence float64, owner string, openedAt time.Time, correlationKey *string, correlationWindowSeconds *int) *Case {
	return &Case{
		ID:                       id,
		TenantID:                 tenantID,
		Title:                    fmt.Sprintf("%s alert (%s severity)", strings.ReplaceAll(category, "_", " "), severity),
		Category:                 category,
		Severity:                 severity,
		Confidence:               confidence,
		Status:                   "new",
		OpenedAt:                 openedAt,
		UpdatedAt:                openedAt,
		Owner:                    owner,
		CorrelationKey:           correlationKey,
		CorrelationWindowSeconds: correlationWindowSeconds,
		RepeatCount:              1,
	}
}

func ValidStatus(s string) bool {
	for _, v := range ValidStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// TimelineEntry backs GET /v1/cases/{caseId}/timeline.
type TimelineEntry struct {
	ID         int64          `json:"id"`
	OccurredAt time.Time      `json:"occurred_at"`
	Kind       string         `json:"kind"`
	Actor      string         `json:"actor"`
	Detail     map[string]any `json:"detail,omitempty"`
}
