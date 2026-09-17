// Package ticketing implements the M2 mock ticketing connector
// (docs/05-integrations/ticketing.md): a ticket record scoped to this lab's
// own Postgres, never a real Jira/ServiceNow call. Summaries never carry
// raw payload or credentials — only case id, severity, owner and a redacted
// summary, per ticketing.md's explicit prohibition. See
// adr/0010-mock-enrichment-and-ticketing-connectors.md.
package ticketing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/oklog/ulid/v2"

	"soar-platform/services/api/connectors"
)

const provider = "mock-ticketing"

var manifest = connectors.Manifest{
	ID:                 "connector-mock-ticketing",
	Owner:              "security-operations",
	Provider:           provider,
	Intents:            []string{"create_ticket", "update_ticket", "health"},
	ErrorClasses:       []string{"validation", "not_found", "policy_denied"},
	DryRun:             true,
	Idempotent:         true,
	DataClassification: "internal",
	TargetAllowlist:    []string{"tickets"},
}

// Ticket is the projection callers (the cases handler, tests) work with.
type Ticket struct {
	ExternalRef   string
	Status        string
	ClosureReason *string
}

// Connector is the ticketing side of the connectors.Connector contract,
// scoped to the caller's transaction so opening a ticket commits atomically
// with the case-update it belongs to.
type Connector struct {
	tx pgx.Tx
}

func New(tx pgx.Tx) *Connector { return &Connector{tx: tx} }

func (c *Connector) Manifest() connectors.Manifest { return manifest }

func (c *Connector) HealthCheck(ctx context.Context) error {
	var n int
	if err := c.tx.QueryRow(ctx, `select count(*) from tickets`).Scan(&n); err != nil {
		return &connectors.ConnectorError{Class: connectors.ErrClassProviderError, Message: "tickets table unreachable"}
	}
	return nil
}

// Invoke implements the connectors.Connector contract for direct contract
// testing. Params must carry "tenant_id", "case_id" and "summary" (string);
// Target must be "tickets".
func (c *Connector) Invoke(ctx context.Context, inv connectors.Invocation) (connectors.Result, error) {
	if inv.Target != "" && inv.Target != "tickets" {
		return connectors.Result{}, &connectors.ConnectorError{
			Class: connectors.ErrClassPolicyDenied, Message: "target not in allowlist",
		}
	}
	tenantID, _ := inv.Params["tenant_id"].(string)
	caseID, _ := inv.Params["case_id"].(string)
	summary, _ := inv.Params["summary"].(string)
	if tenantID == "" || caseID == "" || strings.TrimSpace(summary) == "" {
		return connectors.Result{}, &connectors.ConnectorError{
			Class: connectors.ErrClassValidation, Message: "tenant_id, case_id and summary are required",
		}
	}

	_, created, err := c.open(ctx, tenantID, caseID, summary, nil)
	if err != nil {
		return connectors.Result{}, err
	}
	return connectors.Result{Effect: created}, nil
}

// Open creates a ticket for the case if one doesn't already exist, or
// returns the existing one unchanged — idempotent per (tenant_id, case_id),
// per ticketing.md's "ต้อง idempotent ด้วย case/event key". created reports
// whether this call actually opened a new ticket, so callers can avoid
// logging a duplicate "ticket opened" event on a repeat call.
func Open(ctx context.Context, tx pgx.Tx, tenantID, caseID, summary string, evidenceRefs []string) (t Ticket, created bool, err error) {
	return New(tx).open(ctx, tenantID, caseID, summary, evidenceRefs)
}

func (c *Connector) open(ctx context.Context, tenantID, caseID, summary string, evidenceRefs []string) (Ticket, bool, error) {
	existing, err := lookup(ctx, c.tx, tenantID, caseID)
	if err != nil {
		return Ticket{}, false, err
	}
	if existing != nil {
		return *existing, false, nil
	}
	// evidence_refs is not-null in the schema; a nil slice would bind as SQL
	// NULL rather than falling back to the column default (same class of bug
	// M1 hit with events/observable label arrays).
	if evidenceRefs == nil {
		evidenceRefs = []string{}
	}

	externalRef := "TCK-" + strings.ToUpper(ulid.Make().String()[:10])
	if _, err := c.tx.Exec(ctx, `
		insert into tickets (id, tenant_id, case_id, provider, external_ref, status, summary, evidence_refs)
		values ($1, $2, $3, $4, $5, 'open', $6, $7)`,
		"tkt-"+strings.ToLower(ulid.Make().String()), tenantID, caseID, provider, externalRef, summary, evidenceRefs,
	); err != nil {
		return Ticket{}, false, fmt.Errorf("creating ticket: %w", err)
	}
	return Ticket{ExternalRef: externalRef, Status: "open"}, true, nil
}

// Close closes the ticket for a case, if one exists, recording the same
// closure_reason the case itself was closed with. A case with no ticket is
// not an error — opening a ticket is optional.
func Close(ctx context.Context, tx pgx.Tx, tenantID, caseID, closureReason string) error {
	_, err := tx.Exec(ctx, `
		update tickets set status = 'closed', closed_at = now(), closure_reason = $3
		where tenant_id = $1 and case_id = $2 and status != 'closed'`,
		tenantID, caseID, closureReason,
	)
	if err != nil {
		return fmt.Errorf("closing ticket: %w", err)
	}
	return nil
}

// Get returns the ticket for a case, or nil if none was opened.
func Get(ctx context.Context, q querier, tenantID, caseID string) (*Ticket, error) {
	return lookup(ctx, q, tenantID, caseID)
}

// querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func lookup(ctx context.Context, q querier, tenantID, caseID string) (*Ticket, error) {
	var t Ticket
	err := q.QueryRow(ctx,
		`select external_ref, status, closure_reason from tickets where tenant_id = $1 and case_id = $2`,
		tenantID, caseID,
	).Scan(&t.ExternalRef, &t.Status, &t.ClosureReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up ticket: %w", err)
	}
	return &t, nil
}
