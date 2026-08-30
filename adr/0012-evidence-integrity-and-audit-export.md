# ADR-0012: Evidence integrity verification and append-only audit export for M4

## Status

Accepted — implemented in the M4 slice (`services/api/evidence`,
`services/api/audit`, `services/api/cases` export, `web`).

## Decision

Implement `docs/10-roadmap/milestones.md`'s M4 ("evidence/audit export", gated
by a "hash/redaction tests") as two additions that fill in placeholder
contracts already declared in `api/openapi.yaml` at the M0 baseline:

- **`services/api/evidence`**: evidence metadata per
  `schemas/evidence.schema.json` and `docs/09-governance/evidence-and-custody.md`
  — SOAR keeps references, hashes and custody events, never original bytes
  (those live in the external DFIR tree). Every successful event ingestion
  auto-captures one evidence row pointing at the raw payload M1's ingestion
  already stores (`events.raw_payload`/`raw_ref_uri`). `GET /v1/evidence/{id}`
  is the milestone's actual "hash test" gate: it re-reads the linked event's
  stored payload, recomputes its SHA-256, and compares it to the recorded
  hash — never trusting the stored `integrity_status` — persisting a
  `mismatch` on the spot if they disagree. This mirrors M3's
  "re-check inside the same call, don't trust the caller" pattern for its own
  gate.
- **`services/api/audit`**: an append-only `audit_log` table with exactly the
  field list `docs/06-security/audit-and-nonrepudiation.md` specifies
  (`event_id, request_id, actor, actor_type, case_id, execution_id, action,
  target_hash, policy_version, connector_version, decision, reason, timestamp,
  result_hash`). Every existing case/action mutation (create, transition,
  close, open_ticket, request_block, approve, reject, execute, export) writes
  one record. `target_hash` is always a hash of the affected value, never the
  raw value itself — IPs, ticket refs and case ids never appear directly in
  the audit trail.
- **Case export** (`POST /v1/cases/{caseId}/export`) builds a redacted
  manifest synchronously (no export job queue exists yet) and returns it with
  its own SHA-256 (`manifest_sha256`) in the same response — the other half of
  the "hash test" gate: a caller can recompute the hash over the returned
  manifest and confirm it matches. `redaction_profile` is `standard` (default,
  strips each evidence entry's `uri`/`command_ref`) or `full` (includes them);
  `include_evidence_references: false` (an already-declared request field)
  omits the evidence section entirely. `CaseExportRequest` gains optional
  `purpose`/`recipient` fields, written into the export's audit record, per
  `docs/09-governance/privacy-and-retention.md`'s "export ต้อง redaction และ
  บันทึกผู้รับ/วัตถุประสงค์".

`api/openapi.yaml` fills in `Evidence`, `EvidenceManifestRequest` and
`EvidenceCustodyEvent` (previously `additionalProperties: true` on
`getEvidence`/`createEvidenceManifest`) and adds `manifest_sha256`,
`redaction_profile`, `manifest`, `purpose`, `recipient` to the export
request/response pair — additive, the same pattern as ADR-0009/0010/0011.

## Rationale

`docs/06-security/safe-response-controls.md` and
`docs/09-governance/evidence-and-custody.md` both require fixity (hash-backed
integrity) and redaction discipline before evidence or a case manifest can be
trusted or shared outward. Proving that discipline does not require a real
DFIR filesystem, an export job queue, or a scheduler — none of which exist yet
— so this milestone builds the smallest slice that makes verification and
redaction themselves real and independently testable, the same scoping
judgment ADR-0010/ADR-0011 made for their milestones.

A genuine correctness issue surfaced while implementing the verify-on-read
gate: PostgreSQL's `jsonb` type does not preserve object key order or
whitespace, so a hash computed from the pre-insert JSON bytes (as M1's
`events.go` already did, for `raw_ref_sha256`) will not match a hash
recomputed from the same row's `raw_payload` read back later — not because of
tampering, but because Postgres itself re-serializes the object with a
different key order. `evidence.RecordFromEvent` therefore computes its SHA-256
from a fresh read of `events.raw_payload` immediately after insertion, not
from the bytes `events.go` marshaled before the insert — hashing what
verify-on-read will actually re-read keeps the comparison meaningful. This is
disclosed here rather than silently patched because it means `evidence.sha256`
and `events.raw_ref_sha256` are deliberately *not* the same value for the same
row.

## Consequences

- **Verify-on-read only applies to event-linked evidence.** Manually
  registered evidence (`POST /v1/evidence/manifests`, for artifacts whose
  original bytes live in the external DFIR tree) has no local content this
  system can recompute a hash against, so it stays `integrity_status:
  "pending"` forever — this system cannot independently verify it. This is a
  disclosed gap, not a bug: `evidence-and-custody.md` already says original
  evidence lives outside SOAR.
- **List does not re-verify each row.** `GET /v1/cases/{caseId}/evidence`
  returns the stored `integrity_status` without recomputing per row (that
  would cost one extra query per row on every case-detail view); only the
  single-item `GET /v1/evidence/{id}` re-verifies. A caller that needs a fresh
  answer for every item on a case must call `GET` on each one.
- **The audit log self-hashes per record, not as a chain.** `result_hash`
  lets a reader detect a tampered *individual* row by recomputing it from the
  row's own fields, but does not detect a *deleted* row the way a hash chain
  (each record's hash including the previous record's hash) would. Chaining
  needs a single-writer/strict-sequence guarantee this lab does not have yet
  and is deferred to M5/M6 alongside the scheduler work.
- **Append-only is enforced by omission, not DB grants.** The `audit` package
  exposes only `Append` and `List` — there is no update or delete function at
  all. Nothing stops a direct SQL `UPDATE`/`DELETE` against `audit_log` today;
  real enforcement needs a restricted DB role, deferred alongside the rest of
  this lab's "no RBAC yet" gaps (ADR-0011).
- **A denied execute attempt is audited independently of its rolled-back
  transaction.** `actions.Handler.Execute` runs the actual execute inside
  `store.WithTx`; on an approval-gate denial that transaction rolls back, so
  an audit insert made inside it would vanish too. The denial record is
  written as a second, best-effort statement against the pool after the 403
  has already been decided — a logging failure there does not change the
  response the caller already received.
- **CSV export is deferred.** Only `format: "json"` is implemented;
  `format: "csv"` returns 422. The milestone's gate is about hashing and
  redaction, not a spreadsheet format.
- **No RBAC gates the `full` redaction profile.** Any caller can request
  `redaction_profile: "full"` today — the same class of gap ADR-0011 already
  disclosed for single-approval/no-distinct-approver, deferred to when
  Keycloak lands.
- **No UI exists for `/v1/audit` or case export.** Both are compliance/ops-
  facing, not part of the existing analyst-facing case page, and the
  milestone's gate is provable entirely at the API layer. Only a read-only
  evidence panel was added to the case detail page.
