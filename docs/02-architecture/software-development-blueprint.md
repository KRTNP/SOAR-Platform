# Software development blueprint

This is the authoritative implementation design for the KRTN SOAR lab. It records
the selected stack and boundaries without creating source code or installing services.

## Target outcome

Build a single-tenant, on-premise SOAR lab that receives real Wazuh alerts, correlates
SSH brute-force activity, creates a case, enriches context, requests one-person
approval, blocks an IP on an OPNsense lab firewall for ten minutes, verifies the
post-condition, and removes the block with bounded retry.

The same platform must expose the broader domain model for DFIR, evidence, playbooks,
approvals, connectors and recovery. Only the first vertical scenario is write-enabled;
other response actions remain dry-run until separately authorized.

## Selected stack

| Layer | Selection | Responsibility | Boundary |
|---|---|---|---|
| Web | Next.js + React + TypeScript | SOC dashboard, case/timeline, execution, approvals and evidence views | Calls REST API; receives SSE; never calls providers directly |
| UI system | Tailwind CSS + shadcn/ui + Radix UI | Consistent accessible components and visual states | No business policy in components |
| Client state | TanStack Query + Zustand | Server cache/query state and local UI state | Server truth remains in API |
| Core API | Go + Gin | Authentication context, REST resources, policy evaluation, command submission and SSE | No long-running provider work inside request handlers |
| API contract | OpenAPI + `oapi-codegen` + Orval | Generate Go interfaces/types and TypeScript client/hooks | OpenAPI is the contract source of truth |
| Workflow | Temporal + Go SDK | Durable playbook execution, timers, signals, retries and approval waits | Activities call connectors; workflows remain deterministic |
| Event bus | NATS JetStream | Durable alert/event transport, consumer groups and replay | Ingestion is decoupled from workers |
| Domain store | PostgreSQL + pgx + sqlc | Cases, alerts, policies, approvals, executions, audit metadata and outbox | Transactional source of truth |
| Migration | Atlas | Schema versioning, diff, lint and drift detection | Migration review is mandatory |
| Search | OpenSearch | Derived full-text, time-range and event/alert search | Rebuildable projection, never sole source of truth |
| Index sync | Transactional outbox + NATS + indexer worker | Reliable PostgreSQL-to-OpenSearch projection | Replays are idempotent |
| Identity | Keycloak + OIDC/OAuth2 | SSO, MFA, user/group identity and token claims | SOAR does not store user passwords |
| Authorization | RBAC + resource scope | Role plus tenant/team/case/asset/connector boundaries | Default deny; policy decision is audited |
| Secrets | HashiCorp Vault | Connector tokens, credentials, rotation and secret access audit | Secrets never enter playbook payloads or logs |
| Evidence | Local filesystem for lab | Original, derived, export and manifest separation | Original evidence is read-only after ingest |
| Observability | OpenTelemetry, Prometheus, Grafana, Loki | Traces, metrics, dashboards, logs and alerts | Correlation/request IDs cross all components |
| Packaging | Docker Compose | Repeatable lab service topology | KVM host-only network; no production target |
| Testing | Go testing, Testcontainers, Playwright and OpenAPI contract tests | Unit, integration, UI and API compatibility verification | Tests run against synthetic data or authorized lab targets |

## Service boundaries

| Component | Owns | Must not own |
|---|---|---|
| API | Validation, authorization, resource lifecycle and command submission | Provider credentials, long-running action execution |
| Ingestion | Webhook/API authentication, replay check, raw redaction, normalization and deduplication | Direct containment actions |
| Workflow worker | Temporal workflow/activity execution and state transitions | UI rendering or unrestricted provider access |
| Connector worker | One provider capability set and error mapping | Cross-provider policy decisions |
| Policy engine | Scope, risk, approval and deny/escalate decisions | Provider-specific request construction |
| Evidence service | Manifest, hash, custody and path boundary | Editing original evidence |
| Indexer | Projection of approved domain events into OpenSearch | Authoritative case state |
| Notification service | Approved analyst/communication notifications | Deciding whether a response is allowed |

## Non-negotiable invariants

- Every external command has a timeout, bounded retry, correlation ID, actor and policy version.
- Every write-capable action has an explicit target, risk class, approval state, post-condition and rollback/compensation definition.
- Every playbook execution is pinned to playbook version, connector version and policy version.
- A provider outage creates visible partial failure; it never silently becomes success.
- Untrusted input is data, not workflow logic or executable command text.
- Wazuh, Kali, victim VM and OPNsense are lab-scoped and cannot reach production assets.
