# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@AGENTS.md

## Claude-specific workflow

- Implementation is authorized and underway; **live integration is not**. Keep the lab/dry-run boundary, `safety.default_mode: dry_run`, and the human approval gates for containment, isolation, blocking, disabling, and deletion. Do not point this code at a production or third-party security system, and do not start services, without the user saying so.
- Before specialized work, use the matching skill discovered under `.claude/skills/` and follow its `SKILL.md` instructions. This directory is a symlink to the canonical `.agents/skills/` set; do not install duplicate copies.
- For frontend work, use the project-scoped design skills; for Go/backend/API work, use the matching Go, backend, and API skills.
- Before editing, identify the authoritative document, schema, or ADR affected by the change.
- Before claiming completion, run the relevant validation scripts and report exact results.
- Ask before destructive commands, broad installs, production actions, or changes outside `soar-platform/`.

## Repository nature

Implementation has started. The tree is now mixed: design artifacts (`docs/`, `adr/`, `schemas/`, `examples/`, `playbooks/`) *and* real application code. `adr/0001-documentation-first.md` still governs the **order** — design the contract before writing the code — but it no longer describes the contents, and "correctness" is now both internal consistency across documents *and* a passing build.

- **`services/api/`** — Go module `soar-platform/services/api` (`go 1.25`, Gin), ~70 source files across ~30 packages with ~24 `*_test.go`. Unit tests: `make test`. The `integration/` package sits behind the `integration` build tag and starts PostgreSQL with `testcontainers-go`, so it needs a reachable Docker/Podman socket: `make test-integration`. Because of the tag, a plain `go test ./...` runs unit tests only.
- **`web/`** — Next.js 16 (App Router, Turbopack) / React 19 / Tailwind v4 operator console. From `web/`: `npx tsc --noEmit`, `npm run lint`, `npm run build`. Read `web/AGENTS.md` first — this Next.js version's APIs differ from training data, and the guides under `web/node_modules/next/dist/docs/` are authoritative over recollection.
- **Docs, schemas, playbooks** — still validated by content rather than compilation: `make validate` runs all five `validation/*.sh`, plus `bash -n validation/*.sh` for the scripts themselves.
- **`deploy/compose/docker-compose.core.yml`** — the lab stack, driven by `make db-up` / `make migrate` (Atlas). Starting services still requires the user's authorization per `AGENTS.md`.
- **CI** — `.github/workflows/ci.yml` runs the validators, `golangci-lint`, Go unit tests, the tagged integration tests, and the `web` typecheck/lint/build on every push and PR. Run the matching command locally before handoff rather than discovering it there.

Before claiming completion, run the checks that match what actually changed and report exact results.

## Verifying the console in a browser

`.mcp.json` configures the `chrome-devtools` MCP server, because until it existed every UI claim in this repo was backed by typecheck, lint and source reading rather than by a rendered page — which is how a focus trap that steals focus from the input it was meant to serve survived several review rounds. Use it to *look* before reporting a UI change as done.

It needs the app running (`make api`, `make web`), which still requires the user's authorization.

Four settings in that file are deliberate and should not be dropped:

- `--isolated` — a throwaway user-data-dir, so an agent-driven browser never touches the operator's real Chrome profile, cookies, or sessions.
- `--allowedUrlPattern` limited to `localhost`/`127.0.0.1` — the browser cannot navigate off the lab, so neither a mistake nor a malicious string rendered in a case can send lab data outward. Requires Chrome 149+.
- `--performanceCrux false` — otherwise URLs from performance traces are sent to Google's CrUX API.
- `--usageStatistics false` — opt out of telemetry for a security-lab repository.

`CHROME_PATH` overrides the browser binary; the default points at a Chrome for Testing build already on this machine, since there is no system-wide Chrome (only a flatpak, which Puppeteer cannot launch directly).

## Cross-file architecture

**Numbered docs = lifecycle order.** `docs/00-context/` through `docs/10-roadmap/` is a deliberate sequence (problem → requirements → architecture → domain model → playbooks → integrations → security → operations → testing → governance → roadmap). When a change touches one stage, check whether the adjacent stage needs a matching update (e.g., a new domain field usually implies a schema change *and* an entry in the relevant integration/security doc).

**The domain model is graph-shaped, not a flat list.** `schemas/*.schema.json` cross-reference each other with `$ref` (e.g., `event.schema.json` embeds `observable.schema.json`; `action.schema.json` and `evidence.schema.json` connect back to cases). `examples/*.json` are fixtures asserted against these schemas by `validation/validate-json.sh`, including nested-object checks (e.g., observables embedded in an event or alert example) — editing a schema's `required` fields without updating the matching example breaks validation.

**Playbooks are a structural contract, not free-form YAML.** `validation/validate-playbooks.sh` enforces required top-level keys (`id, name, version, owner, type, status, references, trigger, scope, preconditions, steps, safety, evidence, metrics, tests`), a closed `status` and step-`kind` vocabulary, and — critically — that `safety.default_mode` is `dry_run` and `safety.stop_conditions` is non-empty. This is the mechanical enforcement of the human-in-the-loop / lab-only boundary described in `adr/0003-human-in-the-loop.md`; a playbook cannot be merged in a state that defaults to live action. New playbooks should start from `playbooks/templates/playbook-template.yaml` and are indexed in `playbooks/README.md` and `docs/04-playbooks/playbook-catalog.md`.

**Source traceability links three places.** `docs/00-context/source-catalog.md` registers `SRC-*` IDs with canonical URLs; `docs/09-governance/source-to-design-traceability.md` must reference every one of them; and the source-derived playbooks (`cisa-major-incident-coordination.yaml`, `cisa-vulnerability-response.yaml`, `nist-recovery.yaml`, `dfir-evidence-preservation.yaml`) must each carry `source_basis:`/`adaptation:` fields whose `source_id:` values resolve back to the catalog. `validation/validate-source-traceability.sh` checks all three links together — adding a new source-derived playbook or citing a new source ID requires updating the catalog and traceability matrix in the same change.

**Markdown links are validated repo-wide** by `validation/validate-links.sh`, except under `.agents/skills/` (vendored skill content with intentionally external-relative links) and any `node_modules/`/`.next/` (installed/build artifacts under `services/api` or `web`, never part of this repo's own content) — keep repository-relative links accurate when moving or renaming docs.

**The selected stack, and how much of it is actually wired.** `adr/0005-selected-software-stack.md` selects Next.js/React/TypeScript (UI), Go/Gin (API and workers), Temporal (durable workflows), NATS JetStream (event transport), PostgreSQL via pgx/sqlc/Atlas (source of truth), OpenSearch (derived search), Keycloak (OIDC), Vault (secrets), and OpenTelemetry/Prometheus/Grafana/Loki (observability). Present in `services/api/go.mod` today: Gin, pgx, Vault, Prometheus, OpenTelemetry, and `golang-jwt` for the Keycloak path (`adr/0024`); OpenSearch is reached over plain HTTP from `searchindex/` with no client library. **Temporal and NATS JetStream have no dependency yet** — treat durable workflow execution and event transport as designed but unbuilt, and do not write code that assumes either is running. Design docs that reference implementation details (e.g., `docs/02-architecture/software-development-blueprint.md`, `api/openapi.yaml`) should stay consistent with `adr/0005` unless a new ADR supersedes it.
