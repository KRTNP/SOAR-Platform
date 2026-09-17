# Repository Guidelines

## Project Scope

`soar-platform` is the design and contract baseline for a lab-oriented Security Orchestration, Automation and Response platform. It is documentation-first: the contract is designed before the code. Implementation has started (`services/api/`, `web/`), but do not assume production access or live response authority.

## Project Structure

- `docs/` — requirements, architecture, domain model, integrations, security, operations, testing, governance, roadmap.
- `api/openapi.yaml` — machine-readable HTTP API contract.
- `schemas/` — JSON Schemas for events, cases, observables, actions, playbooks, policies, evidence.
- `examples/` — payloads used by validation and documentation.
- `playbooks/` — response workflows, templates, source-derived playbooks.
- `adr/` — architecture decision records.
- `validation/` — repository validation scripts.
- `services/api/` — Go/Gin API. `web/` — Next.js operator console.
- `.agents/skills/` — project-scoped agent skills; read the relevant `SKILL.md` before specialized work. `.claude/skills/` is a discovery symlink to it; do not duplicate skills there.

## Development Workflow

Read `README.md` and the relevant design documents before changing contracts or playbooks. Preserve the lab/dry-run boundary and human approval gates for containment, isolation, blocking, disabling, deletion, or other high-impact actions. Keep provider-specific behavior inside connector or integration boundaries.

Design artifacts (docs, schemas, examples, playbooks), from this directory:

```bash
make validate            # validate-json, -playbooks, -diagrams, -links, -source-traceability
bash -n validation/*.sh  # the validators' own shell syntax
```

They need `python3` with PyYAML.

Go API:

```bash
cd services/api
gofmt -l .                        # must print nothing
go vet -tags=integration ./...
golangci-lint run ./...           # config: .golangci.yml at the repo root
make -C ../.. test                # unit tests
make -C ../.. test-integration    # tagged; PostgreSQL via testcontainers, needs a Docker/Podman socket
```

Operator console:

```bash
cd web
npx tsc --noEmit
npm run lint    # next/core-web-vitals + the full 33-rule jsx-a11y set
npm run build
```

Run the smallest relevant check first, then everything covering what you changed, and report exact results. `.github/workflows/ci.yml` runs all of the above on every push and pull request. Add focused tests with the owning component and document any new command here.

## Style & Naming

Concise Markdown with descriptive headings, stable terminology, repository-relative links. Valid, deterministic YAML and JSON, explicitly versioned where the schema requires it. Lowercase kebab-case for playbook filenames, `*.schema.json` for schemas. Shell scripts use `#!/usr/bin/env bash`, `set -euo pipefail`, and quoted paths.

## Security & Evidence

Never commit credentials, live tokens, incident artifacts, or real customer data. Keep evidence immutable and write derived material only to approved work/export locations. Treat connector actions as least-privilege and idempotent; record approval, actor, timestamps, inputs, outputs, and rollback information for every response action.

## Changes & Reviews

Use concise imperative commit subjects scoped to this project, such as `soar: add phishing playbook`. Pull requests should explain affected contracts, source references, safety impact, validation commands, and unresolved assumptions. Do not install packages, start services, or connect to live security systems without explicit authorization.
