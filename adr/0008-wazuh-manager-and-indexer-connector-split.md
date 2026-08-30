# ADR-0008: Split the Wazuh connector into manager and indexer boundaries

## Status

Accepted — design only; implementation not started.

## Decision

Represent the Wazuh integration as two connector manifests instead of one:
`connector-wazuh-manager-lab` (provider `wazuh-manager`, Wazuh Manager RESTful API on
port 55000) exposing `get_agent_context`, `get_rule` and `health`; and
`connector-wazuh-indexer-lab` (provider `wazuh-indexer`, Wazuh Indexer/OpenSearch API on
port 9200) exposing `get_alert` and `health`. Both stay investigation-only; no
active-response or write intent is exposed by either connector in this iteration.

## Rationale

Wazuh alert documents live in the Wazuh Indexer's `wazuh-alerts-*` index, while agent
inventory and rule metadata are served by the separate Wazuh Manager API. The two APIs
authenticate differently — the manager mints a short-lived JWT from a Basic Auth call to
`/security/user/authenticate`, while the indexer uses OpenSearch security (Basic Auth or
API key) scoped to an index pattern — and they run as separate services in the lab
(`single-node_wazuh.manager_1:55000`, `single-node_wazuh.indexer_1:9200`).
`schemas/connector.schema.json` models exactly one `auth` block per connector, so a
single manifest would have to pick one credential and quietly proxy the other API
through it, hiding a real trust-boundary difference.

## Consequences

- A playbook step that needs both alert detail and agent/rule context calls two
  connectors, not one.
- Each manifest carries its own rate limit, target allowlist and error classes tuned to
  its backend.
- RBAC/index-scope names in both manifests are drafted from Wazuh 4.9 documentation, not
  verified against the lab's running instance. The rule in
  [provider capability matrix](../docs/05-integrations/provider-capability-matrix.md)
  against inferring capability from endpoint names still applies — confirm scopes in
  sandbox before enabling real credentials.
