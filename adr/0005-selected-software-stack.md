# ADR-0005: Select the lab software stack

## Status

Accepted — design only; implementation not started.

## Decision

Use Next.js/React/TypeScript for the web UI; Go/Gin for the core API and workers;
Temporal for durable workflows; NATS JetStream for event transport; PostgreSQL with
pgx/sqlc and Atlas for authoritative state; OpenSearch for derived search; Keycloak
for OIDC; Vault for secrets; local filesystem for lab evidence; and OpenTelemetry,
Prometheus, Grafana and Loki for observability. Use Go testing, Testcontainers,
Playwright and OpenAPI contract tests for verification.

## Rationale

This balances a strong Go runtime for API/worker/connector services with an ecosystem
that can call specialist DFIR tools. Durable workflow state is required for approval
waits, timers, retries and recovery after worker failure. PostgreSQL remains the source
of truth while search and observability are rebuildable projections.

## Consequences

- The lab has more components than a toy monolith and needs resource planning.
- Connector and workflow boundaries are testable independently.
- Versions must be pinned in the future and upgraded through contract/replay tests.
- DFIR parsers remain specialist tools; Go orchestrates their authorized execution.
