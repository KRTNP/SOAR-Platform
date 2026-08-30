package connectors

import (
	"context"
	"errors"
	"testing"
)

// ContractTestCase supplies the connector-specific fixtures RunContractTests
// needs. Everything here is deliberately data, not behavior — the harness
// owns the assertions so every connector is held to the same bar.
type ContractTestCase struct {
	// Valid is a baseline invocation the connector accepts.
	Valid Invocation
	// Malformed is missing/invalid input expected to fail as ErrClassValidation.
	Malformed Invocation
	// OutsideAllowlistTarget is a target string absent from the manifest's
	// TargetAllowlist; Valid's Target with this substituted must be rejected.
	OutsideAllowlistTarget string
	// CountEffects returns a count of the connector's durable effects (e.g.
	// rows in the table it writes to). The harness diffs it around
	// HealthCheck and a rejected Invoke to prove neither wrote anything.
	CountEffects func(t *testing.T) int
}

// RunContractTests exercises the connector-catalog.md/connector-testing.md
// checklist subset that is meaningful for a local/mock connector (no live
// network target exists at M2, so auth failure, timeout/retry exhaustion,
// and provider rate-limit are out of scope here — see
// adr/0010-mock-enrichment-and-ticketing-connectors.md). newConnector is
// called once per sub-test, receiving that sub-test's own *testing.T so a
// connector backed by a real transaction can register its own cleanup
// (e.g. rolling back), keeping every sub-test's state isolated.
func RunContractTests(t *testing.T, newConnector func(t *testing.T) Connector, tc ContractTestCase) {
	t.Helper()

	t.Run("manifest declares dry_run for a lab-only connector", func(t *testing.T) {
		m := newConnector(t).Manifest()
		if !m.DryRun {
			t.Fatal("a connector with no live external target must declare dry_run: true")
		}
		if len(m.TargetAllowlist) == 0 {
			t.Fatal("manifest must declare a non-empty target_allowlist")
		}
	})

	t.Run("health check is read-only", func(t *testing.T) {
		c := newConnector(t)
		before := tc.CountEffects(t)
		if err := c.HealthCheck(context.Background()); err != nil {
			t.Fatalf("HealthCheck: %v", err)
		}
		if after := tc.CountEffects(t); after != before {
			t.Errorf("HealthCheck changed the effect count from %d to %d", before, after)
		}
	})

	t.Run("malformed input is classified as validation", func(t *testing.T) {
		c := newConnector(t)
		_, err := c.Invoke(context.Background(), tc.Malformed)
		var cerr *ConnectorError
		if !errors.As(err, &cerr) {
			t.Fatalf("expected a *ConnectorError, got %T (%v)", err, err)
		}
		if cerr.Class != ErrClassValidation {
			t.Errorf("class = %q, want %q", cerr.Class, ErrClassValidation)
		}
	})

	t.Run("target outside allowlist is rejected before any effect", func(t *testing.T) {
		c := newConnector(t)
		before := tc.CountEffects(t)
		inv := tc.Valid
		inv.Target = tc.OutsideAllowlistTarget
		_, err := c.Invoke(context.Background(), inv)
		var cerr *ConnectorError
		if !errors.As(err, &cerr) {
			t.Fatalf("expected a *ConnectorError, got %T (%v)", err, err)
		}
		if cerr.Class != ErrClassPolicyDenied {
			t.Errorf("class = %q, want %q", cerr.Class, ErrClassPolicyDenied)
		}
		if after := tc.CountEffects(t); after != before {
			t.Errorf("a rejected out-of-allowlist call changed the effect count from %d to %d", before, after)
		}
	})

	t.Run("repeating an idempotency key does not duplicate the effect", func(t *testing.T) {
		c := newConnector(t)
		first, err := c.Invoke(context.Background(), tc.Valid)
		if err != nil {
			t.Fatalf("first Invoke: %v", err)
		}
		if !first.Effect {
			t.Fatal("expected the first call with a fresh idempotency key to have an effect")
		}
		second, err := c.Invoke(context.Background(), tc.Valid)
		if err != nil {
			t.Fatalf("second Invoke (same idempotency key): %v", err)
		}
		if second.Effect {
			t.Error("expected the repeated call to report no new effect")
		}
	})
}
