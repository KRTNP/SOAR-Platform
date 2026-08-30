// Package enrichment implements the M2 mock/local IOC provider
// (docs/05-integrations/threat-intelligence.md): a lookup against a static
// table of lab-only indicators (RFC 5737/6761 test values), never a real
// TI feed. See adr/0010-mock-enrichment-and-ticketing-connectors.md.
package enrichment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"soar-platform/services/api/connectors"
)

const provider = "mock-local-ti"

var manifest = connectors.Manifest{
	ID:                 "connector-mock-local-ti",
	Owner:              "security-operations",
	Provider:           provider,
	Intents:            []string{"lookup_indicator", "health"},
	ErrorClasses:       []string{"validation", "not_found", "policy_denied"},
	DryRun:             true,
	Idempotent:         true,
	DataClassification: "internal",
	TargetAllowlist:    []string{"ioc_indicators"},
}

// Connector is the enrichment side of the connectors.Connector contract. It
// operates within the caller's transaction so an ingest's enrichment lookups
// commit or roll back atomically with the rest of the pipeline, matching
// M1's single-transaction ingest.
type Connector struct {
	tx pgx.Tx
}

func New(tx pgx.Tx) *Connector { return &Connector{tx: tx} }

func (c *Connector) Manifest() connectors.Manifest { return manifest }

// HealthCheck proves the local indicator table is reachable without writing
// anything (schemas/connector.schema.json's health_check.read_only).
func (c *Connector) HealthCheck(ctx context.Context) error {
	var n int
	if err := c.tx.QueryRow(ctx, `select count(*) from ioc_indicators`).Scan(&n); err != nil {
		return &connectors.ConnectorError{Class: connectors.ErrClassProviderError, Message: "ioc_indicators unreachable"}
	}
	return nil
}

// Invoke implements the connectors.Connector contract for direct contract
// testing. Params must carry "observable_id" (int64), "type" and "value"
// (string); Target must be "ioc_indicators" (the only allowlisted target —
// this connector never calls anything else).
func (c *Connector) Invoke(ctx context.Context, inv connectors.Invocation) (connectors.Result, error) {
	if inv.Target != "" && inv.Target != "ioc_indicators" {
		return connectors.Result{}, &connectors.ConnectorError{
			Class: connectors.ErrClassPolicyDenied, Message: "target not in allowlist",
		}
	}
	observableID, _ := inv.Params["observable_id"].(int64)
	obsType, _ := inv.Params["type"].(string)
	value, _ := inv.Params["value"].(string)
	if observableID == 0 || obsType == "" || value == "" {
		return connectors.Result{}, &connectors.ConnectorError{
			Class: connectors.ErrClassValidation, Message: "observable_id, type and value are required",
		}
	}

	inserted, err := c.lookupAndRecord(ctx, observableID, obsType, value)
	if err != nil {
		return connectors.Result{}, err
	}
	return connectors.Result{Effect: inserted}, nil
}

// enrichableTypes mirrors this connector's intent scope — the same
// indicator types threat-intelligence.md and ioc_indicators cover. Other
// observable types (user, host, process, file, vulnerability) are simply
// not this provider's concern and get no enrichment_results row at all,
// rather than a noisy "unknown" verdict on everything.
var enrichableTypes = map[string]bool{
	"ipv4": true, "ipv6": true, "domain": true, "url": true, "hash": true, "email": true,
}

// LookupAll runs a lookup for every given observable whose type this
// provider covers, tolerant of "no match" (records verdict: unknown, never
// an error) so ingestion never fails because a value isn't in the mock
// store.
func LookupAll(ctx context.Context, tx pgx.Tx, observableIDs []int64) error {
	c := New(tx)
	for _, id := range observableIDs {
		var obsType, value string
		if err := tx.QueryRow(ctx, `select type, value from observables where id = $1`, id).Scan(&obsType, &value); err != nil {
			return fmt.Errorf("loading observable %d for enrichment: %w", id, err)
		}
		if !enrichableTypes[obsType] {
			continue
		}
		if _, err := c.lookupAndRecord(ctx, id, obsType, value); err != nil {
			return err
		}
	}
	return nil
}

// lookupAndRecord is idempotent per (observable_id, provider): a repeat call
// leaves the existing row untouched and reports no new effect.
func (c *Connector) lookupAndRecord(ctx context.Context, observableID int64, obsType, value string) (bool, error) {
	var exists bool
	if err := c.tx.QueryRow(ctx,
		`select exists(select 1 from enrichment_results where observable_id = $1 and provider = $2)`,
		observableID, provider,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("checking existing enrichment result: %w", err)
	}
	if exists {
		return false, nil
	}

	var (
		indicatorID int64
		verdict     string
		reliability *string
		confidence  float64
		rawRef      string
	)
	err := c.tx.QueryRow(ctx,
		`select id, verdict, reliability from ioc_indicators where type = $1 and value = $2`,
		obsType, value,
	).Scan(&indicatorID, &verdict, &reliability)
	switch {
	case err == nil:
		confidence = verdictConfidence(verdict)
		rawRef = fmt.Sprintf("db://ioc_indicators/%d", indicatorID)
	case err == pgx.ErrNoRows:
		verdict = "unknown"
		rawRef = "mock://no-match"
	default:
		return false, fmt.Errorf("querying ioc_indicators: %w", err)
	}

	if _, err := c.tx.Exec(ctx, `
		insert into enrichment_results
			(tenant_id, observable_id, provider, verdict, confidence, source_reliability, sighting_time, raw_reference)
		select tenant_id, $1, $2, $3, $4, $5, now(), $6
		from observables where id = $1`,
		observableID, provider, verdict, confidence, reliability, rawRef,
	); err != nil {
		return false, fmt.Errorf("recording enrichment result: %w", err)
	}
	return true, nil
}

func verdictConfidence(verdict string) float64 {
	switch verdict {
	case "malicious":
		return 0.9
	case "suspicious":
		return 0.6
	case "benign":
		return 0.8
	default:
		return 0.0
	}
}
