# Definition of complete for the SOAR design baseline

“Complete” is split into two claims so that a well-researched design is not confused
with a deployed production service.

## Design baseline complete

The baseline is complete when all of the following are true:

- source catalog records authority, version/date, exact reviewed location, extracted practice, limitation, and affected design artifact;
- every registered source is referenced by the source-to-design traceability matrix;
- every source-derived playbook records source IDs, adaptation notes, safe default, stop conditions, evidence, metrics, and tests;
- canonical schemas, execution semantics, approval model, evidence/custody model, connector contract, and test matrix exist, with the playbook schema aligned to every YAML contract;
- validation checks cover JSON schemas/examples, YAML playbook contracts, diagrams, local links, and source traceability;
- unresolved organization-specific decisions are listed instead of silently assumed.

## Production complete

Production readiness additionally requires evidence that cannot be obtained from public
research alone:

- approved written scope, authority, legal/jurisdiction mapping, RACI, on-call and emergency procedure;
- real provider API contracts and version-pinned sandbox connectors for the first deployment;
- tenant/asset/user mappings, business criticality and operational safety thresholds;
- SSO/MFA, secret manager, least-privilege credentials, network controls, backup/restore drill and measured SLOs;
- replay, negative, security, resilience and tabletop evidence from the target environment;
- signed change approval and service-owner sign-off for every write-capable playbook.

The project therefore claims a complete, source-traceable design baseline after the
validators pass, but must not claim production readiness until the second list has
operational evidence.
