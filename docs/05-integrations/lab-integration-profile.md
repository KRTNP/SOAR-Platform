# Lab integration profile

## Real integrations

| Provider | Connection | Capability enabled | Safety boundary |
|---|---|---|---|
| Wazuh | Existing API/webhook | Alert ingest, agent context and read-only enrichment | Lab tenant and authenticated source only |
| OPNsense | Lab VM API | Add/remove source IP in `SOAR_TEMP_BLOCK` alias | One approved IP, ten-minute TTL, one-person approval |
| Ubuntu victim | Wazuh agent | SSH telemetry, inventory and lab evidence trigger | Snapshot and isolated subnet |
| Kali | Lab VM | Authorized test traffic generator | Victim targets only |

## Real when available, otherwise connector boundary

MISP, Velociraptor, SIEM search, EDR, identity, ticketing and additional firewall
providers have connector interfaces in the design. They are enabled with real
credentials only after their sandbox, scope, rate limit and capability manifest are
recorded. An unavailable provider is represented by a deterministic mock, not by
fake production success.

## Wazuh contract

1. Authenticate webhook/API request and reject replay/oversize payloads.
2. Retain a redacted raw payload reference and checksum.
3. Map provider fields to canonical event/alert/observable objects.
4. Deduplicate using tenant, provider and source event identity.
5. Enrich in read-only mode.
6. Create/update case and analyst task.
7. Emit execution/audit/outbox events.

Investigation/enrichment during a case uses two read-only connectors instead of the webhook path,
split per [ADR-0008](../../adr/0008-wazuh-manager-and-indexer-connector-split.md):

### connector-wazuh-manager-lab

| Operation | Risk | Approval | Post-condition |
|---|---|---|---|
| `get_agent_context` | low | no | Agent record returned; no state mutation |
| `get_rule` | low | no | Rule definition returned; no state mutation |
| `health` | read-only | no | API reachable and authenticated |

### connector-wazuh-indexer-lab

| Operation | Risk | Approval | Post-condition |
|---|---|---|---|
| `get_alert` | low | no | Matching alert document(s) returned; no state mutation |
| `health` | read-only | no | Index reachable and authenticated |

## OPNsense contract

The connector exposes only these lab operations:

| Operation | Risk | Approval | Post-condition |
|---|---|---|---|
| `preview_temp_block` | low | no | Planned alias change returned; no state mutation |
| `add_temp_block` | high | one incident commander | IP is in `SOAR_TEMP_BLOCK` and SSH traffic is blocked |
| `remove_temp_block` | high | inherited from approved action | IP is absent and traffic follows baseline policy |
| `health` | read-only | no | API and alias/rule state are reachable |

The connector never accepts arbitrary firewall commands. It accepts a typed intent with
IP, alias, TTL, case, approval ID and idempotency key, then constructs the provider
request internally.

