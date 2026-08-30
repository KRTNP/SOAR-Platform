# ADR-0010: Mock IOC enrichment and mock ticketing connectors for M2

## Status

Accepted — implemented in the M2 slice (`services/api/connectors`,
`services/api/enrichment`, `services/api/ticketing`, `web`).

## Decision

Implement `docs/10-roadmap/milestones.md`'s M2 ("enrichment และ ticket mock")
as two connectors that touch only this lab's own Postgres database, never a
real external system:

- **Mock IOC enrichment** (`services/api/enrichment`): looks observables up
  against a static `ioc_indicators` table seeded with RFC 5737/6761 test
  values, and records the result in `enrichment_results`
  (`provider, verdict, confidence, source_reliability, sighting_time,
  raw_reference`, per `docs/05-integrations/threat-intelligence.md`'s
  required fields). Runs synchronously inside the same ingest transaction M1
  already uses — no queue/worker introduced at this milestone.
- **Mock ticketing** (`services/api/ticketing`): opens a `tickets` row for a
  case on an analyst-triggered `open_ticket` case-update operation, carrying
  only case id, severity, owner and a redacted summary — never raw payload or
  credentials, per `docs/05-integrations/ticketing.md`'s explicit
  prohibition. Idempotent per `(tenant_id, case_id)`; closing the case also
  closes its ticket with the same `closure_reason`.

Both connectors implement a shared `services/api/connectors.Connector`
interface (`Manifest`, `HealthCheck`, `Invoke`) and are proven by a single
reusable `connectors.RunContractTests` harness — this is the "connector
contract tests" gate `milestones.md` names for M2, and it is written to be
reused unchanged by every future connector, including M5's real ones.

`api/openapi.yaml`'s `Case` gains optional `ticket_ref`/`ticket_status`, and
`CaseObservable` gains optional `verdict`/`ti_confidence`/`source_reliability`
— additive, backward-compatible, same pattern as ADR-0009.

## Rationale

`docs/10-roadmap/mvp-definition.md` is explicit that M2 needs only a
mock/local IOC provider and a mock ticket connector for its contract-test
gate, "not a production integration" — real STIX/TAXII and real
Jira/ServiceNow connectors are deferred. Building the real thing here would
mean speculative provider-specific auth/retry/rate-limit code with nothing
yet to validate it against, and would pull in scope (real credentials, real
network egress from a lab) this milestone doesn't need.

Ticket creation is analyst-triggered rather than automatic because no
automation-trigger policy exists yet (that is a playbook/approval-gate
concern, arriving with M3); exposing it as a case-update operation reuses the
exact human-in-the-loop point M1 already established for status transitions
and closing, rather than inventing a second interaction model.

## Consequences

- **Contract-test coverage is intentionally partial.** `connector-testing.md`
  lists auth failure, timeout/retry exhaustion, and provider rate-limit as
  things a connector test must cover — none are meaningful for a connector
  with no live network target, so `RunContractTests` covers only what is:
  health-check-is-read-only, malformed-input-is-`validation`,
  target-outside-allowlist-is-rejected, and repeat-idempotency-key-has-no-
  duplicate-effect, plus a manifest self-declaration check
  (`dry_run: true`, non-empty `target_allowlist`). A real M5 connector will
  need the network-dependent cases added back — `RunContractTests` is
  designed to keep working unchanged alongside connector-specific tests for
  those.
- **Manifests are hand-synced, not generated.** `examples/ioc-enrichment-connector.json`
  and `examples/mock-ticketing-connector.json` are the checked-in,
  schema-validated fixtures (`schemas/connector.schema.json`); the Go-side
  `connectors.Manifest` values in `enrichment`/`ticketing` are hand-written to
  match them, since `go:embed` cannot reach `examples/` from `services/api/*`
  (outside the package's own directory tree). This is the same disclosed
  hand-sync risk already accepted for `genapi` vs. `openapi.yaml` in M1 —
  a drift here means the manifest fixture and runtime behavior disagree, not
  that either is individually wrong.
- **Enrichment only covers indicator-shaped observable types**
  (`ipv4, ipv6, domain, url, hash, email`) — `user`/`host`/`process`/`file`/
  `vulnerability` observables get no `enrichment_results` row at all, rather
  than a noisy "unknown" verdict on every observable regardless of whether a
  TI lookup is conceptually meaningful for it.
- **A ticket's evidence is a list of `type:value` observable strings**, not
  linked evidence records — there is no evidence domain yet (that is M4).
- Case list/create responses do not carry `ticket_ref`/verdict data, only
  `GET /v1/cases/{caseId}` does — the same scoping ADR-0009 already applied
  to `observables`, to avoid paying for these joins on every list row.
