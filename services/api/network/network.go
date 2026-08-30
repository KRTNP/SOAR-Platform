// Package network implements the M3 mock network-block connector
// (docs/05-integrations/firewall.md): a row in this lab's own Postgres
// standing in for OPNsense's SOAR_TEMP_BLOCK alias, never a real firewall
// call. The real OPNsense connector is M5 ("lab connector"). See
// adr/0011-approval-gated-mock-network-block.md.
package network

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"soar-platform/services/api/connectors"
)

const provider = "mock-network-block"

var manifest = connectors.Manifest{
	ID:                 "connector-mock-network-block",
	Owner:              "security-operations",
	Provider:           provider,
	Intents:            []string{"block_network_indicator", "health"},
	ErrorClasses:       []string{"validation", "not_found", "policy_denied"},
	DryRun:             true,
	Idempotent:         true,
	DataClassification: "internal",
	TargetAllowlist:    []string{"network_blocks"},
}

// Connector is the network side of the connectors.Connector contract. It
// operates within the caller's transaction so a block commits atomically
// with the action state transition that authorized it.
type Connector struct {
	tx pgx.Tx
}

func New(tx pgx.Tx) *Connector { return &Connector{tx: tx} }

func (c *Connector) Manifest() connectors.Manifest { return manifest }

func (c *Connector) HealthCheck(ctx context.Context) error {
	var n int
	if err := c.tx.QueryRow(ctx, `select count(*) from network_blocks`).Scan(&n); err != nil {
		return &connectors.ConnectorError{Class: connectors.ErrClassProviderError, Message: "network_blocks unreachable"}
	}
	return nil
}

// Invoke implements the connectors.Connector contract for direct contract
// testing. Params must carry "tenant_id", "action_id" and "target_value"
// (string); Target must be "network_blocks".
func (c *Connector) Invoke(ctx context.Context, inv connectors.Invocation) (connectors.Result, error) {
	if inv.Target != "" && inv.Target != "network_blocks" {
		return connectors.Result{}, &connectors.ConnectorError{
			Class: connectors.ErrClassPolicyDenied, Message: "target not in allowlist",
		}
	}
	tenantID, _ := inv.Params["tenant_id"].(string)
	actionID, _ := inv.Params["action_id"].(string)
	targetValue, _ := inv.Params["target_value"].(string)
	if tenantID == "" || actionID == "" || targetValue == "" {
		return connectors.Result{}, &connectors.ConnectorError{
			Class: connectors.ErrClassValidation, Message: "tenant_id, action_id and target_value are required",
		}
	}

	created, err := c.block(ctx, tenantID, actionID, targetValue)
	if err != nil {
		return connectors.Result{}, err
	}
	return connectors.Result{Effect: created}, nil
}

// Block adds target_value to the mock block table for the given action, or
// leaves the existing row unchanged if one already exists — idempotent per
// action_id, so a repeated execute never double-blocks.
func Block(ctx context.Context, tx pgx.Tx, tenantID, actionID, targetValue string) (created bool, err error) {
	return New(tx).block(ctx, tenantID, actionID, targetValue)
}

func (c *Connector) block(ctx context.Context, tenantID, actionID, targetValue string) (bool, error) {
	var exists bool
	if err := c.tx.QueryRow(ctx,
		`select exists(select 1 from network_blocks where action_id = $1)`, actionID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("checking existing block: %w", err)
	}
	if exists {
		return false, nil
	}
	if _, err := c.tx.Exec(ctx, `
		insert into network_blocks (tenant_id, action_id, target_value)
		values ($1, $2, $3)`,
		tenantID, actionID, targetValue,
	); err != nil {
		return false, fmt.Errorf("creating network block: %w", err)
	}
	return true, nil
}

// Release marks the block for actionID as released (the rollback path). A
// missing or already-released block is not an error — release is idempotent.
func Release(ctx context.Context, tx pgx.Tx, actionID string) error {
	if _, err := tx.Exec(ctx, `
		update network_blocks set status = 'released', released_at = now()
		where action_id = $1 and status != 'released'`,
		actionID,
	); err != nil {
		return fmt.Errorf("releasing network block: %w", err)
	}
	return nil
}
