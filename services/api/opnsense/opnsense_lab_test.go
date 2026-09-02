//go:build lab

// This file proves opnsense.Connector against the real lab OPNsense VM, not
// a fake server — it is excluded from the default build/test (no `lab` tag)
// so CI and other developers' machines never need live lab infrastructure.
// Run explicitly with: go test -tags lab ./services/api/opnsense/...
// after exporting OPNSENSE_BASE_URL, OPNSENSE_API_KEY, OPNSENSE_API_SECRET
// (and OPNSENSE_INSECURE_SKIP_VERIFY=true for the lab VM's self-signed
// cert). Uses a TEST-NET-3 (RFC 5737) address, never a real host IP, so a
// stray leftover entry can never affect real traffic.
package opnsense_test

import (
	"testing"

	"soar-platform/services/api/connectors"
	"soar-platform/services/api/opnsense"
)

const labTestIP = "203.0.113.55"

func invocationFor(ip string) connectors.Invocation {
	return connectors.Invocation{
		Intent: "add_temp_block",
		Target: "SOAR_TEMP_BLOCK",
		Params: map[string]any{"tenant_id": "lab", "action_id": "lab-test", "target_value": ip},
	}
}

func TestLabConnectorEndToEnd(t *testing.T) {
	cfg, err := opnsense.LoadConfig()
	if err != nil {
		t.Skipf("OPNSENSE_* env vars not set, skipping live lab test: %v", err)
	}
	c := opnsense.NewWithConfig(cfg)
	ctx := t.Context()

	if err := c.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck against live lab VM: %v", err)
	}

	// Start from a known state in case a prior run failed mid-way.
	_ = c.ReleaseTemp(ctx, labTestIP)

	preview, err := c.Preview(ctx, labTestIP)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if preview.AlreadyBlocked {
		t.Fatalf("expected %s to start absent from the alias", labTestIP)
	}

	res, err := c.Invoke(ctx, invocationFor(labTestIP))
	if err != nil {
		t.Fatalf("Invoke (add): %v", err)
	}
	if !res.Effect {
		t.Fatal("expected the first add to report an effect")
	}

	preview, err = c.Preview(ctx, labTestIP)
	if err != nil {
		t.Fatalf("Preview after add: %v", err)
	}
	if !preview.AlreadyBlocked {
		t.Fatalf("expected %s to be present in the alias after add", labTestIP)
	}

	res, err = c.Invoke(ctx, invocationFor(labTestIP))
	if err != nil {
		t.Fatalf("Invoke (repeat add): %v", err)
	}
	if res.Effect {
		t.Error("expected the repeat add to report no new effect")
	}

	if err := c.ReleaseTemp(ctx, labTestIP); err != nil {
		t.Fatalf("ReleaseTemp: %v", err)
	}
	preview, err = c.Preview(ctx, labTestIP)
	if err != nil {
		t.Fatalf("Preview after release: %v", err)
	}
	if preview.AlreadyBlocked {
		t.Fatalf("expected %s to be absent from the alias after release", labTestIP)
	}
}
