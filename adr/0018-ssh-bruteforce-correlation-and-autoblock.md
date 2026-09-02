# ADR-0018: SSH brute-force correlation, auto-block, durable Wazuh checkpoint, and closed-case terminal fix

## Status

Accepted — implemented and proven end-to-end (`TestIngestion_FiveAuthFailuresFromOneIPAutoRequestsBlock`,
`TestIngestion_AllowlistedSourceNeverBlocked`, `TestCaseTransition_RejectsOnceClosed`,
`TestWazuhCheckpoint_PersistsAcrossRuns`, all in `services/api/integration`).

## Decision

This is three related pieces of hardening from the same review session, bundled into one ADR
because two of them (correlation, checkpoint persistence) close gaps this repo's own earlier
ADRs already disclosed, and the third (closed-case terminal fix) was found while testing the
first two.

### 1. Event correlation + SSH brute-force auto-block

`playbooks/ssh-bruteforce-response.yaml` (this project's "first vertical scenario," ADR-0007)
and its tabletop spec (`docs/08-testing/lab-scenario-ssh-bruteforce.md`) describe a real
playbook engine this repo does not have and, on inspection, cannot realistically get without
redesigning `schemas/playbook.schema.json` itself — `trigger.condition` and `steps[].purpose`
are free-text English sentences, not machine-executable fields, across all 16 files in
`playbooks/`. A generic playbook interpreter is out of scope here.

What was achievable: `services/api/events/handler.go`'s `Ingest` created a brand-new case for
*every* event, unconditionally — `cases.correlation_key` (and `alerts.correlation_key`/
`alerts.correlation_window_seconds`) have existed in the schema since the initial migration but
were never written by any Go code, confirmed by grep before this change. `cases.NewFromAlert`'s
own doc comment already said "a future correlation engine may instead attach a new alert to an
existing open case." This ADR finishes that one signature:

- `correlationKeyFor(eventClass, observables)` (`services/api/events/handler.go`) returns a key
  only for `event_class == "authentication"` with an `ipv4` observable — every other event class
  is unchanged, still always opens a new case.
- `cases.FindOpenByCorrelation` looks for a still-open case with the same key opened within
  `correlation_window_seconds` (60, matching the scenario's "5 failed SSH logins within 1
  minute"). A match attaches the alert/observables to that case and increments the new
  `cases.repeat_count` column instead of inserting a new case; a timeline entry uses the
  existing `alert_linked` kind (already defined in `web/components/Timeline.tsx`'s label map,
  previously unused) rather than `case_opened`.
- Once `repeat_count >= 5` on a correlated case, and the source IP is not on the allowlist
  (below), `services/api/events/handler.go` calls `actions.RequestBlock` itself — inside the
  same ingestion transaction, using `actions.EvaluatePolicy` (adr/0016) for the audit reason
  exactly like the manual `request_block` case-update operation does. This is genuinely "a
  playbook step became code," scoped to one signature: an analyst no longer clicks "Request
  block" for repeated SSH-style auth failures from one IP. Everything downstream — approval,
  execute against the real OPNsense connector (adr/0013), scheduled release (adr/0014) — is
  unchanged, already-built code.
- New `services/api/allowlist` package + `trusted_sources` table + `GET`/`POST /v1/allowlist`
  implement the scenario's "Trusted sources: Management/admin allowlist cannot be blocked;
  allowlist changes are audited and expire." No frontend page yet — seeded via curl/API only.

### 2. Durable Wazuh poll checkpoint

`services/api/wazuhpoll.Run`'s `since := time.Now().UTC()` was in-memory only, disclosed as a
gap in ADR-0017 ("a process restart resets the checkpoint to 'now'"). New single-row
`wazuhpoll_checkpoint` table (`id boolean primary key default true check (id)`, the same
settings-singleton trick); `LoadCheckpoint`/`SaveCheckpoint` read/write it, `Run` now takes the
`*pgxpool.Pool` it needs to do so. First-ever run (no row yet) still falls back to `time.Now()`
— a fresh database still doesn't backfill history.

`services/api/scheduler`, by contrast, needed no such fix: it queries `actions` by absolute
expiry time on every sweep and was already stateless/restart-safe — this ADR corrects the
earlier framing (from this session's own priority-ranking discussion) that lumped it in with
the poller's gap.

### 3. Closed case is now a terminal state

Found while testing the above: `services/api/cases/store.go`'s `Transition` applied an
unconditional `update ... set status = $3` regardless of the row's current status.
`cases/handler.go`'s `Update` already rejected transitioning *to* "closed" directly (must use
the `close` operation, which requires `closure_reason`) but never rejected transitioning *away
from* an already-closed case. The `Transition` case-block now checks the already-fetched
`before.Status` and returns `httpx.ErrConflict("case_already_closed", ...)` (409) before calling
`Transition` at all. `web/app/cases/[caseId]/status-control.tsx`'s `StatusControl` now renders
nothing once `caseItem.status === "closed"`, matching `CloseCaseControl`'s own existing
closed-state guard a few lines below it, instead of showing a control that can only ever 409.

## Rationale

Each piece follows this repo's established "smallest slice that proves it for real" pattern
(ADR-0010 through 0017): correlation only covers the one signature this project's own first
vertical scenario names, not a general engine; the checkpoint fix is the minimum needed to stop
losing state across a restart; the closed-case fix is the minimum needed to make "closed" mean
what it says. None of the three required inventing new architecture beyond what
`schemas/case.schema.json`, `docs/08-testing/lab-scenario-ssh-bruteforce.md`, and ADR-0017
already specified or disclosed.

## Consequences

- **Not implemented from the scenario's full acceptance spec**: 5/15/30-second
  backoff-scheduled rollback retry (the scheduler still retries every tick, per ADR-0014,
  not on that specific schedule); privileged-user/critical-asset priority escalation (no
  asset-criticality model exists); auto-close gating on verified rollback (closing stays a
  manual, analyst-driven operation). Each is a separate, smaller follow-up.
- **`correlationKeyFor`'s key shape (`event_class:ip`) is specific to this one signature**, not
  a general correlation engine — a second signature needing correlation picks its own key
  format and window rather than reusing this constant.
- **The allowlist has no frontend UI** — `GET`/`POST /v1/allowlist` exist, but adding/viewing
  trusted sources today means calling the API directly.
- **No auth/RBAC** (adr/0005, still not built): the auto-trigger's audit trail records
  `Actor: "soar-ingestion", ActorType: "system"` (distinguishing it from a human-requested
  block, the same way `services/api/scheduler`'s `Actor: "scheduler"` already does for
  automatic releases), and `POST /v1/allowlist`'s `created_by` is a fixed `"analyst"` string —
  the same disclosed gap every other write path in this codebase already carries.
- **Existing integration tests that called `caseWithBlockTarget` twice against the same server**
  needed a class-parameterized variant (`caseWithBlockTargetClass`) to keep testing "two
  independent cases" now that same-IP `authentication` events correlate by design — only
  `TestScheduler_OneFailingCandidateDoesNotBlockTheRest` needed this; every other caller invokes
  the helper once per test and was unaffected.
