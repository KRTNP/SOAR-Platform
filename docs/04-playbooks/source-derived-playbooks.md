# Source-derived playbook pack

The YAML files in this pack are paraphrased operational designs derived from public
guidance. They are not copied government procedures and are not production authority.
Each one declares `source_basis`, adaptation notes, safety gates, evidence outputs, and
test scenarios. All begin in `dry_run` and require local approval before write actions.

| Playbook | Source basis | Operational purpose |
|---|---|---|
| `cisa-major-incident-coordination.yaml` | SRC-NIST-61R3, SRC-CISA-IRVULN, SRC-CISA-TTX | Coordinate a major incident, preserve evidence, establish roles, communicate, and close with lessons learned |
| `cisa-vulnerability-response.yaml` | SRC-CISA-IRVULN, SRC-NIST-61R3 | Track vulnerability intake through verification and exception handling |
| `nist-recovery.yaml` | SRC-NIST-61R3, SRC-NIST-800184, SRC-CISA-RANSOM | Restore prioritized services from known-good sources with validation and monitoring |
| `dfir-evidence-preservation.yaml` | SRC-LOCAL-DFIR, SRC-NIST-61R3, SRC-CISA-RANSOM | Start a read-only evidence and custody workflow before containment or recovery |

## Adaptation contract

Before a playbook is promoted beyond lab testing, the owner must replace placeholders
with an approved tenant, asset scope, contacts, provider operations, legal rules,
rollback plan, sandbox fixture, and change authority. A public source supplies the
pattern; it does not authorize actions against a real environment.

