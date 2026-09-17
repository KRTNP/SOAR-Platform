# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

One person: the repository owner, running this as a solo home security lab. They wear
every role the design documents separate — Tier-1 triage, investigator, incident
commander approving containment, DFIR evidence handler, platform operator, and auditor —
and switch between those hats within a single sitting.

`docs/00-context/stakeholders-and-personas.md` defines eight personas. Those describe the
*platform's* separation-of-duties model and drive the RBAC design (adr/0024); they are not
eight people sitting in front of this console. RBAC is still enforced, but the interface
must not fragment into per-persona views that this user would have to log out to reach.

The console is opened in bursts, not watched continuously: the user runs a lab scenario
(an SSH brute-force simulation, a Wazuh alert replay), then opens the console to see what
the system did. The dominant question on arrival is "what just happened, and what is
waiting on me?" — not "what is happening right now."

## Product Purpose

A lab-oriented SOAR console: receive security signals from Wazuh and other sources,
normalize and deduplicate them, enrich with threat intelligence, open and track cases,
gate high-impact response behind explicit human approval, and record an audit and evidence
trail that can be verified and replayed afterwards.

Four jobs carry equal weight, all confirmed as primary:

1. **Triage** — read incoming alerts, decide what is real, open or update a case.
2. **Approve or reject gated actions** — decide on `block`/`isolate`-class requests, with
   the blast radius and rollback path visible *before* the decision.
3. **Inspect evidence and audit trail** — case timeline, who did what and when, whether
   evidence integrity still verifies.
4. **Read system state** — KPI snapshot, connector health, alert volume; confirm the
   pipeline itself is alive.

Success for the first release (`docs/00-context/mission-and-scope.md`): a real Wazuh alert
is ingested, normalized, deduplicated, enriched, opens a case, requests approval, and
records the outcome in the lab — without destroying evidence and with the same events
replayable.

## Positioning

Most SOAR interfaces optimize for throughput across a staffed SOC floor. This one is built
around an auditable contract instead: every response action carries policy evaluation,
human approval, dry-run default, idempotency key, TTL rollback, and an immutable evidence
record — and the console is a view over those contracts (`api/openapi.yaml`, `schemas/`),
not a layer that decorates them.

The design baseline is documentation-first and source-traceable: playbooks are mechanically
validated against required structure, and source-derived playbooks resolve back to catalogued
CISA/NIST references. The console's differentiator is that it can always show *why* an
action was permitted and *what would undo it*, because the system was designed to be unable
to act otherwise.

## Operating Context

- **Session shape:** episodic. Open → scan for what changed → act on what is blocked on a
  human → close. Sessions are minutes, not hours, and a session usually follows a
  deliberately triggered lab scenario.
- **Lab topology** (adr/0006): VMs on a local bridge — Wazuh manager/indexer, OPNsense
  firewall, target hosts. The user is often working alongside a terminal and the OPNsense
  UI, cross-checking that what the console claims actually happened on the box.
- **Backing services:** Go/Gin API and workers, PostgreSQL as source of truth, OpenSearch
  as derived search projection, Keycloak for OIDC, Temporal for durable workflows, Vault
  (opt-in) for lab credentials, Prometheus/Loki/Grafana for observability (adr/0005, 0020,
  0022, 0023, 0024).
- **Evidence handling** is governed by the neighboring `dfir/` workspace: original evidence
  is immutable, derived material is written only to approved work/export locations.

## Capabilities and Constraints

**Real, working against live lab systems**

- Wazuh real alert polling and ingest (adr/0017).
- SSH brute-force correlation with retry escalation and a close gate (adr/0018, 0019).
- OPNsense firewall block / verify / release against the real lab VM, env-gated
  (adr/0013), with a TTL rollback scheduler (adr/0014).
- Keycloak OIDC login and RBAC enforcement (adr/0024).
- Evidence integrity hashing and audit export (adr/0012); KPI snapshot (adr/0015).
- OpenSearch-backed search across cases and events (adr/0022).

**Deliberately mocked** — must remain visibly distinguishable from the real ones:

- IOC enrichment against a seeded local table of RFC 5737/6761 test values (adr/0010).
- Ticketing, which writes a local `tickets` row and never reaches an external tracker
  (adr/0010).

**Hard constraints**

- `safety.default_mode` is `dry_run`; a playbook cannot merge in a state that defaults to
  live action (adr/0003).
- Block, isolate, disable, and delete require a human approval gate. Playbook author and
  approver are separated by policy.
- Fail closed: when data is incomplete or policy cannot be evaluated, stop and escalate.
- The system must function with no LLM in the loop.
- Idempotent by default; repeated event delivery must not repeat an action.
- No secrets in source, workflow YAML, or example JSON. No real customer data, ever.
- Original evidence is never modified.

**Terminology the interface must not rename**

`event`, `alert`, `case`, `observable`, `evidence`, `action` / `pending_action`,
`approval`, `connector`, `playbook`, `tenant`. Case status ladder: `new`, `triage`,
`investigating`, `contained`, `eradication`, `recovery`, `escalated`, `dismissed`,
`closed`. Severity ladder: `informational`, `low`, `medium`, `high`, `critical`. Action
mode is `dry_run` or `execute`. Evidence carries an `integrity_status` of `verified`,
`pending`, `mismatch`, or `unavailable`.

## Evidence on Hand

- A running lab with real Wazuh alerts and a real OPNsense firewall the console has
  actually blocked and released addresses on.
- Real case, evidence, audit, and KPI data produced by lab scenarios.
- `examples/*.json` fixtures asserted against `schemas/*.schema.json` by
  `validation/validate-json.sh`.
- A catalogued source set (`docs/00-context/source-catalog.md`) of CISA, NIST SP 800-61r3,
  OASIS CACAO, OCSF, TAXII, and Sigma references, traced to design in
  `docs/09-governance/source-to-design-traceability.md`.

**Absent — future work must not fabricate these:** customers, testimonials, case studies,
press, pricing, licensing, SLA claims, benchmark numbers, user counts, production
deployments, or any claim that this has been used outside the owner's lab. There is also
no logo, wordmark, or brand asset set.

## Product Principles

1. **Answer "what needs me?" on arrival.** The user arrives after an event, not during
   one. Whatever is blocked on a human decision outranks everything else on screen.
2. **One person, every role.** RBAC is enforced but never used as a reason to hide or
   scatter capability; the interface stays one coherent surface, not eight.
3. **Never outrun the contract.** The console shows only what `api/openapi.yaml` and
   `schemas/` actually return. An honest "unavailable" or "unknown" beats a plausible
   placeholder — this is a system whose entire value is that its record can be trusted.
4. **Consequence before commitment.** Any high-impact action presents its target, blast
   radius, mode (`dry_run` vs `execute`), approval state, and rollback path before it can
   be triggered.
5. **Lab truth over demo polish.** Real connectors and mocked connectors must be
   distinguishable at a glance, and no screen may imply a capability the lab does not have.
