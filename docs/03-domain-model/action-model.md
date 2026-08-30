# Action model

action เป็น intent ที่ถูกส่งให้ connector ไม่ใช่คำสั่ง raw จากผู้ใช้

## Action contract

```json
{
  "action_id": "act-001",
  "intent": "block_network_indicator",
  "target": {"type": "ipv4", "value": "198.51.100.10"},
  "case_id": "case-001",
  "risk": "high",
  "mode": "dry_run",
  "requested_by": "analyst-001",
  "approval": {"required": true, "status": "pending"},
  "idempotency_key": "case-001:block:ipv4:198.51.100.10",
  "policy_version": "policy-1.0"
}
```

ผลลัพธ์ต้องแยก `transport_status`, `provider_status`, `postcondition_status`, `changed`, `rollback_ref` และ `error_class`

