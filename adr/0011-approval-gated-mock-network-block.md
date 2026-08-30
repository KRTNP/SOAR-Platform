# ADR-0011: Approval-gated mock network-block action for M3

## Status

Accepted — implemented in the M3 slice (`services/api/actions`,
`services/api/network`, `web`).

## Decision

Implement `docs/10-roadmap/milestones.md`'s M3 ("approval/deny/expiry",
gated by a "no-action-before-approval test") as a single approval-gated
action intent, `block_network_indicator`, executed by a mock connector that
touches only this lab's own Postgres — never the real OPNsense lab VM
(`docs/05-integrations/firewall.md`). The real OPNsense connector is a
later milestone (`M5 | lab connector | sandbox + tabletop approval`); this
ADR is the same mock-now/real-later split ADR-0010 already used for
enrichment and ticketing.

- **`services/api/network`**: a `connectors.Connector` implementation that
  records a block as a row in `network_blocks`, standing in for the
  OPNsense `SOAR_TEMP_BLOCK` alias.
- **`services/api/actions`**: owns the `actions`/`approvals` state machine —
  `pending_approval → approved/rejected/expired → executed → released`. An
  analyst requests a block from a case (`request_block` case-update
  operation, reusing the human-in-the-loop entry point M1/M2 already
  established for transitions, close, and open_ticket). A separate `approve`
  or `reject` call decides the approval; only `POST /v1/actions/{id}/execute`
  can invoke the connector, and it does so only after re-loading the
  approval inside the same transaction and confirming it is `approved` —
  this re-check, not the caller's say-so, is the actual "no action before
  approval" gate the milestone names.
- **Single approval**, per ADR-0007's already-documented exception for this
  action class at its first vertical scenario — not the dual-approval
  default `docs/04-playbooks/automation-levels.md` sets for automation
  level L4. Approving is idempotent (a repeat call returns the same decided
  approval and never re-executes); rejecting after approval, or approving
  after rejection, is a 409 conflict rather than allowed to flip the
  decision.
- **Two independent TTLs**, both taken from
  `docs/08-testing/lab-scenario-ssh-bruteforce.md`: the approval itself
  expires 15 minutes after being requested if never decided (the scenario's
  analyst review SLA); once executed, the block carries its own 10-minute
  TTL (`actions.block_ttl_seconds`) before it should roll back.
- **`api/openapi.yaml`** fills in the previously-untyped `Action` schema
  (`getAction`, `Case.pending_action`) and adds `approved_by`/`reason`/
  `decided_at` to `Approval` — additive, same pattern as ADR-0009/ADR-0010.

## Rationale

`adr/0003-human-in-the-loop.md` and
`docs/06-security/safe-response-controls.md` require blast-radius-sensitive
actions to stop at an approval bound to the exact action before touching a
target. Proving that gate does not require a real firewall, a scheduler, or
a playbook engine — none of which exist yet (M1's `main.go` already
discloses no Temporal/NATS) — so this milestone builds the smallest slice
that makes the gate itself real and independently testable, the same
scoping judgment ADR-0010 made for M2's connectors.

## Consequences

- **TTL/rollback enforcement is lazy, not scheduled.** There is no
  Temporal/worker yet to expire a stale approval or roll back a block at
  exactly its TTL. Instead, every read or decision path
  (`Get`/`List`/`Approve`/`Reject`/`Execute`) re-evaluates `now` against
  `expires_at` and persists a `pending → expired` transition on the spot.
  A manual `Release` (early rollback) exists so an action isn't stuck
  waiting for a poll, but scheduled rollback at exact TTL and the
  5/15/30-second retry budget `lab-scenario-ssh-bruteforce.md` specifies are
  deferred to M5/M6, when a scheduler exists to drive them.
- **No RBAC-checked distinct approver.** M1 already discloses there is no
  real auth; every approval is recorded under a fixed `"incident-commander"`
  actor rather than a verified second identity. Enforcing "the requester
  cannot also approve" needs real identity (Keycloak), out of scope until
  that lands.
- **Dual approval is not implemented** even though it is automation level
  L4's default — this repeats ADR-0007's already-accepted single-approval
  exception for this specific action class rather than introducing a new
  one.
- **The UI's one "Approve" button performs two API calls** (`approve` then
  `execute`) so an analyst experiences one decision, but the server still
  enforces the gate independently on each call — a client that skipped the
  UI and called `execute` directly without ever approving is still denied.
- **`network.Manifest` is hand-synced**, not generated, matching
  ADR-0010's already-accepted `examples/*-connector.json` vs.
  `connectors.Manifest` disclosure.
- Case list/create responses do not carry `pending_action`, only
  `GET /v1/cases/{caseId}` does — the same scoping ADR-0009/ADR-0010 already
  applied to `observables`/`ticket_ref`.
