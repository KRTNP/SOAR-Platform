# ADR-0019: SSH brute-force backoff retry, priority escalation, and close gate

## Status

Accepted — implemented and proven end-to-end. Completes the three items
ADR-0018 explicitly deferred from `docs/08-testing/lab-scenario-ssh-bruteforce.md`'s
full acceptance spec.

## Decision

### 1. 5/15/30-second backoff retry + escalation on rollback failure

`services/api/scheduler` previously retried a failed release on every tick indefinitely
(ADR-0014's disclosed "no backoff/alerting" gap), not the scenario's specific "retry rollback
at 5, 15 and 30 seconds; retain the block and escalate if all fail."

- `actions.release_attempts`/`next_retry_at`/`escalated` (new columns) track per-action retry
  state. `scheduler.retrySchedule = []time.Duration{5s, 15s, 30s}` — `recordFailure` (called as
  its own statement against the pool, not inside the already-rolled-back release transaction,
  the same posture `actions/handler.go`'s `auditDeniedExecute` already uses) schedules the next
  attempt via `next_retry_at`, or once `release_attempts >= len(retrySchedule)`, calls
  `escalate`: marks `escalated = true`, writes an `action.release_escalated` audit record
  (`Decision: "completed"` — `audit_log.decision` is a closed enum with no "escalated" member;
  the actual detail is in `Reason`), and leaves `status` at `"executed"` — the block is
  retained, never silently removed, and `candidateQuery` excludes `escalated = true` rows so
  the scheduler stops touching it; a human must call the existing manual `POST
  /v1/actions/{id}/release` to resolve it.

### 2. Priority escalation for privileged users / critical assets

"Escalation: Medium becomes higher priority only when source targets a privileged user or
critical asset." New `services/api/criticaltargets` package + `critical_targets` table (same
shape as `services/api/allowlist`'s `trusted_sources`, but an unrelated concern — one says
"never block this source," the other says "this destination deserves higher priority") +
`GET`/`POST /v1/critical-targets`.

`services/api/events/handler.go`'s correlation path (adr/0018) checks any `user`/`host`
observable an event carries against `criticaltargets.IsCritical`; if matched and the case's
current severity ranks below `"high"` (`cases.SeverityRank`), `cases.EscalateSeverity` raises
it and a `priority_escalated` timeline entry records the before/after. Scoped to the same one
correlated signature as adr/0018's auto-trigger — not a general "any case touching a critical
target escalates" rule — and checked from the first correlated event, not gated on the 5-event
auto-block threshold (escalation is about *who*, not *how many times*).

### 3. Close gate: cannot close a case with a block still in effect, or while a repeated alert may still arrive

"Case closes only after rollback succeeds, no repeated alert is present, and analyst review is
recorded" — all three clauses, not just the first:

- **Rollback succeeded**: new `actions.HasLiveAction(tenantID, caseID)`
  (`select exists(... where case_id = $1 and status = 'executed')`) — checked in
  `cases/handler.go`'s `close` operation before `Close` runs; if any action on the case is still
  `"executed"` (not yet released), returns `httpx.ErrConflict("case_has_live_action", ...)` (409).
  Deliberately scoped to `status = 'executed'` only, not `pending_approval`/`approved` too — an
  analyst rejecting-and-closing a case with an undecided approval must still be able to; those
  statuses haven't touched real infrastructure, so there is nothing to "roll back" yet.
- **No repeated alert is present**: a second check, only for a case that actually reached the
  auto-block threshold (`repeat_count >= 5`, the same threshold `events/handler.go`'s auto-trigger
  uses, hand-synced as a local constant since `cases` must not import `events`) — if
  `now < updated_at + correlation_window_seconds`, closing is rejected
  (`case_correlation_window_open`, 409), since another matching alert could still arrive. The
  first version of this checked *any* correlated case (`correlation_key != nil`), not just ones
  that reached the threshold — that broke closing a case with only 1-2 low-signal correlated
  events almost immediately after creation, completely normal usage, caught by
  `TestCaseClose_RejectsWhileActionExecuted`'s own regression during this same change. Narrowing
  to the auto-block threshold fixed it; `TestCaseClose_RejectsWhileCorrelationWindowOpenAfterAutoBlock`
  covers the actually-intended case (reject while the window is open, succeed once it has
  elapsed — forced via `updated_at` the same way TTL tests force `executed_at`).
- **Analyst review is recorded**: no new mechanism — `close` already requires a non-empty
  `closure_reason` from a human-attributed actor, timestamped (`closed_at`) and audited
  (`case.close`). That already *is* a recorded analyst review in every sense this codebase has a
  concept for; adding a separate "reviewed" flag on top would be a duplicate of data this
  operation already captures, not a new precondition.

## Rationale

Each of these three was named explicitly in the same scenario doc ADR-0018 already drew the
correlation/auto-block/allowlist slice from, and each maps to already-built primitives
(`scheduler`'s existing retry loop, `allowlist`'s exact CRUD shape, `actions.Release`'s existing
status model) rather than new architecture — the same "smallest slice, reuse what exists"
posture this repo's ADR sequence has held throughout.

## Consequences

- **The correlation-window close-gate only applies to cases that reached the auto-block
  threshold**, not every correlated case — a case with 1-2 correlated events that never
  auto-triggered a block can be closed immediately, matching normal usage rather than the
  literal "any repeated alert at all."
- **Escalation only ever raises severity, never lowers it** (`EscalateSeverity` is a plain
  write; the caller's `SeverityRank` comparison is the only guard) — there is no de-escalation
  path if a critical-target match turns out to be a false positive; an analyst would need to
  manually transition the case's severity down (not currently exposed as its own operation —
  severity isn't one of `Handler.Update`'s editable fields today).
- **The backoff schedule is fixed at 5/15/30s**, not configurable — matches the scenario
  literally; a different signature needing a different schedule would need its own constant or
  a config knob, not implemented here.
- **No frontend UI for `/v1/critical-targets`**, same disclosed gap ADR-0018 already carries for
  `/v1/allowlist`.
- No auth/RBAC (adr/0005): `POST /v1/critical-targets`'s `created_by` is a fixed `"analyst"`
  string, and the escalation/retry/close-gate audit trail all record under the same fixed actor
  strings (`"soar-ingestion"`, `"scheduler"`, `"analyst"`) every other write path already uses.
