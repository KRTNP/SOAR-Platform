package cases

import (
	"testing"
	"time"
)

func TestValidStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"new", true},
		{"triage", true},
		{"investigating", true},
		{"contained", true},
		{"eradication", true},
		{"recovery", true},
		{"escalated", true},
		{"dismissed", true},
		{"closed", true},
		{"", false},
		{"NEW", false},
		{"resolved", false}, // that's an Alert status, not a Case status
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := ValidStatus(tt.status); got != tt.want {
				t.Errorf("ValidStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestNewFromAlert(t *testing.T) {
	opened := time.Date(2026, 8, 30, 4, 0, 3, 0, time.UTC)
	c := NewFromAlert("case-test-0001", "lab", "account_access", "medium", 0.82, "system", opened)

	if c.Status != "new" {
		t.Errorf("expected initial status 'new', got %q", c.Status)
	}
	if c.OpenedAt != opened || c.UpdatedAt != opened {
		t.Errorf("expected opened_at/updated_at to both be %v, got opened=%v updated=%v", opened, c.OpenedAt, c.UpdatedAt)
	}
	if c.Severity != "medium" || c.Confidence != 0.82 || c.Category != "account_access" {
		t.Errorf("severity/confidence/category not carried through: %+v", c)
	}
	if c.Title == "" {
		t.Error("expected a non-empty generated title")
	}
	// underscores in category should read as spaces in the generated title
	if got, want := c.Title, "account access alert (medium severity)"; got != want {
		t.Errorf("Title = %q, want %q", got, want)
	}
}
