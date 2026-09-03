//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"soar-platform/services/api/actions"
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

	released, err := scheduler.Sweep(ctx, pool, mockNetworkConnector(), testLogger(), nil)
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

	released, err := scheduler.Sweep(context.Background(), pool, mockNetworkConnector(), testLogger(), nil)
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
	released, err := scheduler.Sweep(context.Background(), pool, mockNetworkConnector(), testLogger(), nil)
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

	released, err := scheduler.Sweep(ctx, pool, failing, testLogger(), nil)
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

// alwaysFailingConnector is mockNetworkConnector with Release replaced to
// always error, for the retry/escalation tests below.
func alwaysFailingConnector() actions.NetworkConnector {
	failing := mockNetworkConnector()
	failing.Release = func(ctx context.Context, tx pgx.Tx, actionID, targetValue string) error {
		return fmt.Errorf("simulated connector outage")
	}
	return failing
}

// TestScheduler_FailedReleaseSchedulesBackoffRetry: adr/0019 — a failed
// release does not just get retried on the scheduler's normal tick; it
// schedules a 5-second backoff (retrySchedule[0]) and is excluded from the
// next sweep until that time passes, so a fast-ticking scheduler doesn't
// hammer a genuinely down connector every interval.
func TestScheduler_FailedReleaseSchedulesBackoffRetry(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()
	ctx := context.Background()

	caseID := caseWithBlockTarget(t, srv, "scheduler-backoff-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approveApproval(t, srv, findApprovalForAction(t, srv, action.Id).Id).Body.Close()
	executeAction(t, srv, action.Id).Body.Close()
	if _, err := pool.Exec(ctx, `update actions set executed_at = now() - interval '20 minutes' where id = $1`, action.Id); err != nil {
		t.Fatalf("forcing executed_at into the past: %v", err)
	}

	failing := alwaysFailingConnector()
	if _, err := scheduler.Sweep(ctx, pool, failing, testLogger(), nil); err != nil {
		t.Fatalf("first Sweep: %v", err)
	}

	var attempts int
	var nextRetryAt *time.Time
	var escalated bool
	if err := pool.QueryRow(ctx, `select release_attempts, next_retry_at, escalated from actions where id = $1`, action.Id).
		Scan(&attempts, &nextRetryAt, &escalated); err != nil {
		t.Fatalf("reading action retry state: %v", err)
	}
	if attempts != 1 {
		t.Errorf("release_attempts = %d, want 1", attempts)
	}
	if escalated {
		t.Error("expected not escalated after only 1 failure")
	}
	if nextRetryAt == nil || !nextRetryAt.After(time.Now().UTC()) {
		t.Errorf("expected next_retry_at set in the future, got %v", nextRetryAt)
	}

	// A second sweep immediately after must not retry it yet (backoff not
	// elapsed) — release_attempts stays at 1, not 2.
	if released, err := scheduler.Sweep(ctx, pool, failing, testLogger(), nil); err != nil {
		t.Fatalf("second Sweep: %v", err)
	} else if released != 0 {
		t.Errorf("expected 0 released on the immediate second sweep, got %d", released)
	}
	if err := pool.QueryRow(ctx, `select release_attempts from actions where id = $1`, action.Id).Scan(&attempts); err != nil {
		t.Fatalf("reading release_attempts after second sweep: %v", err)
	}
	if attempts != 1 {
		t.Errorf("release_attempts after immediate second sweep = %d, want still 1 (backoff not elapsed)", attempts)
	}
}

// TestScheduler_EscalatesAfterRetriesExhausted: once retrySchedule (5s, 15s,
// 30s) is exhausted, the action is marked escalated, an
// action.release_escalated audit record is written, and the block is
// retained (status stays "executed", never silently unblocked) —
// docs/08-testing/lab-scenario-ssh-bruteforce.md's "retain the block and
// escalate if all retries fail."
func TestScheduler_EscalatesAfterRetriesExhausted(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()
	ctx := context.Background()

	caseID := caseWithBlockTarget(t, srv, "scheduler-escalate-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approveApproval(t, srv, findApprovalForAction(t, srv, action.Id).Id).Body.Close()
	executeAction(t, srv, action.Id).Body.Close()
	// Simulate having already exhausted every scheduled retry.
	if _, err := pool.Exec(ctx, `
		update actions set executed_at = now() - interval '20 minutes', release_attempts = 3
		where id = $1`, action.Id); err != nil {
		t.Fatalf("forcing exhausted retry state: %v", err)
	}

	failing := alwaysFailingConnector()
	if _, err := scheduler.Sweep(ctx, pool, failing, testLogger(), nil); err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	var status string
	var escalated bool
	if err := pool.QueryRow(ctx, `select status, escalated from actions where id = $1`, action.Id).Scan(&status, &escalated); err != nil {
		t.Fatalf("reading action state: %v", err)
	}
	if !escalated {
		t.Error("expected escalated = true after exhausting retrySchedule")
	}
	if status != "executed" {
		t.Errorf("expected the block retained (status still executed), got %q", status)
	}

	auditResp, err := http.Get(srv.URL + "/v1/audit")
	if err != nil {
		t.Fatalf("listing audit: %v", err)
	}
	defer auditResp.Body.Close()
	var page struct {
		Items []struct {
			Action   string `json:"action"`
			Decision string `json:"decision"`
		} `json:"items"`
	}
	_ = json.NewDecoder(auditResp.Body).Decode(&page)
	found := false
	for _, r := range page.Items {
		if r.Action == "action.release_escalated" {
			found = true
			if r.Decision != "completed" {
				t.Errorf("expected decision=completed (audit_log.decision is a closed enum; escalation detail is in Reason), got %q", r.Decision)
			}
		}
	}
	if !found {
		t.Error("expected an action.release_escalated audit record")
	}

	// A sweep after escalation must not keep retrying it — escalated
	// actions are a human's problem now, not the scheduler's.
	if released, err := scheduler.Sweep(ctx, pool, failing, testLogger(), nil); err != nil {
		t.Fatalf("Sweep after escalation: %v", err)
	} else if released != 0 {
		t.Errorf("expected 0 released for an escalated action, got %d", released)
	}
}
