# SOAR Platform Design

Design documents for a Security Orchestration, Automation and Response (SOAR) system that connects SIEM, EDR, firewall, identity, threat intelligence, ticketing, and DFIR into a single response process.

This repository is a design baseline. Nothing in it has permission to act on production systems yet. All examples must run in a lab or in dry-run mode until authorization, approval, and rollback are clearly defined.

## Goals

- Cut analyst triage time and the amount of switching between tools
- Make incident response repeatable, auditable, and measurable
- Keep investigative actions separate from containment and remediation actions
- Preserve evidence and chain of custody following the `dfir/` workspace guidelines
- Use a common schema that works across multiple vendors
- Keep a human in the loop for high-impact actions

## Document map

| Section | Contents |
|---|---|
| Context | Problem, stakeholders, scope, and shared terminology |
| Requirements | Capabilities, constraints, use cases, and acceptance criteria |
| Architecture | Service boundaries, data flow, deployment, and resilience |
| Domain model | Event, alert, case, observable, evidence, and action |
| Playbooks | Lifecycle format, automation tiers, approval, and catalog |
| Integrations | Wazuh, SIEM, EDR, firewall, identity, TI, ticketing, DFIR |
| Security | Threat model, RBAC, secrets, audit, safe response, and AI safety |
| Operations | Metrics, observability, backup, maintenance, and troubleshooting |
| Testing | Connector, workflow, replay, tabletop, and failure injection |
| Governance | Policy, privacy, retention, custody, and compliance mapping |
| Roadmap | MVP, milestones, and architecture decisions |

## Design principles

1. **Evidence before action.** Every response must point to data that can be verified.
2. **Least privilege.** Each connector gets only the permissions one action needs.
3. **Human judgment for high impact.** Block, isolate, disable, and delete all require an approval gate.
4. **Idempotent by default.** A duplicate event must not trigger the same action again in a way that causes damage.
5. **Fail closed.** If data is incomplete or a policy can't be evaluated, stop and escalate.
6. **Business-aware response.** Check asset criticality and service impact before containment.
7. **Version everything.** Schemas, connectors, playbooks, policies, and evidence manifests are all versioned.
8. **Provider-neutral core.** Vendor adapters live at the edge of the system, and vendor-specific fields stay out of the domain model.

## Main references

- [NIST SP 800-61 Rev. 3](https://csrc.nist.gov/pubs/sp/800/61/r3/final)
- [CISA Cybersecurity Incident and Vulnerability Response Playbooks](https://www.cisa.gov/sites/default/files/2024-08/Federal_Government_Cybersecurity_Incident_and_Vulnerability_Response_Playbooks_508C.pdf)
- [OASIS CACAO Security Playbooks 2.0](https://docs.oasis-open.org/cacao/security-playbooks/v2.0/security-playbooks-v2.0.html)
- [OCSF](https://ocsf.io/)
- [OASIS TAXII 2.1](https://docs.oasis-open.org/cti/taxii/v2.1/taxii-v2.1.html)
- [Sigma Correlation Specification](https://sigmahq.io/sigma-specification/specification/sigma-correlation-rules-specification.html)

## Where to start

Read these in order:

1. [Mission and scope](docs/00-context/mission-and-scope.md)
2. [System overview](docs/02-architecture/system-overview.md)
3. [Domain model](docs/03-domain-model/event-model.md)
4. [Playbook standard](docs/04-playbooks/playbook-standard.md)
5. [MVP definition](docs/10-roadmap/mvp-definition.md)

To see how complete the design is, check the [Completeness audit](docs/10-roadmap/completeness-audit.md), [Decision register](docs/10-roadmap/decision-register.md), [Source catalog](docs/00-context/source-catalog.md), [Source-to-design traceability](docs/09-governance/source-to-design-traceability.md), and [Definition of complete](docs/10-roadmap/definition-of-complete.md).

For implementation detail, continue with:

- [Software development blueprint](docs/02-architecture/software-development-blueprint.md)
- [API design](docs/02-architecture/api-design.md)
- [Data architecture](docs/02-architecture/data-architecture.md)
- [Lab deployment profile](docs/02-architecture/lab-deployment-profile.md)
- [Lab integration profile](docs/05-integrations/lab-integration-profile.md)
- [Execution semantics](docs/02-architecture/execution-semantics.md)
- [Response decision matrix](docs/04-playbooks/response-decision-matrix.md)
- [Data-source coverage](docs/01-requirements/data-source-coverage.md)
- [Provider capability matrix](docs/05-integrations/provider-capability-matrix.md)

The machine-readable API contract is [api/openapi.yaml](api/openapi.yaml).

Playbooks based on real sources are in [source-derived playbooks](docs/04-playbooks/source-derived-playbooks.md). They are checked with `validation/validate-source-traceability.sh`.
