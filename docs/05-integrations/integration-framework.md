# Integration framework

## Connector contract

ทุก connector ต้องประกาศ:

- connector ID/version/vendor
- supported intents และ risk class
- input/output schema
- auth method และ required scopes
- timeout/retry/rate-limit behavior
- idempotency support
- dry-run support
- health check
- error taxonomy
- data classification and egress destination

## Adapter layers

```text
Playbook intent
  → domain action
  → connector adapter
  → provider API/CLI
  → provider result
  → normalized action result
```

Connector ห้ามรับ raw user-supplied URL/command โดยตรง ต้องผ่าน validation, target allowlist และ policy engine

## Auth scope naming

`auth.scopes` ใช้ syntax ตาม native RBAC/permission model ของแต่ละ provider ตรงๆ (ไม่บังคับ convention เดียวทั้ง repo) เช่น OPNsense ใช้ dot notation ของ alias permission (`firewall.alias.read`), Wazuh manager ใช้ colon notation ของ RBAC action (`agent:read`), Wazuh indexer/OpenSearch ใช้ index-scoped role (`read:wazuh-alerts-*`) — ต้อง verify scope string จริงกับ provider ก่อนใช้ credential จริงเสมอ ตาม [provider capability matrix](provider-capability-matrix.md)

