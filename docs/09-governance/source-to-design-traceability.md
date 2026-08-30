# Source-to-design traceability

This matrix closes the gap between “we read a source” and “the source changed the
design”. IDs in this document are design anchors. They are intentionally independent
of any one vendor implementation and can be carried into requirements, code, or
control evidence later.

## Traceability matrix

| Anchor | Source evidence | Requirement / control decision | Playbooks or artifacts | Verification |
|---|---|---|---|---|
| TR-IR-LIFECYCLE | SRC-NIST-61R3, SRC-CISA-IRVULN | FR-IR-001: model preparation, detection, response, recovery, communication, and lessons learned as linked states | `docs/01-requirements/maturity-model.md`, `docs/02-architecture/execution-semantics.md`, all incident playbooks | `test-matrix.md` lifecycle and recovery rows; tabletop closure record |
| TR-COORDINATION | SRC-NIST-61R3, SRC-CISA-IRVULN, SRC-CISA-TTX | FR-CASE-001: case has commander/owner, task assignment, communications, decisions, approvals, and escalation | `docs/03-domain-model/case-model.md`, `docs/04-playbooks/approval-model.md`, `playbooks/cisa-major-incident-coordination.yaml` | approval, escalation, communication failure, and tabletop fixtures |
| TR-EVIDENCE | SRC-NIST-61R3, SRC-CISA-RANSOM, SRC-LOCAL-DFIR | CTRL-EVD-001: preserve raw reference, hash, custody, collector, timestamps, retention, and read-only boundary before destructive response | `docs/03-domain-model/evidence-model.md`, `docs/09-governance/evidence-and-custody.md`, `playbooks/dfir-evidence-preservation.yaml` | hash match/mismatch, unavailable evidence, and custody continuity tests |
| TR-RECOVERY | SRC-NIST-61R3, SRC-NIST-800184, SRC-CISA-RANSOM | FR-REC-001: recovery requires known-good source, dependency order, validation, monitoring, communications, and lessons learned | `docs/07-operations/backup-and-recovery.md`, `playbooks/nist-recovery.yaml`, `playbooks/ransomware-response.yaml` | restore drill, post-condition mismatch, and recovery tabletop |
| TR-VULN | SRC-CISA-IRVULN | FR-VULN-001: vulnerability response separates intake, validation, prioritization, mitigation, remediation, verification, and exception/closure | `docs/05-integrations/connector-catalog.md`, `playbooks/cisa-vulnerability-response.yaml` | duplicate advisory, asset unknown, SLA breach, patch verification failure |
| TR-PLAYBOOK-STRUCTURE | SRC-CACAO-20 | CTRL-PB-001: playbook metadata, explicit workflow states, branch/loop/parallel semantics, version and environment adaptation are first-class | `docs/04-playbooks/playbook-standard.md`, `docs/02-architecture/execution-semantics.md`, `schemas/playbook.schema.json` | YAML contract, branch/timeout/retry/replay tests |
| TR-PLAYBOOK-SAFETY | SRC-CACAO-20, SRC-SOAR-USE | CTRL-SAFE-001: read/enrich is separated from write/contain; high-impact actions require explicit approval and safe default | `docs/04-playbooks/approval-model.md`, `docs/06-security/safe-response-controls.md`, all YAML playbooks | dry-run, approval expiry, policy deny, and blast-radius tests |
| TR-NORMALIZATION | SRC-OCSF, SRC-WAZUH-API | FR-ING-001: provider payload is retained as redacted source evidence and mapped to canonical event with provenance | `docs/03-domain-model/event-model.md`, `docs/01-requirements/data-source-coverage.md`, `playbooks/wazuh-alert-triage.yaml` | Wazuh fixture, unknown field, missing timezone, checksum verification |
| TR-CTI | SRC-TAXII-21, SRC-OCSF | NFR-CTI-001: connector supports discovery, auth, pagination, filtering, timeouts, provider status, and object lifecycle | `docs/05-integrations/threat-intelligence.md`, `docs/05-integrations/error-taxonomy.md` | pagination, 401/403, timeout, rate limit, stale verdict fixtures |
| TR-DETECTION-CORRELATION | SRC-SIGMA-CORR | FR-DETECT-001: correlation declares rules, grouping key, time window, condition, false positives, and severity adjustment | `docs/01-requirements/detection-engineering.md`, `docs/03-domain-model/relationship-and-correlation.md`, `playbooks/credential-stuffing.yaml` | ordered/unordered temporal, threshold boundary, and false-positive tests |
| TR-OPS-CONTEXT | SRC-OIIRP | CTRL-OPS-001: response decision records business dependency, critical threshold, expected operational impact, and owner | `docs/04-playbooks/response-decision-matrix.md`, `playbooks/cisa-major-incident-coordination.yaml`, `playbooks/ransomware-response.yaml` | critical dependency branch and unknown-impact stop test |
| TR-HITL-QUALITY | SRC-SOAR-USE, SRC-ICSECURE | NFR-HUMAN-001: analyst decision assistance, task quality/completeness, overrides, and overautomation indicators are measurable | `docs/06-security/ai-safety.md`, `docs/07-operations/metrics-and-kpis.md`, `docs/08-testing/test-strategy.md` | analyst review, ticket completeness, override rate, and tabletop findings |
| TR-CONNECTOR-DISCOVERY | SRC-APIRO, SRC-SHUFFLE-WF | FR-CONN-001: connector capabilities, API operation, auth scope, idempotency, error mapping, and sandbox contract are explicit | `docs/05-integrations/provider-capability-matrix.md`, `docs/05-integrations/integration-framework.md` | connector contract and mock/sandbox tests |
| TR-LOCAL-FORENSICS | SRC-LOCAL-DFIR | CTRL-DFIR-001: original evidence never receives writes; derived artifacts have case/work/export location, hashes, collection order, and stop conditions | `docs/09-governance/evidence-and-custody.md`, `playbooks/dfir-evidence-preservation.yaml`, `SECURITY.md` | path-boundary, hash, custody, and authorized-lab tests |

## Coverage summary

| Source class | Sources in catalog | Trace anchors | Coverage interpretation |
|---|---:|---:|---|
| Normative standards/guidance | 5 | 10 | Used for schema, protocol, lifecycle, and control baselines |
| Practitioner playbooks | 6 | 8 | Used for operational sequences, communications, evidence, recovery, and exercises |
| Research | 4 | 3 | Used for human-factors, operational-context, and connector-design hypotheses |
| Local operational guidance | 1 | 2 | Used for KRTN evidence boundary and DFIR collection requirements |

## Unresolved environment decisions

These are intentionally not guessed from public sources:

- legal notification time windows and jurisdiction-specific reporting;
- asset criticality and operational safety thresholds;
- provider-specific fields, API versions, rate limits, and sandbox identities;
- role ownership, on-call roster, approval quorum, and emergency authority;
- retention periods, tenant partitioning, and data residency;
- measured SLO/KPI baselines and acceptable false-positive/false-negative rates.

They must be resolved by the system owner before enabling write actions. This is a
completeness boundary, not an omission from the source review.
