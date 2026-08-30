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
}

// NewFromAlert builds the M1 case for a newly created alert: one alert
// always opens exactly one case. A future correlation engine may instead
// attach a new alert to an existing open case — this constructor is only the
// "always open a new one" path.
func NewFromAlert(id, tenantID, category, severity string, confidence float64, owner string, openedAt time.Time) *Case {
	return &Case{
		ID:         id,
		TenantID:   tenantID,
		Title:      fmt.Sprintf("%s alert (%s severity)", strings.ReplaceAll(category, "_", " "), severity),
		Category:   category,
		Severity:   severity,
		Confidence: confidence,
		Status:     "new",
		OpenedAt:   openedAt,
		UpdatedAt:  openedAt,
		Owner:      owner,
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
