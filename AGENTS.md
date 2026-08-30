# Repository Guidelines

## Project Scope

`soar-platform` is the design and contract baseline for a lab-oriented Security Orchestration, Automation and Response platform. It is documentation-first: do not assume production access, live response authority, or completed application code.

## Project Structure

- `docs/` — requirements, architecture, domain model, integrations, security, operations, testing, governance, and roadmap.
- `api/openapi.yaml` — machine-readable HTTP API contract.
- `schemas/` — JSON Schemas for events, cases, observables, actions, playbooks, policies, and evidence.
- `examples/` — representative payloads used by validation and documentation.
- `playbooks/` — response workflows, templates, and source-derived playbooks.
- `adr/` — architecture decision records.
- `validation/` — repository validation scripts.
- `.agents/skills/` — project-scoped agent skills; inspect the relevant `SKILL.md` before specialized work.
- `.claude/skills/` — Claude Code discovery bridge to the canonical `.agents/skills/` set; do not duplicate skills here.

## Development Workflow

Read `README.md` and the relevant design documents before changing contracts or playbooks. Preserve the lab/dry-run boundary and human approval gates for containment, isolation, blocking, disabling, deletion, or other high-impact actions. Keep provider-specific behavior inside connector or integration boundaries.

Run validation from this directory:

```bash
bash validation/validate-json.sh
bash validation/validate-playbooks.sh
bash validation/validate-diagrams.sh
bash validation/validate-links.sh
bash validation/validate-source-traceability.sh
bash -n validation/*.sh
```

Run the smallest relevant check first, then the full set before handoff. If adding implementation later, add focused tests with the owning component and document the command here.

## Style & Naming

Use concise Markdown with descriptive headings, stable terminology, and repository-relative links. Keep YAML and JSON valid, deterministic, and explicitly versioned where the schema requires it. Use lowercase kebab-case for new playbook filenames and `*.schema.json` for new schemas. Shell scripts use `#!/usr/bin/env bash`, `set -euo pipefail`, and quoted paths.

## Security & Evidence

Never commit credentials, live tokens, incident artifacts, or real customer data. Keep evidence immutable and write derived material only to approved work/export locations. Treat connector actions as least-privilege and idempotent; record approval, actor, timestamps, inputs, outputs, and rollback information for every response action.

## Changes & Reviews

Use concise imperative commit subjects scoped to this project, such as `soar: add phishing playbook`. Pull requests should explain affected contracts, source references, safety impact, validation commands, and any unresolved assumptions. Do not install packages, start services, or connect to live security systems without explicit authorization.
