// Package evidence owns evidence metadata per schemas/evidence.schema.json and
// docs/09-governance/evidence-and-custody.md: SOAR keeps references, hashes
// and custody events, never original bytes (those live in the external DFIR
// tree). The one exception is the raw payload M1's ingestion already stores
// in Postgres for the event itself — RecordFromEvent points evidence at that
// existing hash, and Get re-verifies it on every read rather than trusting
// the stored integrity_status. See adr/0012-evidence-integrity-and-audit-export.md.
package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
)

var ErrNotFound = errors.New("evidence not found")

// ErrValidation is the sentinel every manual-registration field error wraps;
// the HTTP handler maps it to a 400 via errors.Is.
var ErrValidation = errors.New("invalid evidence manifest")

var validSensitivities = map[string]bool{"public": true, "internal": true, "confidential": true, "restricted": true}

// CustodyEvent mirrors schemas/evidence.schema.json's custody_events items.
type CustodyEvent struct {
	Event    string
	Actor    string
	At       time.Time
	FromHash *string
	ToHash   *string
}

// Evidence mirrors schemas/evidence.schema.json.
type Evidence struct {
	ID                string
	CaseID            string
	SourceEventID     *string
	ArtifactType      string
	URI               string
	SHA256            string
	Size              int
	Source            string
	AcquisitionMethod string
	Collector         string
	CollectedAt       time.Time
	CreatedAt         time.Time
	ReceivedAt        time.Time
	Timezone          string
	ToolName          string
	ToolVersion       string
	CommandRef        string
	Analyst           string
	Purpose           string
	Limitations       []string
	DerivedFrom       []string
	Sensitivity       string
	RetentionClass    string
	IntegrityStatus   string
	CustodyEvents     []CustodyEvent
}

// querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func newID() string { return "evd-" + strings.ToLower(ulid.Make().String()) }

func nowUTC() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// RecordFromEvent auto-captures the raw payload of a just-ingested event as
// evidence, in the same transaction as the ingest — the "no manual step
// needed for the common path" judgment already used for alert/case creation.
//
// The SHA-256 is computed from a fresh read of events.raw_payload (not the
// pre-insert bytes events.go marshaled), because Postgres jsonb does not
// preserve object key order or whitespace: re-reading the same logical
// payload later would otherwise never hash-match the value computed before
// insertion. Hashing what verify-on-read will actually re-read keeps the
// comparison in Get meaningful. See adr/0012-evidence-integrity-and-audit-export.md.
func RecordFromEvent(ctx context.Context, tx pgx.Tx, tenantID, caseID, eventID, source, uri string, observedAt time.Time) (Evidence, error) {
	var rawPayload []byte
	if err := tx.QueryRow(ctx, `select raw_payload from events where tenant_id = $1 and id = $2`, tenantID, eventID).Scan(&rawPayload); err != nil {
		return Evidence{}, fmt.Errorf("loading inserted event for evidence hashing: %w", err)
	}
	sum := sha256.Sum256(rawPayload)
	sha256Hex := hex.EncodeToString(sum[:])

	now := nowUTC()
	ev := Evidence{
		ID:                newID(),
		CaseID:            caseID,
		SourceEventID:     &eventID,
		ArtifactType:      "captured_event_payload",
		URI:               uri,
		SHA256:            sha256Hex,
		Size:              len(rawPayload),
		Source:            source,
		AcquisitionMethod: "authenticated_webhook_capture",
		Collector:         "soar-ingestion",
		CollectedAt:       observedAt,
		CreatedAt:         now,
		ReceivedAt:        now,
		Timezone:          "UTC",
		ToolName:          "soar-ingestion",
		ToolVersion:       "m1.0",
		CommandRef:        fmt.Sprintf("case://%s/commands/ingest-%s", caseID, eventID),
		Analyst:           "system",
		Purpose:           "Preserve the normalized event payload for triage and replay.",
		Limitations:       []string{},
		DerivedFrom:       []string{},
		Sensitivity:       "internal",
		RetentionClass:    "lab-short",
		IntegrityStatus:   "verified",
	}
	if err := insert(ctx, tx, tenantID, ev); err != nil {
		return Evidence{}, err
	}
	if err := appendCustody(ctx, tx, ev.ID, CustodyEvent{Event: "collected", Actor: ev.Collector, At: now, ToHash: &sha256Hex}); err != nil {
		return Evidence{}, err
	}
	return ev, nil
}

// ManualEvidence is the subset of Evidence an analyst supplies when
// registering an artifact whose original bytes live outside this system
// (POST /v1/evidence/manifests).
type ManualEvidence struct {
	CaseID            string
	ArtifactType      string
	URI               string
	SHA256            string
	Size              int
	Source            string
	AcquisitionMethod string
	Collector         string
	CollectedAt       time.Time
	ToolName          string
	ToolVersion       string
	CommandRef        string
	Analyst           string
	Purpose           string
	Limitations       []string
	DerivedFrom       []string
	Sensitivity       string
	RetentionClass    string
}

// Register validates and inserts a manually-supplied evidence manifest.
// integrity_status is always "pending": this system has no local copy of the
// original bytes to recompute a hash against, per evidence-and-custody.md's
// "original evidence lives in the DFIR system" rule.
func Register(ctx context.Context, tx pgx.Tx, tenantID string, m ManualEvidence) (Evidence, error) {
	if err := validateManual(m); err != nil {
		return Evidence{}, err
	}
	now := nowUTC()
	ev := Evidence{
		ID:                newID(),
		CaseID:            m.CaseID,
		ArtifactType:      m.ArtifactType,
		URI:               m.URI,
		SHA256:            strings.ToLower(m.SHA256),
		Size:              m.Size,
		Source:            m.Source,
		AcquisitionMethod: m.AcquisitionMethod,
		Collector:         m.Collector,
		CollectedAt:       m.CollectedAt,
		CreatedAt:         now,
		ReceivedAt:        now,
		Timezone:          "UTC",
		ToolName:          m.ToolName,
		ToolVersion:       m.ToolVersion,
		CommandRef:        m.CommandRef,
		Analyst:           m.Analyst,
		Purpose:           m.Purpose,
		Limitations:       nonNil(m.Limitations),
		DerivedFrom:       nonNil(m.DerivedFrom),
		Sensitivity:       m.Sensitivity,
		RetentionClass:    m.RetentionClass,
		IntegrityStatus:   "pending",
	}
	if err := insert(ctx, tx, tenantID, ev); err != nil {
		return Evidence{}, err
	}
	if err := appendCustody(ctx, tx, ev.ID, CustodyEvent{Event: "registered", Actor: m.Analyst, At: now}); err != nil {
		return Evidence{}, err
	}
	return ev, nil
}

func validateManual(m ManualEvidence) error {
	missing := func(cond bool, field string) error {
		if cond {
			return fmt.Errorf("%w: %s is required", ErrValidation, field)
		}
		return nil
	}
	for _, err := range []error{
		missing(strings.TrimSpace(m.CaseID) == "", "case_id"),
		missing(strings.TrimSpace(m.ArtifactType) == "", "artifact_type"),
		missing(strings.TrimSpace(m.URI) == "", "uri"),
		missing(len(m.SHA256) != 64, "sha256 (must be 64 hex characters)"),
		missing(m.Size < 0, "size"),
		missing(strings.TrimSpace(m.Source) == "", "source"),
		missing(strings.TrimSpace(m.AcquisitionMethod) == "", "acquisition_method"),
		missing(strings.TrimSpace(m.Collector) == "", "collector"),
		missing(m.CollectedAt.IsZero(), "collected_at"),
		missing(strings.TrimSpace(m.ToolName) == "", "tool_name"),
		missing(strings.TrimSpace(m.ToolVersion) == "", "tool_version"),
		missing(strings.TrimSpace(m.CommandRef) == "", "command_ref"),
		missing(strings.TrimSpace(m.Analyst) == "", "analyst"),
		missing(strings.TrimSpace(m.Purpose) == "", "purpose"),
		missing(!validSensitivities[m.Sensitivity], "sensitivity"),
		missing(strings.TrimSpace(m.RetentionClass) == "", "retention_class"),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func insert(ctx context.Context, q querier, tenantID string, ev Evidence) error {
	_, err := q.Exec(ctx, `
		insert into evidence (
			id, tenant_id, case_id, source_event_id, artifact_type, uri, sha256, size,
			source, acquisition_method, collector, collected_at, created_at, received_at,
			timezone, tool_name, tool_version, command_ref, analyst, purpose, limitations,
			derived_from, sensitivity, retention_class, integrity_status
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`,
		ev.ID, tenantID, ev.CaseID, ev.SourceEventID, ev.ArtifactType, ev.URI, ev.SHA256, ev.Size,
		ev.Source, ev.AcquisitionMethod, ev.Collector, ev.CollectedAt, ev.CreatedAt, ev.ReceivedAt,
		ev.Timezone, ev.ToolName, ev.ToolVersion, ev.CommandRef, ev.Analyst, ev.Purpose, ev.Limitations,
		ev.DerivedFrom, ev.Sensitivity, ev.RetentionClass, ev.IntegrityStatus,
	)
	if err != nil {
		return fmt.Errorf("inserting evidence: %w", err)
	}
	return nil
}

func appendCustody(ctx context.Context, q querier, evidenceID string, ce CustodyEvent) error {
	_, err := q.Exec(ctx, `
		insert into evidence_custody_events (evidence_id, event, actor, at, from_hash, to_hash)
		values ($1,$2,$3,$4,$5,$6)`,
		evidenceID, ce.Event, ce.Actor, ce.At, ce.FromHash, ce.ToHash,
	)
	if err != nil {
		return fmt.Errorf("inserting custody event: %w", err)
	}
	return nil
}

// Get is the M4 gate: when the evidence is linked to a live event
// (source_event_id set), it recomputes the SHA-256 from that event's stored
// raw_payload and compares it to the recorded hash — never trusting the
// stored integrity_status — and persists a mismatch on the spot, exactly
// like actions.expireIfStale's lazy re-check in M3.
func Get(ctx context.Context, tx pgx.Tx, tenantID, id string) (*Evidence, error) {
	ev, err := selectOne(ctx, tx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if ev.SourceEventID == nil {
		return ev, nil
	}

	var rawPayload []byte
	err = tx.QueryRow(ctx, `select raw_payload from events where tenant_id = $1 and id = $2`, tenantID, *ev.SourceEventID).Scan(&rawPayload)
	if err == pgx.ErrNoRows {
		return ev, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loading source event for verification: %w", err)
	}

	sum := sha256.Sum256(rawPayload)
	recomputed := hex.EncodeToString(sum[:])
	if recomputed == ev.SHA256 {
		if ev.IntegrityStatus != "verified" {
			if err := setIntegrityStatus(ctx, tx, id, "verified"); err != nil {
				return nil, err
			}
			ev.IntegrityStatus = "verified"
		}
		return ev, nil
	}

	if err := setIntegrityStatus(ctx, tx, id, "mismatch"); err != nil {
		return nil, err
	}
	from := ev.SHA256
	if err := appendCustody(ctx, tx, id, CustodyEvent{Event: "integrity_mismatch", Actor: "system", At: nowUTC(), FromHash: &from, ToHash: &recomputed}); err != nil {
		return nil, err
	}
	ev.IntegrityStatus = "mismatch"
	return ev, nil
}

func setIntegrityStatus(ctx context.Context, q querier, id, status string) error {
	_, err := q.Exec(ctx, `update evidence set integrity_status = $2 where id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("updating integrity_status: %w", err)
	}
	return nil
}

func selectOne(ctx context.Context, q querier, tenantID, id string) (*Evidence, error) {
	var ev Evidence
	err := q.QueryRow(ctx, `
		select id, case_id, source_event_id, artifact_type, uri, sha256, size, source,
		       acquisition_method, collector, collected_at, created_at, received_at, timezone,
		       tool_name, tool_version, command_ref, analyst, purpose, limitations, derived_from,
		       sensitivity, retention_class, integrity_status
		from evidence where tenant_id = $1 and id = $2`,
		tenantID, id,
	).Scan(
		&ev.ID, &ev.CaseID, &ev.SourceEventID, &ev.ArtifactType, &ev.URI, &ev.SHA256, &ev.Size, &ev.Source,
		&ev.AcquisitionMethod, &ev.Collector, &ev.CollectedAt, &ev.CreatedAt, &ev.ReceivedAt, &ev.Timezone,
		&ev.ToolName, &ev.ToolVersion, &ev.CommandRef, &ev.Analyst, &ev.Purpose, &ev.Limitations, &ev.DerivedFrom,
		&ev.Sensitivity, &ev.RetentionClass, &ev.IntegrityStatus,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("loading evidence: %w", err)
	}
	custody, err := loadCustody(ctx, q, ev.ID)
	if err != nil {
		return nil, err
	}
	ev.CustodyEvents = custody
	return &ev, nil
}

// ListForCase does not re-verify each row's hash — that would make a case's
// evidence list pay one extra query per row. Only single-item Get re-verifies;
// see adr/0012-evidence-integrity-and-audit-export.md.
func ListForCase(ctx context.Context, q querier, tenantID, caseID string) ([]Evidence, error) {
	rows, err := q.Query(ctx, `
		select id, case_id, source_event_id, artifact_type, uri, sha256, size, source,
		       acquisition_method, collector, collected_at, created_at, received_at, timezone,
		       tool_name, tool_version, command_ref, analyst, purpose, limitations, derived_from,
		       sensitivity, retention_class, integrity_status
		from evidence where tenant_id = $1 and case_id = $2 order by created_at asc`,
		tenantID, caseID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing evidence: %w", err)
	}
	defer rows.Close()

	var out []Evidence
	for rows.Next() {
		var ev Evidence
		if err := rows.Scan(
			&ev.ID, &ev.CaseID, &ev.SourceEventID, &ev.ArtifactType, &ev.URI, &ev.SHA256, &ev.Size, &ev.Source,
			&ev.AcquisitionMethod, &ev.Collector, &ev.CollectedAt, &ev.CreatedAt, &ev.ReceivedAt, &ev.Timezone,
			&ev.ToolName, &ev.ToolVersion, &ev.CommandRef, &ev.Analyst, &ev.Purpose, &ev.Limitations, &ev.DerivedFrom,
			&ev.Sensitivity, &ev.RetentionClass, &ev.IntegrityStatus,
		); err != nil {
			return nil, fmt.Errorf("scanning evidence row: %w", err)
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		custody, err := loadCustody(ctx, q, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].CustodyEvents = custody
	}
	return out, nil
}

func loadCustody(ctx context.Context, q querier, evidenceID string) ([]CustodyEvent, error) {
	rows, err := q.Query(ctx, `select event, actor, at, from_hash, to_hash from evidence_custody_events where evidence_id = $1 order by at asc, id asc`, evidenceID)
	if err != nil {
		return nil, fmt.Errorf("loading custody events: %w", err)
	}
	defer rows.Close()

	var out []CustodyEvent
	for rows.Next() {
		var ce CustodyEvent
		if err := rows.Scan(&ce.Event, &ce.Actor, &ce.At, &ce.FromHash, &ce.ToHash); err != nil {
			return nil, fmt.Errorf("scanning custody event: %w", err)
		}
		out = append(out, ce)
	}
	return out, rows.Err()
}
