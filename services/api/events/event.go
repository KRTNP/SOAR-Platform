// Package events owns the canonical Event aggregate: normalizing an
// EventIngestRequest into schemas/event.schema.json shape, deterministic
// dedup, persistence, and the HTTP handlers for /v1/events. Alert and case
// creation from a normalized event live in the alerts/cases packages —
// events only knows how to become an Event.
package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"soar-platform/services/api/genapi"
	"soar-platform/services/api/httpx"
)

var validEventClasses = []string{
	"authentication", "network", "process", "file", "email",
	"vulnerability", "cloud", "malware", "other",
}

var validSeverities = []string{"informational", "low", "medium", "high", "critical"}

var validObservableTypes = []string{
	"ipv4", "ipv6", "domain", "url", "hash", "email", "user", "host", "process", "file", "vulnerability",
}

var validSensitivities = []string{"public", "internal", "confidential", "restricted"}

// RawRef mirrors event.schema.json's raw_ref object.
type RawRef struct {
	URI    string
	SHA256 string
}

// Observable mirrors schemas/observable.schema.json for the subset of fields
// M1 persists (no algorithm-specific hash normalization yet).
type Observable struct {
	Type          string
	Value         string
	Algorithm     *string
	Normalization string
	FirstSeen     time.Time
	LastSeen      time.Time
	Confidence    float64
	Sensitivity   string
	SourceRefs    []string
	Labels        []string
}

// Event mirrors schemas/event.schema.json.
type Event struct {
	ID            string
	TenantID      string
	Source        string
	SourceEventID *string
	ObservedAt    time.Time
	ReceivedAt    time.Time
	EventClass    string
	Severity      string
	Confidence    float64
	RawPayload    json.RawMessage
	RawRef        RawRef
	Labels        []string
	Extensions    map[string]any
	SchemaVersion string
	Observables   []Observable
}

// knownPayloadKeys are the canonical fields Normalize extracts explicitly.
// Anything else in the payload is preserved in Extensions rather than
// dropped, per docs/03-domain-model/normalization-strategy.md.
var knownPayloadKeys = map[string]bool{
	"event_class": true, "severity": true, "confidence": true,
	"source_event_id": true, "raw_ref": true, "observables": true,
	"labels": true, "schema_version": true, "observed_at": true,
}

// Normalize maps an EventIngestRequest into the canonical Event shape. It
// requires the payload to already carry event_class/severity/confidence —
// today's only accepted sources (the lab's own fixtures and a synthetic
// replay client) produce payloads in that shape. A vendor-specific field
// adapter (e.g. mapping Wazuh's rule.level to severity) is out of scope for
// M1 and belongs in a later milestone's source-adapter layer.
func Normalize(req genapi.EventIngestRequest, tenantID, id string, receivedAt time.Time) (*Event, error) {
	payload := req.Payload
	if payload == nil {
		payload = map[string]any{}
	}

	eventClass, err := stringEnum(payload, "event_class", validEventClasses)
	if err != nil {
		return nil, err
	}
	severity, err := stringEnum(payload, "severity", validSeverities)
	if err != nil {
		return nil, err
	}
	confidence, err := numberInRange(payload, "confidence", 0, 1)
	if err != nil {
		return nil, err
	}
	sourceEventID := stringPtr(payload, "source_event_id")

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, httpx.ErrValidation("invalid_payload", "payload could not be re-encoded").WithCause(err)
	}

	rawRef, err := extractRawRef(payload, id, rawPayload)
	if err != nil {
		return nil, err
	}

	observables, err := extractObservables(payload)
	if err != nil {
		return nil, err
	}

	labels := stringSlice(payload, "labels")

	schemaVersion, _ := payload["schema_version"].(string)
	if schemaVersion == "" {
		schemaVersion = "event-1.0"
	}

	extensions := map[string]any{}
	for k, v := range payload {
		if !knownPayloadKeys[k] {
			extensions[k] = v
		}
	}

	return &Event{
		ID:            id,
		TenantID:      tenantID,
		Source:        req.Source,
		SourceEventID: sourceEventID,
		ObservedAt:    req.OccurredAt,
		ReceivedAt:    receivedAt,
		EventClass:    eventClass,
		Severity:      severity,
		Confidence:    confidence,
		RawPayload:    rawPayload,
		RawRef:        rawRef,
		Labels:        labels,
		Extensions:    extensions,
		SchemaVersion: schemaVersion,
		Observables:   observables,
	}, nil
}

func extractRawRef(payload map[string]any, id string, rawPayload []byte) (RawRef, error) {
	if raw, ok := payload["raw_ref"].(map[string]any); ok {
		uri, _ := raw["uri"].(string)
		sha, _ := raw["sha256"].(string)
		if uri != "" && sha != "" {
			return RawRef{URI: uri, SHA256: sha}, nil
		}
	}
	sum := sha256.Sum256(rawPayload)
	return RawRef{URI: fmt.Sprintf("db://events/%s", id), SHA256: hex.EncodeToString(sum[:])}, nil
}

func extractObservables(payload map[string]any) ([]Observable, error) {
	raw, ok := payload["observables"].([]any)
	if !ok {
		return nil, nil
	}
	out := make([]Observable, 0, len(raw))
	for i, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, httpx.ErrValidation("invalid_observable", fmt.Sprintf("observables[%d] must be an object", i))
		}
		obsType, err := stringEnum(m, "type", validObservableTypes)
		if err != nil {
			return nil, err
		}
		value, _ := m["value"].(string)
		if value == "" {
			return nil, httpx.ErrValidation("invalid_observable", fmt.Sprintf("observables[%d].value is required", i))
		}
		normalization, _ := m["normalization"].(string)
		if normalization == "" {
			return nil, httpx.ErrValidation("invalid_observable", fmt.Sprintf("observables[%d].normalization is required", i))
		}
		firstSeen, err := timeField(m, "first_seen")
		if err != nil {
			return nil, err
		}
		lastSeen, err := timeField(m, "last_seen")
		if err != nil {
			return nil, err
		}
		confidence, err := numberInRange(m, "confidence", 0, 1)
		if err != nil {
			return nil, err
		}
		sensitivity, err := stringEnum(m, "sensitivity", validSensitivities)
		if err != nil {
			return nil, err
		}
		var algorithm *string
		if a, ok := m["algorithm"].(string); ok && a != "" {
			algorithm = &a
		}
		out = append(out, Observable{
			Type:          obsType,
			Value:         value,
			Algorithm:     algorithm,
			Normalization: normalization,
			FirstSeen:     firstSeen,
			LastSeen:      lastSeen,
			Confidence:    confidence,
			Sensitivity:   sensitivity,
			SourceRefs:    stringSlice(m, "source_refs"),
			Labels:        stringSlice(m, "labels"),
		})
	}
	return out, nil
}

func stringEnum(m map[string]any, key string, allowed []string) (string, error) {
	v, ok := m[key].(string)
	if !ok || v == "" {
		return "", httpx.ErrValidation("missing_field", fmt.Sprintf("payload.%s is required", key))
	}
	for _, a := range allowed {
		if v == a {
			return v, nil
		}
	}
	return "", httpx.ErrValidation("invalid_enum", fmt.Sprintf("payload.%s %q is not one of %v", key, v, allowed))
}

func numberInRange(m map[string]any, key string, min, max float64) (float64, error) {
	v, ok := m[key].(float64)
	if !ok {
		return 0, httpx.ErrValidation("missing_field", fmt.Sprintf("payload.%s is required and must be a number", key))
	}
	if v < min || v > max {
		return 0, httpx.ErrValidation("invalid_range", fmt.Sprintf("payload.%s must be between %v and %v", key, min, max))
	}
	return v, nil
}

func stringPtr(m map[string]any, key string) *string {
	if v, ok := m[key].(string); ok && v != "" {
		return &v
	}
	return nil
}

func stringSlice(m map[string]any, key string) []string {
	raw, ok := m[key].([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func timeField(m map[string]any, key string) (time.Time, error) {
	s, ok := m[key].(string)
	if !ok || s == "" {
		return time.Time{}, httpx.ErrValidation("missing_field", fmt.Sprintf("%s is required", key))
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, httpx.ErrValidation("invalid_timestamp", fmt.Sprintf("%s must be RFC3339 with a timezone", key)).WithCause(err)
	}
	return t, nil
}
