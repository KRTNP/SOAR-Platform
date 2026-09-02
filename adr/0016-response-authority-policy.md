# ADR-0016: Response-authority policy for approval roles

## Status

Accepted — implemented (`services/api/actions/policy.go`,
`examples/policy.default.json`, `web/app/approvals`).

## Decision

Give `schemas/policy.schema.json` — present since the design baseline but
never implemented, exampled, or referenced by `services/api` — an actual
fixture and a first caller, so "which role should approve this class of
action" is a data-driven table instead of an unwritten assumption baked
into a hardcoded actor string.

- **`examples/policy.default.json`**: one rule per `action.schema.json`'s
  `intent` enum value, each an `allow` or `require_approval` decision with a
  human-readable `reason`. Low-risk, read-only or easily-reversible intents
  (`notify`, `query_siem`, `create_ticket`, `enrich_observable`) are `allow`;
  the rest — including `block_network_indicator`, the only intent actually
  wired end-to-end today — require approval.
- **`services/api/actions/policy.go`**: `defaultPolicy`, a Go map mirroring
  the same rules by hand, and `EvaluatePolicy(intent, risk) (decision string,
  approvalRoles []string)`, a pure function with no I/O. It is **not** loaded
  from `examples/policy.default.json` at runtime: that file sits outside
  `services/api/go.mod`'s module tree (`examples/` is a repo-root sibling of
  `services/api/`, and `//go:embed` cannot reach outside a module's own
  directory tree), so a JSON-file-in-service approach would need either a
  duplicate copy inside `services/api/` or a build step to stage one — both
  disproportionate to a ten-rule table at this scope. The schema/example
  pair stays the design-level contract; the Go map is its by-hand mirror,
  same posture ADR-0011 already accepted for `network.Manifest` vs.
  `examples/*-connector.json`.
- **One caller so far**: `services/api/cases/handler.go`'s `request_block`
  case-update operation calls `actions.EvaluatePolicy(a.Intent, a.Risk)`
  right after `actions.RequestBlock` creates the action, and records the
  resolved `approvalRoles` as the `action.request_block` audit entry's
  `reason` (previously always `nil`) — e.g. `"requires approval from:
  incident-commander"`. This is the only intent with a real create-and-audit
  path to hang the call on; the other nine intents in the policy exist as a
  complete design artifact ahead of their own implementation, same as most
  of this repo's schemas today.
- **`web/app/approvals`**: a new page reading the already-implemented
  `GET /v1/approvals` (wired since M3 but never given frontend UI), resolving
  each approval's linked action via a new `getAction()` client call to show
  intent/target/risk/case — the queue view this policy's roles are meant to
  be read against.

## Rationale

A commercial reference SOAR's own "Response authority" screen (reviewed this
session, `https://demo.soar.zcr.ai/config/response-authority`) answers "how
should the UI differ per user group" not with separate views per role, but
with one data-driven table: action class → requires-approval? → approver
role → on-timeout behavior. `policy.schema.json`'s `rules`/`approval_roles`
shape already anticipated exactly this, just with nothing behind it. Writing
the fixture and wiring one real caller is the smallest slice that makes the
artifact real instead of aspirational — the same standard ADR-0011 through
ADR-0015 have each held their own pieces to.

## Consequences

- **`approvalRoles` is descriptive, not enforced.** There is no
  identity/authz system yet (`adr/0005` selects Keycloak; not built) to
  check a caller's actual role against `approval_roles` before letting them
  hit `POST /v1/approvals/{id}/approve`. Every approval is still decided
  under the fixed `"incident-commander"` actor ADR-0011 already disclosed.
  This ADR only makes the *recommended* role data-driven and visible on the
  audit trail — the same "flag enforcement is not wired" honesty pattern the
  reference product itself uses on its own System page for an analogous gap.
- **No new column, no migration.** `approvalRoles` is surfaced through the
  existing `audit.Record.Reason` field, not a new field on `Action` or
  `Approval`. A future real enforcement pass will likely need a persisted
  column (e.g. `actions.required_role`) rather than a free-text audit
  reason; that is out of scope here.
- **Nine of ten intents are policy-only.** `collect_artifact`,
  `quarantine_email`, `revoke_session`, `isolate_host`, `disable_account`,
  and the three `allow` intents have a rule in the table but no
  `RequestX`-style creation path in `services/api/actions` yet — the same
  gap that already exists for those intents independent of this ADR.
- **`on_timeout` (the reference product's third policy column) is not
  modeled.** `policy.schema.json` has no field for it and this ADR does not
  add one; ADR-0011's existing lazy-expiry-to-`expired` behavior is the only
  timeout behavior this lab implements, uniformly, regardless of intent.
- **`GET /v1/approvals`'s response schema stays the thin `Approval` shape**
  (`id`, `action_id`, `status`, …) — no `intent`/`target`/`risk` were added
  to it. `web/app/approvals` resolves those by a second call per row
  (`getAction`), accepting N+1 calls at lab scale rather than changing the
  openapi contract for one new page.
