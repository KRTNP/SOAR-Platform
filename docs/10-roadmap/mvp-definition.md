# MVP definition

## Included

- authenticated webhook ingestion
- canonical event/alert/observable/case model
- deduplication ด้วย deterministic idempotency key
- real Wazuh API/webhook connector using the existing Wazuh instance
- IOC enrichment แบบ mock/local provider
- case creation และ analyst task
- approval gate สำหรับ OPNsense temporary IP block intent
- audit/evidence metadata
- dry-run, replay และ validation scripts
- Mermaid architecture source

## Deferred

- production firewall/EDR write connectors beyond the lab OPNsense action
- identity/session-revoke write connectors (account-compromise, cloud-identity-compromise and credential-stuffing playbooks stay dry_run/approval-only; see [decision register](decision-register.md) #34-#50, scoped to the OPNsense action only)
- real ticketing provider connector — [milestones](milestones.md) M2 requires only a mock ticket connector for its contract-test gate, not a production integration
- multi-tenant federation
- visual workflow editor
- automated playbook generation
- LLM agent execution
- large-scale data lake/query engine

## MVP exit criteria

ผ่าน acceptance criteria ทั้งหมดใน `docs/01-requirements/acceptance-criteria.md`, มี [SSH brute-force lab scenario](../08-testing/lab-scenario-ssh-bruteforce.md), มี playbook replay fixture, มี approval/post-condition/rollback tests และตรวจได้ว่า external target นอก lab ไม่ถูกเปลี่ยน
