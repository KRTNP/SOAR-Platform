# ADR-0020: Prometheus metrics, Loki logs, and Grafana dashboards

## Status

Accepted — implemented and proven end-to-end against the real running lab
Compose stack.

## Decision

Implement the first slice of `docs/07-operations/observability.md`'s already-
decided target (OpenTelemetry SDK metrics exported as Prometheus exposition
format; Loki for logs; Grafana for dashboards): metrics and logs, not tracing.

### 1. Metrics: `services/api/telemetry` + `GET /metrics`

A new package wraps a private `prometheus.Registry` (not
`prometheus.DefaultRegisterer`) with the metrics `docs/07-operations/observability.md`
and `docs/07-operations/metrics-and-kpis.md` already name as worth capturing:
HTTP request count/duration by route template, ingestion outcomes (accepted/
duplicate/rejected/error), case outcomes (created vs correlated), the action
funnel (requested → approved/rejected → executed → released/escalated),
connector call outcomes, and scheduler/wazuhpoll cycle durations. `GET
/metrics` (plain, unauthenticated like `/healthz`/`/readyz` — this is a lab
with no auth boundary yet) serves it via `promhttp`.

Every `telemetry.Recorder` method is nil-receiver-safe, so `events.Handler`,
`cases.Handler`, `actions.Handler`, `scheduler.Sweep`/`Run`, and
`wazuhpoll.Run` all take a `*telemetry.Recorder` parameter that existing unit/
integration tests pass as `nil` without needing to construct a real one —
the same "optional collaborator, nil-safe" shape this codebase already uses
elsewhere (e.g. `NetworkConnector.Preview`).

Business-metric recording lives at the HTTP-handler/orchestration layer
(`events.Handler.Ingest`, `actions.Handler.Execute/Release/Approve/Reject`,
`cases.Handler`'s manual `request_block` operation, `scheduler.Sweep`/
`escalate`), not inside the pure domain packages (`actions.go`, `cases.go`) —
keeping instrumentation out of business logic the same way audit-writing
already stays a caller concern rather than something `actions.Execute` does
itself.

### 2. Logs: Loki + Promtail via a shared file, not the runtime socket

`services/api` already logs structured JSON to stdout (`slog.NewJSONHandler`)
with a request-id (see `services/api/httpx/middleware.go`). The first design
tried here was Promtail's `docker_sd_configs` — no application change at
all, just point Promtail at the container runtime's Docker-API-compatible
socket. That does not work on this lab's rootless-podman + SELinux-enforcing
host: neither plain uid matching nor `--userns=keep-id` let Promtail open
the mounted `podman.sock` ("permission denied" persisted under both,
verified directly), and relabeling that socket with `:Z` was ruled out as
unsafe — it is a live, shared resource every other podman command on the
host also depends on.

What's actually implemented instead: `services/api/main.go`'s new
`newLogger()` writes to `LOG_FILE_PATH` (in addition to stdout, when that
env var is set — unset is byte-for-byte today's behavior) onto a named
`soar_api_logs` volume shared with the `promtail` container, which tails it
as a plain file (`static_configs` with `__path__`, no discovery, no socket).
Verified end-to-end: a live request's log line was queried back out of Loki
directly (`{job="soar-api"}`) and through Grafana's own datasource proxy.

### 2b. Networking: prometheus/grafana also need this lab's host-network override

`docker-compose.override.yml` (gitignored, real-lab credentials) already
host-networks `api`/`web` so they can reach the real Wazuh indexer and the
OPNsense lab VM (see that file's own header comment — pasta's gateway
couldn't forward to rootlessport-published ports). Once `api` leaves the
compose bridge network, a bridge-networked `prometheus` can no longer
resolve it by service name (`api:8080`) or reach it via
`host.containers.internal` (the same pasta-gateway limitation, confirmed by
testing it directly: connection refused). The override therefore also
host-networks `prometheus` and `grafana`, pointing them at `localhost:8080`/
`localhost:9090`/`localhost:3100` via a local-only
`prometheus/prometheus.local.yml` and `grafana/provisioning.local/`
(gitignored, mirroring the committed defaults with `localhost` targets) —
and moves Grafana off port 3000 (`GF_SERVER_HTTP_PORT: "3002"`, since `web`
already occupies the host's port 3000 once host-networked). `loki` and
`promtail` needed no such override — `loki`'s port is already
host-published regardless of network mode, and `promtail` reaches it via a
volume mount, not the network. The **committed** `prometheus.yml` and
`docker-compose.core.yml` still assume normal bridge networking
(`api:8080`) — the correct, portable default for anyone running this stack
without the host-networking override.

Also discovered along the way (unrelated to networking, but blocking the
same verification pass): rootless podman under an SELinux-enforcing host
denies a bind-mounted config file to a container process unless it carries
the `:Z` relabel option, even when the file itself is world-readable and
every parent directory is traversable — `prometheus.yml`, the Grafana
provisioning/dashboards mounts, and the (now-removed) Promtail config mount
all needed `:Z` added.

### 3. Dashboards: Grafana, provisioned

One starter dashboard (`deploy/compose/grafana/dashboards/soar-overview.json`)
covers HTTP request rate, ingestion outcomes, the action funnel, connector
call outcomes, and scheduler/wazuhpoll p95 duration — the same KPI list
`docs/07-operations/metrics-and-kpis.md` names. Grafana is provisioned with
both datasources (Prometheus, Loki) and the dashboard file at container
start, no manual click-through setup.

### 4. `deploy/compose/docker-compose.core.yml`: new `observability` profile

Replaces the commented placeholder with real `prometheus`, `loki`,
`promtail`, `grafana` services (`profiles: ["observability"]`), matching the
profile scheme this file's own header comment already described.

## Explicitly deferred

- **Distributed tracing (OpenTelemetry spans/OTLP).** This is a single Go
  monolith process with no distributed call chain yet — Temporal/NATS aren't
  implemented (adr/0005 selected them, nothing consumes them today). Adding
  span instrumentation now would connect nothing to nothing; it is deferred
  until there is an actual multi-hop request path worth tracing.
- **Alertmanager / paging.** This pass is dashboards-only, matching this
  project's lab (not on-call) posture.
- **Per-tenant metric breakdown.** Every metric here is process-wide, the
  same simplification `adr/0015-kpi-snapshot.md` already disclosed for `GET
  /v1/metrics` — there is only one tenant in practice today (see
  `services/api/httpx.TenantStubMiddleware`).
- **A generic Docker-API-based container log discovery for Promtail** — works
  on a normal Docker host, but this lab's rootless-podman + SELinux-enforcing
  host denied it regardless of uid mapping (see §2); the file-based approach
  actually committed works everywhere a shared volume works, so this isn't a
  capability gap, just a design choice made after the first one failed here.

## Consequences

- `events.NewHandler`, `cases.NewHandler`, `actions.NewHandler`,
  `scheduler.Sweep`/`Run`, and `wazuhpoll.Run` all gained a trailing
  `*telemetry.Recorder` parameter — every existing call site
  (`services/api/main.go`, `services/api/integration/*_test.go`) was updated;
  tests pass `nil`.
- Metric cardinality is bounded deliberately: HTTP metrics label by route
  *template* (`c.FullPath()`), never the raw path, so an unbounded set of
  case/action/evidence IDs never becomes a Prometheus label value.
- Connector-call outcome recording happens at the `actions.Handler.Execute`
  HTTP layer (success implies the connector call inside `actions.Execute`
  succeeded), not inside the connector call itself — a proxy, not a
  first-class hook into `connectors.Connector.Invoke`; a future pass wanting
  per-attempt (not per-request) connector telemetry would need to thread the
  recorder into `actions.go` directly.
- No Alertmanager, no tracing, no per-tenant breakdown (see above) — each is
  a disclosed, separate follow-up, not attempted here.
- `services/api/main.go` gained `newLogger()` and a `LOG_FILE_PATH` env var
  (opt-in, unset preserves today's stdout-only behavior exactly) — the only
  application-code change this ADR needed, everything else is Compose/config.
- `docker-compose.core.yml` gained a `soar_api_logs` named volume shared by
  `api` and `promtail`, and `web`'s logs are not shipped (only `api`'s —
  `web`'s Next.js output isn't structured JSON, lower value, not attempted
  here).
- This lab's `docker-compose.override.yml` and two new gitignored local-only
  files (`prometheus/prometheus.local.yml`, `grafana/provisioning.local/`)
  encode real findings (§2b) that would otherwise be silently rediscovered —
  they are not committed, but are what was actually run to produce every
  verification claim in this ADR.
- **`podman-compose` (1.6.0) manages every service as one pod**, and was
  observed silently reverting a running `grafana` container back to the
  *committed, non-override* config (base `prometheus:9090`/`loki:3100`
  datasource URLs) after an unrelated `up -d` was run against this same
  project naming only `docker-compose.core.yml` (no `-f
  docker-compose.override.yml`) — confirmed by inspecting the running
  container's actual mounted `datasources.yml` after the dashboard started
  showing `dial tcp: lookup prometheus on ...: server misbehaving` errors in
  a real browser session, caught only because a Playwright-driven check
  loaded the live dashboard rather than trusting an earlier API-only check.
  **Always pass both `-f docker-compose.core.yml -f
  docker-compose.override.yml` together for every `up`/`--force-recreate`
  against this project on this host** — a single-file command can silently
  revert sibling services' overrides, not just the one named.
