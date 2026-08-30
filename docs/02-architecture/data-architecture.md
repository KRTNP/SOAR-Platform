# Data architecture

The filesystem paths below are relative to the `soar-platform/` project root.

## System of record

PostgreSQL is authoritative for all mutable domain state. OpenSearch is a derived
projection for search and dashboards. Local filesystem is authoritative only for lab
evidence bytes; its manifest and custody state are recorded in PostgreSQL.

## PostgreSQL aggregates

| Aggregate | Core fields | Consistency rule |
|---|---|---|
| Event | source, source_event_id, occurred_at, raw reference, schema version, provenance | Unique source/idempotency key per tenant |
| Alert | detection, severity, confidence, status, related event IDs | Alert updates produce audit/outbox events |
| Case | owner, severity, SLA, status, scope, related alerts | State transition checked by policy and role |
| Observable | type, value hash/redaction, source, confidence, sightings | Sensitive values are access-scoped |
| Evidence manifest | artifact ID, path class, hash, collector, custody, retention | Original path is never a write target |
| Playbook | ID, version, status, source refs, checksum, owner | Execution pins immutable version |
| Execution | playbook/policy/connector versions, status, inputs/outputs refs | Workflow state is reconciled from Temporal events |
| Approval | action hash, target, risk, approver, expiry, decision | Approval is invalid if request hash changes |
| Action | intent, target, provider operation, post-condition, rollback | Provider command is generated inside connector |
| Policy | scope, thresholds, risk rules, version, effective time | Evaluation result is immutable audit data |
| Audit | actor, operation, target, decision, hash chain | Append-only; no delete API |
| Outbox event | aggregate, event type, payload reference, sequence, published state | Written in same transaction as domain mutation |

## OpenSearch projections

Use versioned indexes such as `soar-events-v1`, `soar-alerts-v1`, `soar-cases-v1`,
`soar-observables-v1` and `soar-audit-v1`. Each document carries `tenant_id`, source
version, PostgreSQL ID, projection version and `indexed_at`. Indexer processing is
idempotent by PostgreSQL event ID and can rebuild an index from outbox/replay data.

OpenSearch must not be used as the only copy of evidence, approval decisions, policy
versions or audit chain state.

## Evidence filesystem contract

```text
../dfir/
├── evidence/<case>/                         # original, read-only
├── cases/<case>/work/{intake,parsed,derived,recovered,correlation,scratch}/
├── exports/<case>/{reports,indicators,artifacts}/
└── cases/<case>/logs/{evidence.sha256,custody,manifests}/
```

The path class is stored with every artifact in the SOAR evidence manifest. The ingest
operation calculates the configured hash before marking an original artifact read-only.
Analysis writes only to `cases/<case>/work/`; exports are redacted and permission-checked.
The SOAR service stores references and metadata; it does not rewrite the DFIR evidence
tree.

## Retention and deletion

Retention is policy-driven and recorded per tenant/case/artifact. A legal hold or
active case prevents deletion. Deletion, when authorized, creates a tombstone audit
event and never silently removes the custody record. The lab starts with short
retention and synthetic data; production periods require owner approval.
