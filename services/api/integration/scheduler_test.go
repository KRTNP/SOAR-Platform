//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5"

	"soar-platform/services/api/scheduler"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestScheduler_ReleasesPastTTL: an executed action whose block TTL has
// elapsed is released and audited with actor="scheduler" by one Sweep call.
func TestScheduler_ReleasesPastTTL(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "scheduler-past-ttl-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approval := findApprovalForAction(t, srv, action.Id)
	approveApproval(t, srv, approval.Id).Body.Close()
	executeAction(t, srv, action.Id).Body.Close()

	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`update actions set executed_at = now() - interval '20 minutes' where id = $1`, action.Id,
	); err != nil {
		t.Fatalf("forcing executed_at into the past: %v", err)
	}

	released, err := scheduler.Sweep(ctx, pool, mockNetworkConnector(), testLogger())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if released != 1 {
		t.Fatalf("expected 1 release, got %d", released)
	}

	var status string
	if err := pool.QueryRow(ctx, `select status from actions where id = $1`, action.Id).Scan(&status); err != nil {
		t.Fatalf("reading action status: %v", err)
	}
	if status != "released" {
		t.Errorf("expected action status released, got %q", status)
	}

	var blockStatus string
	if err := pool.QueryRow(ctx, `select status from network_blocks where action_id = $1`, action.Id).Scan(&blockStatus); err != nil {
		t.Fatalf("reading network_blocks status: %v", err)
	}
	if blockStatus != "released" {
		t.Errorf("expected network_blocks status released, got %q", blockStatus)
	}

	var actor string
	if err := pool.QueryRow(ctx,
		`select actor from audit_log where execution_id = $1 and action = 'action.release' order by "timestamp" desc limit 1`, action.Id,
	).Scan(&actor); err != nil {
		t.Fatalf("reading audit_log: %v", err)
	}
	if actor != "scheduler" {
		t.Errorf("expected audit actor=scheduler, got %q", actor)
	}
}

// TestScheduler_LeavesActionWithinTTLAlone: an executed action not yet past
// its TTL is left untouched.
func TestScheduler_LeavesActionWithinTTLAlone(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "scheduler-within-ttl-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approval := findApprovalForAction(t, srv, action.Id)
	approveApproval(t, srv, approval.Id).Body.Close()
	executeAction(t, srv, action.Id).Body.Close()

	released, err := scheduler.Sweep(context.Background(), pool, mockNetworkConnector(), testLogger())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if released != 0 {
		t.Fatalf("expected 0 releases for an action within its TTL, got %d", released)
	}

	var status string
	if err := pool.QueryRow(context.Background(), `select status from actions where id = $1`, action.Id).Scan(&status); err != nil {
		t.Fatalf("reading action status: %v", err)
	}
	if status != "executed" {
		t.Errorf("expected action status still executed, got %q", status)
	}
}

// TestScheduler_SweepWithNoCandidatesIsANoop: an empty table is a no-op, not
// an error.
func TestScheduler_SweepWithNoCandidatesIsANoop(t *testing.T) {
	pool := setupPool(t)
	released, err := scheduler.Sweep(context.Background(), pool, mockNetworkConnector(), testLogger())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if released != 0 {
		t.Fatalf("expected 0 releases against an empty actions table, got %d", released)
	}
}

// TestScheduler_OneFailingCandidateDoesNotBlockTheRest: a connector error
// releasing one candidate must not abort the sweep for the others — each
// candidate runs in its own transaction.
func TestScheduler_OneFailingCandidateDoesNotBlockTheRest(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	caseID1 := caseWithBlockTarget(t, srv, "scheduler-partial-fail-0001")
	action1 := requestBlock(t, srv, caseID1, "198.51.100.10")
	approveApproval(t, srv, findApprovalForAction(t, srv, action1.Id).Id).Body.Close()
	executeAction(t, srv, action1.Id).Body.Close()

	// A different event class than case 1's, so adr/0018's correlation
	// (same-IP "authentication" events) doesn't merge this onto caseID1 —
	// this test needs two independent cases, not one correlated case.
	caseID2 := caseWithBlockTargetClass(t, srv, "scheduler-partial-fail-0002", "network")
	action2 := requestBlock(t, srv, caseID2, "198.51.100.10")
	approveApproval(t, srv, findApprovalForAction(t, srv, action2.Id).Id).Body.Close()
	executeAction(t, srv, action2.Id).Body.Close()

	ctx := context.Background()
	for _, id := range []string{action1.Id, action2.Id} {
		if _, err := pool.Exec(ctx, `update actions set executed_at = now() - interval '20 minutes' where id = $1`, id); err != nil {
			t.Fatalf("forcing executed_at into the past: %v", err)
		}
	}

	// A NetworkConnector whose Release fails only for action1's id.
	failing := mockNetworkConnector()
	realRelease := failing.Release
	failing.Release = func(ctx context.Context, tx pgx.Tx, actionID, targetValue string) error {
		if actionID == action1.Id {
			return fmt.Errorf("simulated connector outage")
		}
		return realRelease(ctx, tx, actionID, targetValue)
	}

	released, err := scheduler.Sweep(ctx, pool, failing, testLogger())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if released != 1 {
		t.Fatalf("expected exactly 1 successful release despite one failure, got %d", released)
	}

	var status2 string
	if err := pool.QueryRow(ctx, `select status from actions where id = $1`, action2.Id).Scan(&status2); err != nil {
		t.Fatalf("reading action2 status: %v", err)
	}
	if status2 != "released" {
		t.Errorf("expected action2 released, got %q", status2)
	}
	var status1 string
	if err := pool.QueryRow(ctx, `select status from actions where id = $1`, action1.Id).Scan(&status1); err != nil {
		t.Fatalf("reading action1 status: %v", err)
	}
	if status1 != "executed" {
		t.Errorf("expected action1 still executed after its release failed, got %q", status1)
	}
}
