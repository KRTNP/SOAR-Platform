# Connector catalog

| Connector | Investigation intents | Response intents | Default risk |
|---|---|---|---|
| Wazuh | get alert, rule, agent context | none (lab manifests are investigation-only; see [ADR-0008](../../adr/0008-wazuh-manager-and-indexer-connector-split.md)) | low |
| SIEM | bounded search, related event | none | low |
| EDR/XDR | process, host, network, artifact | isolate, kill, quarantine | high |
| Firewall | rule lookup, indicator sightings | block/unblock | high |
| Identity/AD | sign-in, MFA, role, token | revoke/reset/disable | high |
| TI/STIX/TAXII | indicator reputation, sightings | publish indicator | low/medium |
| Mail security | header, recipients, message search | quarantine/release | medium/high |
| Ticketing | ticket lookup/create/update | close/escalate | low |
| DFIR | targeted collection, timeline | none by default | medium |

ก่อนเปิด connector ต้องมี owner, credential scope, data classification, health check, contract tests และ target allowlist

