# ADR-0015: KPI snapshot endpoint for M6

## Status

Accepted — implemented (`services/api/metrics`, `GET /v1/metrics`). This
covers the "KPI baseline" third of M6's gate
(`docs/10-roadmap/milestones.md`: "controlled production pilot", gate
"owner sign-off, rollback, KPI baseline"), alongside ADR-0014's rollback
scheduler. The owner (KRTNP) gave sign-off for this project's own lab scope
on 2026-09-02, closing `milestones.md`'s M6 row as Done (lab scope) —
explicitly not a claim that a real, time-accumulated KPI baseline exists
yet, only that the *capability* to compute one from recorded data is built
and the owner accepts that as sufficient for a personal lab.

## Decision

`services/api/metrics.Compute(ctx, pool, tenantID)` computes a `Snapshot` —
a point-in-time read of whichever KPIs from
`docs/07-operations/metrics-and-kpis.md` are honestly derivable from data
this system *already* records, with straightforward SQL aggregates over
`audit_log`/`actions`/`cases`/`evidence` (no new tables, no time-series
store):

- **connector reliability** — `action.execute` count grouped by
  `connector_version`/`decision`.
- **analyst override** — count of `approval.reject` audit records.
- **auto vs. manual release** — count of `action.release` split on
  `actor = 'scheduler'` (ADR-0014) vs. everything else.
- **mean time to containment** — average seconds between a case's
  `opened_at` and its first allowed `action.execute` audit record.
- **evidence completeness** — cases with at least one `evidence` row whose
  `integrity_status = 'verified'`, over total cases.

`GET /v1/metrics` (`services/api/metrics.Handler.Get`) returns this as
plain JSON; `api/openapi.yaml` declares the path with an open
(`additionalProperties: true`) response schema rather than a named
component — this snapshot is expected to grow as more of
`metrics-and-kpis.md`'s list becomes computable, and pinning a strict schema
now would just mean revising it every time.

`Snapshot` also carries `connector_target`/`connector_version` — the
currently-wired `actions.NetworkConnector`'s identity (`main.go`'s
`buildNetworkConnector` result), reported unconditionally rather than only
derived from `connector_reliability` (which is empty until something has
actually executed). Added alongside the M6 frontend work
(`web/app/metrics`, a Sidebar connector badge) so the UI has a reliable way
to show whether the mock or the real OPNsense connector (adr/0013) is
currently active, without depending on execution history existing yet.

**Not implemented, and disclosed rather than faked:** `metrics-and-kpis.md`
also lists MTTT (alert-to-triage — no triage timestamp exists anywhere in
the schema), false-positive harm (needs a manual "this action was wrong"
tag that doesn't exist), and automation coverage (needs a "would have been
manual" baseline this system has no way to know). These are left out of
`Snapshot` entirely rather than approximated with a misleading proxy.

## Rationale

Proportionate to what this milestone and this project actually are — a
personal lab, not a KPI platform. A dedicated metrics/time-series store,
historical trending, or per-tenant dashboards would all be solving problems
this lab doesn't have; a handful of `SELECT ... GROUP BY` queries answering
"what do the numbers say right now" is enough to prove the gate is
buildable from already-recorded data, matching the same minimal-slice
judgment every prior milestone's ADR already made for its own piece.

## Consequences

- **This is a snapshot, not a baseline.** A KPI *baseline* needs numbers
  observed over real operational time; running this against a lab database
  with a handful of test cases produces numbers, but not ones worth
  reporting to an owner as "the baseline". `milestones.md`'s M6 row reflects
  that distinction.
- **No historical trend, no time range, no per-tenant breakdown.** One
  global snapshot for the caller's tenant, computed fresh on every request —
  fine at lab query volume; would need caching or a real metrics backend
  (Prometheus, per `adr/0005`'s selected observability stack) before this
  could be queried often or by a dashboard.
- **MTTT, false-positive harm and automation coverage are not reported at
  all**, not approximated — see Decision. Adding them for real means adding
  the underlying data first (a triage timestamp, a manual outcome tag), not
  a metrics-package change.
