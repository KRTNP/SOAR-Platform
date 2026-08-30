# Completeness audit

เอกสารนี้ตอบคำถามว่า design ปัจจุบัน “พอ” ในระดับใด โดยแยก design coverage ออกจาก production readiness

## Current assessment

| Area | Status | Evidence | ยังขาดอะไร |
|---|---|---|---|
| Context/mission | Covered | context docs, personas, constraints, glossary | validate กับ stakeholders จริง |
| Research/standards | Source-traceable baseline | source catalog, exact sections/pages, source-to-design matrix, NIST/CISA/CACAO/OCSF/TAXII/Sigma and research papers | organization-specific policy mapping and provider conformance evidence |
| Domain model | Covered baseline | event/alert/case/observable/evidence/action schemas, sanitized examples and playbook schema | migration, DB indexes, full runtime conformance tests |
| Architecture | Decision-complete design baseline | component, data flow, trust boundary, storage, execution semantics, selected stack, API/data/deployment profiles and [decision register](decision-register.md) | deployed reference implementation |
| Playbook method | Covered baseline | standard, lifecycle, risk tier, approval, test methodology | provider-specific executable steps |
| Playbook scenarios | Broad and source-derived | 16 operational YAML contracts (including the first end-to-end SSH brute-force → OPNsense scenario, [ADR-0007](../../adr/0007-first-vertical-scenario.md)) plus 1 authoring template, with references, scope, evidence, metrics, source IDs where applicable, adaptation notes, safety gates and tests | exact field/query/action mapping ต่อ environment |
| Integrations | Design-complete for lab path | OpenAPI contract, Wazuh/OPNsense lab profile, connector catalog, machine-readable OPNsense/Wazuh-manager/Wazuh-indexer manifests ([ADR-0008](../../adr/0008-wazuh-manager-and-indexer-connector-split.md)) and error taxonomy | live connector implementation and sandbox evidence |
| Security | Covered baseline | threat model, RBAC, secrets, audit, safe response, AI safety | formal STRIDE review, security test evidence |
| DFIR/evidence | Covered baseline | evidence model, custody, read-only boundary | actual case API/export adapter |
| Operations | Covered baseline | metrics, SLO proposal, backup, outage, maintenance, resource profile | measured SLO and restore drill |
| Testing | Defined and traceable | test strategy, matrix, replay, tabletop, failure injection, lab scenario, source-traceability validator and strict playbook contract validation | runnable test harness and target-environment fixtures |
| Governance | Defined | policy, RACI, retention, change, compliance mapping | approved organizational policy |

## Answer

- **Context:** พอสำหรับ review และออกแบบ MVP; source catalog และ traceability ปิดช่องว่างเรื่องแหล่งอ้างอิงแล้ว
- **เนื้อหา:** กว้างและครอบคลุม architecture baseline แต่ยังไม่ใช่รายละเอียด implementation ทุก service
- **ข้อมูล/เทคนิค:** พอสำหรับกำหนด direction; ต้องเติมข้อมูลจริงของ Wazuh, SIEM, EDR, firewall, identity และ asset inventory ก่อน lock policy
- **Playbook:** มีทั้ง template, scenario catalog และ source-derived coordination/vulnerability/recovery/DFIR pack; ยังไม่ใช่ production-runbook เพราะยังไม่มี provider-specific query, API request/response, fixture และ target sandbox
- **Production:** ยังไม่พร้อมเปิด automation ที่เปลี่ยนระบบจริง
- **Decision record:** คำตอบและค่าที่เลือกจากการคุยถูกบันทึกครบใน [decision register](decision-register.md) และรายละเอียดแยกอยู่ใน blueprint/ADR/API/data/lab documents

## Definition of ready for implementation

ถือว่า source-traceable design baseline พร้อมให้เริ่มเขียน engine เมื่อมีครบ:

1. canonical schema และ compatibility test
2. OpenAPI/API error contract
3. connector manifest และ mock contract ต่อ provider แรก
4. deterministic risk/approval policy
5. one end-to-end playbook ที่ replay ได้
6. audit/evidence export ที่ตรวจ hash ได้
7. test fixture สำหรับ success, duplicate, timeout, deny และ post-condition failure
8. owner, scope และ rollback ของ target sandbox
9. source catalog, source-to-design matrix และ source-derived playbook validator ผ่าน

## Definition of ready for production

นอกจากข้อข้างบน ต้องมี written authorization, secrets manager, SSO/MFA, least-privilege credentials, restore drill, tabletop, measured KPI, change approval และ service-owner sign-off
