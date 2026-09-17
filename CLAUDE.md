# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@AGENTS.md

## Claude-specific workflow

- Implementation is authorized; **live integration is not**. Keep the lab/dry-run boundary, `safety.default_mode: dry_run`, and the human approval gates for containment, isolation, blocking, disabling, and deletion. Do not point this code at a production or third-party security system, and do not start services, without the user saying so.
- Before specialized work, use the matching skill under `.claude/skills/` (a symlink to the canonical `.agents/skills/`; do not duplicate skills there) and follow its `SKILL.md`.
- For frontend work the design authority is the `impeccable` plugin **plus** these UX skills: `ux-usability-foundations`, `accessibility-inclusive-design`, `ui-visual-composition`, `ux-writing-content-design`, `information-architecture-navigation`, `web-design-guidelines`. Reach for them by name — `impeccable` crowds them out otherwise, and past rounds surfaced accessibility findings late as a result.
- For Go/backend/API work, use the matching Go, backend, and API skills.
- Before editing, identify the authoritative document, schema, or ADR affected.
- Before claiming completion, run the checks that match what changed and report exact results.
- Ask before destructive commands, broad installs, production actions, or changes outside `soar-platform/`.

## Repository nature

The tree is mixed: design artifacts (`docs/`, `adr/`, `schemas/`, `examples/`, `playbooks/`) *and* real application code. `adr/0001-documentation-first.md` governs the **order** — design the contract before writing the code — not the contents. "Correctness" is now both internal consistency across documents *and* a passing build.

- **`services/api/`** — Go module `soar-platform/services/api` (`go 1.25`, Gin), ~70 source files, ~24 `*_test.go`. `integration/` sits behind the `integration` build tag and starts PostgreSQL via `testcontainers-go`, so plain `go test ./...` runs unit tests only.
- **`web/`** — Next.js 16 (App Router, Turbopack) / React 19 / Tailwind v4 operator console. Read `web/AGENTS.md` first: this Next.js version's APIs differ from training data, and `web/node_modules/next/dist/docs/` is authoritative over recollection.
- **Docs, schemas, playbooks** — validated by content, not compilation: `make validate`.
- **`deploy/compose/docker-compose.core.yml`** — the lab stack (`make db-up`, `make migrate` via Atlas). Starting it requires the user's authorization.
- **CI** — `.github/workflows/ci.yml` runs everything in `AGENTS.md` on every push and PR.

## Verifying the console in a browser

`.mcp.json` configures the `chrome-devtools` MCP server. Use it to *look* at a page before reporting a UI change as done; typecheck, lint, and source reading have missed defects that one screenshot would have caught. It needs `make api` and `make web` running, which requires the user's authorization.

Four settings in that file are deliberate (JSON cannot carry comments) and should not be dropped:

- `--isolated` — throwaway user-data-dir, so the agent browser never touches the operator's real Chrome profile or sessions.
- `--allowedUrlPattern` limited to `localhost`/`127.0.0.1` — the browser cannot navigate off the lab, so nothing rendered in a case can send lab data outward. Requires Chrome 149+.
- `--performanceCrux false` — otherwise trace URLs go to Google's CrUX API.
- `--usageStatistics false` — no telemetry from a security-lab repo.

`CHROME_PATH` overrides the binary; the default points at a Chrome for Testing build already on this machine (there is no system Chrome, only a flatpak Puppeteer cannot launch).

## Cross-file architecture

**Numbered docs = lifecycle order.** `docs/00-context/` → `docs/10-roadmap/` runs problem → requirements → architecture → domain model → playbooks → integrations → security → operations → testing → governance → roadmap. A change in one stage usually needs a matching update in the adjacent one (a new domain field implies a schema change *and* an integration/security doc entry).

**The domain model is graph-shaped.** `schemas/*.schema.json` cross-reference via `$ref` (`event.schema.json` embeds `observable.schema.json`; `action` and `evidence` connect back to cases). `examples/*.json` are fixtures asserted against those schemas by `validation/validate-json.sh`, including nested objects — changing a schema's `required` without updating the matching example breaks validation.

**Playbooks are a structural contract.** `validation/validate-playbooks.sh` enforces the required top-level keys (`id, name, version, owner, type, status, references, trigger, scope, preconditions, steps, safety, evidence, metrics, tests`), closed `status` and step-`kind` vocabularies, and — critically — `safety.default_mode: dry_run` with a non-empty `safety.stop_conditions`. That is the mechanical enforcement of `adr/0003-human-in-the-loop.md`: a playbook cannot merge defaulting to live action. Start from `playbooks/templates/playbook-template.yaml`; index new ones in `playbooks/README.md` and `docs/04-playbooks/playbook-catalog.md`.

**Source traceability links three places**, all checked together by `validation/validate-source-traceability.sh`: `docs/00-context/source-catalog.md` registers `SRC-*` IDs; `docs/09-governance/source-to-design-traceability.md` must reference every one; and the source-derived playbooks (`cisa-major-incident-coordination`, `cisa-vulnerability-response`, `nist-recovery`, `dfir-evidence-preservation`) carry `source_basis:`/`adaptation:` whose `source_id:` resolves back to the catalog. Adding a source-derived playbook or citing a new source ID means updating all three in the same change.

**Markdown links are validated repo-wide** by `validation/validate-links.sh`, except `.agents/skills/` (vendored, intentionally external-relative) and `node_modules/`/`.next/`.

**The stack is selected but only partly wired.** `adr/0005-selected-software-stack.md` selects Next.js/React/TypeScript, Go/Gin, Temporal, NATS JetStream, PostgreSQL (pgx/sqlc/Atlas), OpenSearch, Keycloak, Vault, and OpenTelemetry/Prometheus/Grafana/Loki. Actually in `services/api/go.mod`: Gin, pgx, Vault, Prometheus, OpenTelemetry, `golang-jwt` (Keycloak path, `adr/0024`); OpenSearch is reached over plain HTTP from `searchindex/` with no client library. **Temporal and NATS JetStream have no dependency at all** — durable workflow execution and event transport are designed but unbuilt, so do not write code assuming either is running.
