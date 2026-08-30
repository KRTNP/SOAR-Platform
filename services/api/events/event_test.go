package events

import (
	"errors"
	"testing"
	"time"

	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
)

func baseRequest(payload map[string]any) genapi.EventIngestRequest {
	return genapi.EventIngestRequest{
		Source:     "wazuh",
		OccurredAt: time.Date(2026, 8, 30, 4, 0, 0, 0, time.UTC),
		Payload:    payload,
	}
}

func TestNormalize_Valid(t *testing.T) {
	req := baseRequest(map[string]any{
		"event_class":     "authentication",
		"severity":        "medium",
		"confidence":      0.82,
		"source_event_id": "wazuh-alert-0001",
		"rule":            map[string]any{"id": "5710", "level": float64(10)},
		"observables": []any{
			map[string]any{
				"type": "ipv4", "value": "198.51.100.10", "normalization": "rfc5737-test-net",
				"first_seen": "2026-08-30T04:00:00Z", "last_seen": "2026-08-30T04:00:00Z",
				"confidence": 0.82, "sensitivity": "internal",
				"source_refs": []any{"wazuh-alert-0001"},
			},
		},
	})

	ev, err := Normalize(req, "lab", "evt-test-0001", time.Date(2026, 8, 30, 4, 0, 2, 0, time.UTC))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if ev.EventClass != "authentication" || ev.Severity != "medium" || ev.Confidence != 0.82 {
		t.Errorf("canonical fields not extracted: %+v", ev)
	}
	if ev.SourceEventID == nil || *ev.SourceEventID != "wazuh-alert-0001" {
		t.Errorf("source_event_id not extracted: %+v", ev.SourceEventID)
	}
	if len(ev.Observables) != 1 || ev.Observables[0].Type != "ipv4" {
		t.Errorf("observables not extracted: %+v", ev.Observables)
	}
	// "rule" is not a canonical field; it must be preserved, not dropped.
	if _, ok := ev.Extensions["rule"]; !ok {
		t.Errorf("unknown payload field %q was dropped instead of preserved in extensions", "rule")
	}
	// event_class/severity/etc. must NOT leak into extensions twice.
	if _, ok := ev.Extensions["event_class"]; ok {
		t.Errorf("canonical field %q leaked into extensions", "event_class")
	}
}

func TestNormalize_MissingRequiredField(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
	}{
		{"missing event_class", map[string]any{"severity": "medium", "confidence": 0.5}},
		{"missing severity", map[string]any{"event_class": "network", "confidence": 0.5}},
		{"missing confidence", map[string]any{"event_class": "network", "severity": "low"}},
		{"invalid event_class enum", map[string]any{"event_class": "not-a-class", "severity": "low", "confidence": 0.5}},
		{"invalid severity enum", map[string]any{"event_class": "network", "severity": "extreme", "confidence": 0.5}},
		{"confidence out of range", map[string]any{"event_class": "network", "severity": "low", "confidence": 1.5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Normalize(baseRequest(tt.payload), "lab", "evt-test", time.Now())
			if err == nil {
				t.Fatalf("Normalize() expected an error, got nil")
			}
			var appErr *httpx.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("Normalize() error is not an *httpx.AppError: %v", err)
			}
			if appErr.Status != 400 {
				t.Errorf("expected 400, got %d", appErr.Status)
			}
		})
	}
}

func TestNormalize_MalformedObservable(t *testing.T) {
	_, err := Normalize(baseRequest(map[string]any{
		"event_class": "network", "severity": "low", "confidence": 0.5,
		"observables": []any{"not-an-object"},
	}), "lab", "evt-test", time.Now())
	if err == nil {
		t.Fatal("expected an error for a malformed observable")
	}
}

func TestNormalize_UnknownFieldsGoToExtensions(t *testing.T) {
	ev, err := Normalize(baseRequest(map[string]any{
		"event_class": "network", "severity": "low", "confidence": 0.5,
		"agent":                map[string]any{"id": "001", "name": "lab-endpoint-01"},
		"vendor_specific_junk": 42.0,
	}), "lab", "evt-test", time.Now())
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if _, ok := ev.Extensions["agent"]; !ok {
		t.Error("expected 'agent' to be preserved in extensions")
	}
	if _, ok := ev.Extensions["vendor_specific_junk"]; !ok {
		t.Error("expected 'vendor_specific_junk' to be preserved in extensions")
	}
}

func TestNormalize_DefaultsRawRefWhenAbsent(t *testing.T) {
	ev, err := Normalize(baseRequest(map[string]any{
		"event_class": "network", "severity": "low", "confidence": 0.5,
	}), "lab", "evt-test-0002", time.Now())
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if ev.RawRef.URI != "db://events/evt-test-0002" {
		t.Errorf("expected a default db:// raw_ref.uri, got %q", ev.RawRef.URI)
	}
	if len(ev.RawRef.SHA256) != 64 {
		t.Errorf("expected a 64-char sha256 hex digest, got %q", ev.RawRef.SHA256)
	}
}
