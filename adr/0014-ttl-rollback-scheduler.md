# ADR-0014: In-process TTL rollback scheduler for M6

## Status

Accepted — implemented (`services/api/scheduler`), covering the "rollback"
third of M6's gate (`docs/10-roadmap/milestones.md`: "controlled production
pilot", gate "owner sign-off, rollback, KPI baseline"). KPI baseline
reporting (`adr/0015`) and owner sign-off followed the same day; see
`milestones.md`'s M6 row, now Done (lab scope) as of 2026-09-02 — that
closure is explicitly not a claim of production readiness, only of this
repository's own narrower M6 gate for a personal-lab project.

## Decision

Every milestone since M3 disclosed the same gap: nothing automatically
released an executed `block_network_indicator` action once its
`block_ttl_seconds` elapsed (ADR-0011, ADR-0012, ADR-0013). Until this
session, the only way to release one early was the manual
`POST /v1/actions/{id}/release` endpoint (also added this session) — a human
had to remember to call it.

`services/api/scheduler` closes this gap with the smallest slice that proves
it for real:

- `Sweep(ctx, pool, net, logger) (released int, err error)` — one pass.
  Selects candidates directly from `actions` (`network_blocks` is not queried
  directly; its `status`/`released_at` are kept in sync by `net.Release`
  inside `actions.Release`, exactly as the manual release path already
  relies on):
  ```sql
  select id, tenant_id from actions
  where status = 'executed'
    and executed_at + make_interval(secs => block_ttl_seconds) <= now()
  order by executed_at limit 100
  ```
  Each candidate is released in its **own** `store.WithTx` call — never one
  transaction for the whole batch — so one candidate's failure cannot roll
  back another's success. A failure is logged (`slog.Error` with
  `action_id`/`tenant_id`/`err`) and skipped; `Sweep` only returns a non-nil
  error if the candidate query itself fails.
- `Run(ctx, pool, net, interval, logger)` — a `time.Ticker` loop calling
  `Sweep` until `ctx` is done.
- Every release reuses `actions.Release` + `audit.Append` exactly as
  `actions.Handler.Release` already does, with one deliberate difference:
  `Actor: "scheduler"`, `ActorType: "system"` (plain `text not null` columns,
  no check constraint — confirmed against
  `services/api/migrations/20260830000004_m4_evidence_audit.sql`) instead of
  the fixed `"incident-commander"`/`"analyst"` every human-triggered call
  uses. This is the one place the audit trail now distinguishes an automatic
  rollback from a human-requested one.
- Config: `SCHEDULER_ENABLED` (default `true`), `SCHEDULER_INTERVAL_SECONDS`
  (default `30`) in `services/api/config`. `main.go`'s `run()` starts the
  loop bound to the same `signal.NotifyContext` ctx that already drives
  graceful HTTP shutdown, and waits on a `schedulerDone` channel after
  `srv.Shutdown` (and explicitly calls `stop()` before that wait, rather than
  relying on the deferred one) — `defer pool.Close()` runs before the
  deferred `stop()` on the way out of `run()`, so without an explicit,
  earlier cancellation the pool could close while a sweep is still in
  flight.

## Rationale

**No new approval gate on auto-release.** The original approval already
authorized the block for a finite, TTL-bounded duration
(`actions.block_ttl_seconds`, itself part of the approved request) —
releasing it on schedule enforces the boundary that approval already
granted, not new authority. This mirrors the existing precedent that
`POST /release` (ADR-0013's session) already has zero approval gate, only an
`ErrNotExecuted` state check; a scheduled release behaves identically,
triggered by time instead of a human request. Whether this was even an open
question worth deciding explicitly, rather than an obvious default, is
itself worth recording: ADR-0011's own Consequences section only ever said
scheduled rollback was "deferred to M5/M6", not whether it would need
approval — this ADR is the first place that question is actually answered.

**A ticker, not Temporal**, even though `adr/0005`'s selected target stack
lists Temporal for durable workflows. Standing up a workflow engine to run
one polling query is disproportionate at this milestone's scope — the same
mock-now/target-later judgment ADR-0010 through ADR-0013 each already made
explicitly for their own pieces (mock connectors before real ones, no
scheduler before this one). This does not change the Temporal decision;
it defers building it until cross-process coordination, backoff/retry, or
visibility into long-running in-flight work is actually needed.

**No row-level locking** (`SELECT ... FOR UPDATE`, `SKIP LOCKED`, or an
advisory lock) against a human calling `POST /release` for the same action
at the moment the scheduler also picks it as a candidate. Tracing the actual
failure mode: both would fetch the row under READ COMMITTED and see
`status = 'executed'`; both would call `net.Release` (already idempotent by
construction — `network.Release`'s own `where status != 'released'` guard,
and the real OPNsense connector's list-before-mutate check); both would
`update actions set status = 'released'` (itself idempotent, same end
state); both would append an `action.release` audit row. The only
observable effect of the race is one extra, harmless audit row — a class of
duplication `TestReleaseAction_IsIdempotent` already accepts for sequential
double-calls to the same endpoint. Adding locking here would be solving a
problem this codebase has already decided to tolerate elsewhere.

**No new migration or index.** At lab table sizes, a sequential scan
filtered on `status = 'executed'` is trivial.

## Consequences

- **Polling lag up to one `SCHEDULER_INTERVAL_SECONDS` (default 30s).** A
  block can outlive its TTL by up to one interval before the scheduler
  notices. Configurable; a push-based timer (e.g., a real Temporal workflow
  timer) would remove this lag entirely at the cost of the infrastructure
  ADR-0005 already defers.
- **Single-instance assumption.** No distributed or advisory lock across
  processes. Running two API instances against the same database would
  double-run sweeps — harmless per the idempotency argument above, but
  wasteful. A `pg_try_advisory_lock` wrapped around the whole sweep (not
  per-row `SKIP LOCKED`, since the goal would be "only one instance sweeps
  at a time") is the documented upgrade path if this ever runs as more than
  one process; not built now.
- **A permanently-failing candidate retries forever, with no backoff and no
  alerting** beyond the `slog.Error` line on every tick. Acceptable at lab
  scale (a human watching logs will notice); a real production pilot would
  want a retry counter and/or an alert threshold before relying on this.
- **`actor="scheduler"` is a fixed string, not a real authenticated
  principal** — the same "no RBAC yet" gap ADR-0011 already discloses, now
  extended to a second synthetic actor alongside `"incident-commander"`/
  `"analyst"`/`"system"` (ingestion).
- **Concurrent manual-release/scheduler races can produce a duplicate
  `action.release` audit row** (see Rationale) — accepted, not a
  correctness bug, and not expected to be common at lab volume.
- **KPI/metrics reporting, the other half of M6's gate beyond rollback, is
  explicitly deferred** to a separate, later ADR and is not addressed here.
- **A latent test-helper bug was found and fixed in the same change:**
  `services/api/integration/approval_test.go`'s `caseWithBlockTarget`
  returned `page[len(page)-1].Id` from `GET /v1/cases`, which lists newest
  first (`order by id desc` in `cases/store.go`) — every existing test only
  ever called the helper once per fresh database, so picking the oldest
  case instead of the newest never surfaced until this session's
  `TestScheduler_OneFailingCandidateDoesNotBlockTheRest` called it twice
  against the same server and got the same case both times. Now returns
  `page[0].Id`.
