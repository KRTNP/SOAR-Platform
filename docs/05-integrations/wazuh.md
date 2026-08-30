# Wazuh integration

## Baseline flow

Existing Wazuh instance → authenticated webhook/API → SOAR ingestion → normalize → deduplicate → playbook mapping → case

Wazuh ระบุแนวทางส่ง alert เข้า Shuffle workflow ผ่าน webhook ในเอกสาร external API integration [Wazuh external API integration](https://documentation.wazuh.com/current/user-manual/manager/integration-with-external-apis.html)

## Mapping

เก็บอย่างน้อย:

- alert ID, rule ID, rule level และ rule description
- agent ID/name, manager, host และ asset reference
- timestamp/timezone
- decoder, location และ full_log reference
- source/destination IP, user, process, file และ hash ที่พบ
- MITRE technique/tactic ถ้ามี

## Connectors

| Connector | Operation | Backend | Auth | Risk | Post-condition |
|---|---|---|---|---|---|
| `connector-wazuh-manager-lab` | `get_agent_context` | Manager API (`:55000`) | JWT via vault-stored service account | low | Agent record returned; no state change |
| `connector-wazuh-manager-lab` | `get_rule` | Manager API (`:55000`) | JWT via vault-stored service account | low | Rule definition returned; no state change |
| `connector-wazuh-manager-lab` | `health` | Manager API (`:55000`) | JWT via vault-stored service account | read-only | API reachable and authenticated |
| `connector-wazuh-indexer-lab` | `get_alert` | Indexer API (`:9200`, `wazuh-alerts-*`) | Basic/API key via vault-stored service account | low | Matching alert document(s) returned; no state change |
| `connector-wazuh-indexer-lab` | `health` | Indexer API (`:9200`) | Basic/API key via vault-stored service account | read-only | Index reachable and authenticated |

Manifests: [`examples/wazuh-manager-connector.json`](../../examples/wazuh-manager-connector.json) and
[`examples/wazuh-indexer-connector.json`](../../examples/wazuh-indexer-connector.json). Rationale for the
manager/indexer split is in [ADR-0008](../../adr/0008-wazuh-manager-and-indexer-connector-split.md).
RBAC/index-scope names are drafted from Wazuh 4.9 documentation and are unverified until tested against
the lab's running instance, per [provider capability matrix](provider-capability-matrix.md).

Neither connector exposes a response/write intent. Wazuh's own active response stays a Wazuh-native,
SOAR-independent mechanism per the Safety section below.

## Safety

- webhook ต้อง authenticate และตรวจ replay
- filter noisy rule ก่อนเปิด response playbook
- active response ของ Wazuh แยกจาก SOAR response และต้องป้องกัน action ซ้ำ
- ใช้ Wazuh เป็น detection source; case/evidence state อยู่ใน SOAR/DFIR boundary

MVP ใช้ Wazuh จริงตาม [lab integration profile](lab-integration-profile.md) ไม่ใช่เพียง fixture; fixture ยังใช้สำหรับ replay และ negative tests เท่านั้น
