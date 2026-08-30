package evidence

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func validManual() ManualEvidence {
	return ManualEvidence{
		CaseID:            "case-001",
		ArtifactType:      "memory_image",
		URI:               "../dfir/evidence/case-001/mem.raw",
		SHA256:            strings.Repeat("a", 64),
		Size:              1024,
		Source:            "host-01",
		AcquisitionMethod: "manual_capture",
		Collector:         "analyst-tier2",
		CollectedAt:       time.Now(),
		ToolName:          "ftk-imager",
		ToolVersion:       "4.7",
		CommandRef:        "case://case-001/commands/capture-0001",
		Analyst:           "analyst-tier2",
		Purpose:           "Preserve host memory for later analysis.",
		Sensitivity:       "internal",
		RetentionClass:    "lab-short",
	}
}

func TestValidateManual(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ManualEvidence)
		wantErr bool
	}{
		{"valid", func(m *ManualEvidence) {}, false},
		{"missing case_id", func(m *ManualEvidence) { m.CaseID = "" }, true},
		{"missing artifact_type", func(m *ManualEvidence) { m.ArtifactType = "" }, true},
		{"short sha256", func(m *ManualEvidence) { m.SHA256 = "abc" }, true},
		{"negative size", func(m *ManualEvidence) { m.Size = -1 }, true},
		{"zero collected_at", func(m *ManualEvidence) { m.CollectedAt = time.Time{} }, true},
		{"invalid sensitivity", func(m *ManualEvidence) { m.Sensitivity = "top_secret" }, true},
		{"missing retention_class", func(m *ManualEvidence) { m.RetentionClass = "" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := validManual()
			tt.mutate(&m)
			err := validateManual(m)
			if tt.wantErr && err == nil {
				t.Fatalf("validateManual() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateManual() = %v, want nil", err)
			}
			if tt.wantErr && !errors.Is(err, ErrValidation) {
				t.Errorf("error %v does not wrap ErrValidation", err)
			}
		})
	}
}

func TestNewIDHasEvdPrefix(t *testing.T) {
	id := newID()
	if !strings.HasPrefix(id, "evd-") {
		t.Errorf("newID() = %q, want evd- prefix", id)
	}
}
