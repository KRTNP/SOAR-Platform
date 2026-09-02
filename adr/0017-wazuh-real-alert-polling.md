# ADR-0017: Real Wazuh alert polling into ingestion

## Status

Accepted — implemented (`services/api/wazuhpoll`) and proven end-to-end
against the real lab Wazuh stack (`~/wazuh-docker/single-node`): a manager
restart produced a genuine alert, `wazuhpoll` polled it from the Indexer,
forwarded it to this same process's `POST /v1/events`, and it landed as a
real case (`case-01m1h7v8aqtyt5h6ct7fqk93cb`, closed immediately after as
test noise, not left in the queue).

## Decision

Implement `docs/05-integrations/wazuh.md`'s baseline flow ("Existing Wazuh
instance → authenticated webhook/API → SOAR ingestion → normalize →
deduplicate → ... → case") as a poller, not a manager-side webhook:

- **`services/api/wazuhpoll`**: a ticker loop (`WAZUH_POLL_ENABLE_REAL`
  master switch, default false — same posture as `OPNSENSE_ENABLE_REAL`)
  that queries the real Wazuh Indexer's `wazuh-alerts-*/_search` (basic auth,
  `WAZUH_INDEXER_*` env vars) for alerts newer than an in-memory checkpoint,
  ascending, up to 50 per pass, and forwards each one as a
  `POST {self}/v1/events` call — the exact same ingestion path a manual curl
  or `services/api/integration/ingestion_test.go` already exercise. The
  checkpoint starts at process-start time, not the beginning of the index,
  so a fresh process does not mass-reingest the lab's existing backlog
  (2353+ historical alerts at the time this was built).
- **Polling, not a manager-side `<integration>` webhook**, because it needs
  no change to the running Wazuh manager container's `ossec.conf` (which
  would require reaching into that container's config and trusting it
  persists across image updates), keeps this entirely soar-platform-owned
  and testable in isolation (see `wazuhpoll_test.go`'s `httptest`-based
  suite — no Testcontainers needed), and matches how most of this
  ingestion-side design baseline already frames connectors as pull-based
  reads (`docs/05-integrations/wazuh.md`'s own connector table is
  read-only `get_agent_context`/`get_alert`/`health`).
- **`source_event_id` is the indexer document's own `_id`**, so a
  re-polled or restart-overlapping alert dedupes naturally via
  `services/api/events`' existing natural-key path
  (`TestIngestion_NaturalKeyDedup`) rather than this package needing its own
  dedup state.
- **`eventClassFromGroups` maps Wazuh's open `rule.groups` taxonomy onto
  `event.schema.json`'s closed `event_class` enum**
  (authentication/network/process/file/email/vulnerability/cloud/malware/
  other). This was not a design guess — an early version passed
  `rule.groups[0]` straight through and got a real 400 from the real
  indexer's first alert (`groups: ["ossec"]`, the manager's own internal-log
  group, not a member of the enum). The mapping is necessarily lossy
  (`web`/`firewall`/`ids`/`attack` all collapse to `network`) with `other`
  as the catch-all for anything unrecognized — `wazuhpoll_test.go`'s
  `TestEventClassFromGroups_AlwaysInSchemaEnum` pins every mapped case to a
  real enum member so this specific validation gap cannot silently
  regress.
- **`severityFromLevel`/`confidenceFromLevel`** bucket Wazuh's 0–15
  `rule.level` onto this platform's severity enum and a normalized
  `level/15` confidence — both are documented heuristics, not an
  authoritative Wazuh convention or a calibrated probability.
- **`data.srcip`/`data.dstip`/`data.srcuser` become observables** when
  present (per `docs/05-integrations/wazuh.md`'s mapping list); most alert
  types (like the manager's own health logs used to prove this end-to-end)
  carry none, and the event is forwarded with no `observables` field at all
  rather than a guessed one — confirmed optional by
  `TestIngestion_IdempotencyKeyReplay`'s existing fixture.
- **`services/api/config.WazuhIngestBaseURL`** is derived from `Addr`
  (`127.0.0.1:<port>`), never a separate env var — this package always
  forwards to its own process, so there is nothing for an operator to get
  wrong here.

### Deployment note (not code, but load-bearing for "real")

Running this against the real lab Wazuh Indexer from the containerized
`deploy/compose` stack required `network_mode: host` for the `api`/`web`
services in a local, gitignored `docker-compose.override.yml` — the
podman-compose bridge network's `host.containers.internal` (pasta's
169.254.1.2 gateway) could reach the Indexer's DNS name but connections were
refused against `rootlessport`'s published `0.0.0.0:9200`, so container
networking could not reach a rootless-podman-published host port. Host
networking matches how this stack ran before being containerized (bare
`go run`) and is what OPNsense's `10.10.10.1` (the `virbr-lab` libvirt
bridge, host-local) also needs.

## Rationale

The design baseline (`docs/05-integrations/wazuh.md`) already specified the
flow and field mapping before any Go code existed; the real Wazuh stack
(`~/wazuh-docker/single-node`) and lab VM (`opnsense-lab`) already existed
from ADR-0013's sandbox build. This closes the one piece of that flow with
no code yet — turning "detection source" from a manual curl into an actual
running poller — following the same mock-now/real-when-ready sequencing
ADR-0010–0013 already used for their own pieces, except here "real" was
achievable directly since the sandbox was already standing.

## Consequences

- **No persisted checkpoint.** A process restart resets the checkpoint to
  "now," so any alert that arrived during the restart window is silently
  never polled — accepted for lab scope, same class of gap ADR-0014's
  scheduler discloses for its own lack of distributed locking. A durable
  checkpoint (a small Postgres row, or reusing `idempotency_key`-style
  natural dedup more aggressively with a wider lookback window) is the
  documented upgrade path if this matters later.
- **A permanently-failing alert (e.g. an ingest 400 from a future schema
  mismatch) is logged and skipped, not retried or alerted on** — the
  checkpoint still advances past it, so it is not stuck forever, but it is
  also never recovered. Same "log and skip" posture `services/api/scheduler`
  already accepts for its own per-candidate failures.
- **`eventClassFromGroups`'s collapsing is real information loss** — an
  analyst reading a case's `category` cannot tell "web attack" from
  "firewall block" from "IDS alert," all three land as `network`. The full
  `rule.groups` list is still carried in the forwarded payload's `rule`
  object (not modeled further by `events.Normalize`, but present in
  `raw_ref`), so it is not lost, just not queryable as `category` today.
- **`network_mode: host` in the local override is a broader network surface
  than the bridge-networked default** — acceptable for a single-user
  personal-lab host per this session's explicit authorization to connect
  real integrations, not a change to the committed, still-bridge-networked
  `docker-compose.core.yml` default anyone else running this repo gets.
- Real auth/RBAC is still not built (`adr/0005`) — this poller's forwarded
  events are ingested under the same `DEFAULT_TENANT_ID` stub every other
  ingestion path already uses.
