# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@AGENTS.md

## Claude-specific workflow

- Treat this repository as a lab design baseline until the user explicitly authorizes implementation or live integration.
- Before specialized work, use the matching skill discovered under `.claude/skills/` and follow its `SKILL.md` instructions. This directory is a symlink to the canonical `.agents/skills/` set; do not install duplicate copies.
- For frontend work, use the project-scoped design skills; for Go/backend/API work, use the matching Go, backend, and API skills.
- Before editing, identify the authoritative document, schema, or ADR affected by the change.
- Before claiming completion, run the relevant validation scripts and report exact results.
- Ask before destructive commands, broad installs, production actions, or changes outside `soar-platform/`.

## Repository nature

There is no application code yet — `adr/0001-documentation-first.md` and `adr/0005-selected-software-stack.md` mark implementation as not started. Every change here is to a document, schema, example, playbook, or ADR, and "correctness" means internal consistency across those artifacts, not passing a build. Run `bash -n validation/*.sh` plus the smallest relevant validator from `AGENTS.md`'s Development Workflow before handoff; there is no separate build/lint/test toolchain to invoke.

## Cross-file architecture

**Numbered docs = lifecycle order.** `docs/00-context/` through `docs/10-roadmap/` is a deliberate sequence (problem → requirements → architecture → domain model → playbooks → integrations → security → operations → testing → governance → roadmap). When a change touches one stage, check whether the adjacent stage needs a matching update (e.g., a new domain field usually implies a schema change *and* an entry in the relevant integration/security doc).

**The domain model is graph-shaped, not a flat list.** `schemas/*.schema.json` cross-reference each other with `$ref` (e.g., `event.schema.json` embeds `observable.schema.json`; `action.schema.json` and `evidence.schema.json` connect back to cases). `examples/*.json` are fixtures asserted against these schemas by `validation/validate-json.sh`, including nested-object checks (e.g., observables embedded in an event or alert example) — editing a schema's `required` fields without updating the matching example breaks validation.

**Playbooks are a structural contract, not free-form YAML.** `validation/validate-playbooks.sh` enforces required top-level keys (`id, name, version, owner, type, status, references, trigger, scope, preconditions, steps, safety, evidence, metrics, tests`), a closed `status` and step-`kind` vocabulary, and — critically — that `safety.default_mode` is `dry_run` and `safety.stop_conditions` is non-empty. This is the mechanical enforcement of the human-in-the-loop / lab-only boundary described in `adr/0003-human-in-the-loop.md`; a playbook cannot be merged in a state that defaults to live action. New playbooks should start from `playbooks/templates/playbook-template.yaml` and are indexed in `playbooks/README.md` and `docs/04-playbooks/playbook-catalog.md`.

**Source traceability links three places.** `docs/00-context/source-catalog.md` registers `SRC-*` IDs with canonical URLs; `docs/09-governance/source-to-design-traceability.md` must reference every one of them; and the source-derived playbooks (`cisa-major-incident-coordination.yaml`, `cisa-vulnerability-response.yaml`, `nist-recovery.yaml`, `dfir-evidence-preservation.yaml`) must each carry `source_basis:`/`adaptation:` fields whose `source_id:` values resolve back to the catalog. `validation/validate-source-traceability.sh` checks all three links together — adding a new source-derived playbook or citing a new source ID requires updating the catalog and traceability matrix in the same change.

**Markdown links are validated repo-wide** by `validation/validate-links.sh`, except under `.agents/skills/` (vendored skill content with intentionally external-relative links) and any `node_modules/`/`.next/` (installed/build artifacts under `services/api` or `web`, never part of this repo's own content) — keep repository-relative links accurate when moving or renaming docs.

**The selected (not-yet-built) stack** is Next.js/React/TypeScript (UI), Go/Gin (API and workers), Temporal (durable workflows), NATS JetStream (event transport), PostgreSQL via pgx/sqlc/Atlas (source of truth), OpenSearch (derived search), Keycloak (OIDC), Vault (secrets), and OpenTelemetry/Prometheus/Grafana/Loki (observability) — see `adr/0005-selected-software-stack.md`. Design docs that reference implementation details (e.g., `docs/02-architecture/software-development-blueprint.md`, `api/openapi.yaml`) should stay consistent with this decision unless a new ADR supersedes it.
