# SOAR Platform Design

เอกสารออกแบบระบบ Security Orchestration, Automation and Response (SOAR) สำหรับเชื่อม SIEM, EDR, firewall, identity, threat intelligence, ticketing และ DFIR ให้ทำงานเป็นกระบวนการเดียวกัน

สถานะของ repository นี้คือ **design baseline**: ยังไม่เปิดสิทธิ์ตอบสนองต่อระบบจริง และตัวอย่างทั้งหมดต้องทำงานใน lab หรือ dry-run จนกว่าจะมี authorization, approval และ rollback ที่ชัดเจน

## เป้าหมาย

- ลดเวลาการ triage และ context switching ของ analyst
- ทำให้ incident response ทำซ้ำได้ ตรวจสอบย้อนหลังได้ และวัดผลได้
- แยก investigative action ออกจาก containment/remediation action
- รักษาหลักฐานและ chain of custody ตามแนวทางของ workspace `dfir/`
- ใช้ schema กลางที่รองรับ vendor หลายราย
- เปิดให้ human-in-the-loop ควบคุม action ที่มีผลกระทบสูง

## แผนที่เอกสาร

| หมวด | เนื้อหา |
|---|---|
| Context | ปัญหา ผู้เกี่ยวข้อง ขอบเขต และศัพท์กลาง |
| Requirements | ความสามารถ ข้อจำกัด use case และเกณฑ์ยอมรับ |
| Architecture | service boundaries, data flow, deployment และ resilience |
| Domain model | event, alert, case, observable, evidence และ action |
| Playbooks | รูปแบบ lifecycle, automation tier, approval และ catalog |
| Integrations | Wazuh, SIEM, EDR, firewall, identity, TI, ticketing, DFIR |
| Security | threat model, RBAC, secrets, audit, safe response และ AI safety |
| Operations | metrics, observability, backup, maintenance และ troubleshooting |
| Testing | connector, workflow, replay, tabletop และ failure injection |
| Governance | policy, privacy, retention, custody และ compliance mapping |
| Roadmap | MVP, milestones และ architecture decisions |

## Design principles

1. **Evidence before action** — ทุกการตอบสนองต้องอ้างอิงข้อมูลที่ตรวจสอบได้
2. **Least privilege** — connector มีสิทธิ์เท่าที่จำเป็นต่อ action เดียว
3. **Human judgment for high impact** — block, isolate, disable และ delete ต้องมี gate
4. **Idempotent by default** — การส่ง event ซ้ำต้องไม่ทำ action ซ้ำจนเกิดผลเสีย
5. **Fail closed** — เมื่อข้อมูลไม่ครบหรือ policy ประเมินไม่ได้ ให้หยุดและ escalate
6. **Business-aware response** — ประเมิน criticality และผลกระทบต่อบริการก่อน containment
7. **Version everything** — schema, connector, playbook, policy และ evidence manifest ต้อง version ได้
8. **Provider-neutral core** — vendor adapter อยู่ที่ขอบระบบ ไม่กระจาย vendor field ไปทั่ว domain

## แหล่งอ้างอิงหลัก

- [NIST SP 800-61 Rev. 3](https://csrc.nist.gov/pubs/sp/800/61/r3/final)
- [CISA Cybersecurity Incident and Vulnerability Response Playbooks](https://www.cisa.gov/sites/default/files/2024-08/Federal_Government_Cybersecurity_Incident_and_Vulnerability_Response_Playbooks_508C.pdf)
- [OASIS CACAO Security Playbooks 2.0](https://docs.oasis-open.org/cacao/security-playbooks/v2.0/security-playbooks-v2.0.html)
- [OCSF](https://ocsf.io/)
- [OASIS TAXII 2.1](https://docs.oasis-open.org/cti/taxii/v2.1/taxii-v2.1.html)
- [Sigma Correlation Specification](https://sigmahq.io/sigma-specification/specification/sigma-correlation-rules-specification.html)

## เอกสารเริ่มต้น

อ่านตามลำดับนี้:

1. [Mission and scope](docs/00-context/mission-and-scope.md)
2. [System overview](docs/02-architecture/system-overview.md)
3. [Domain model](docs/03-domain-model/event-model.md)
4. [Playbook standard](docs/04-playbooks/playbook-standard.md)
5. [MVP definition](docs/10-roadmap/mvp-definition.md)

สำหรับคำตอบตรง ๆ ว่า design ครบระดับใด ให้ดู [Completeness audit](docs/10-roadmap/completeness-audit.md), [Decision register](docs/10-roadmap/decision-register.md), [Source catalog](docs/00-context/source-catalog.md), [Source-to-design traceability](docs/09-governance/source-to-design-traceability.md) และ [Definition of complete](docs/10-roadmap/definition-of-complete.md)

เอกสาร implementation-level ที่ควรอ่านต่อคือ [Software development blueprint](docs/02-architecture/software-development-blueprint.md), [API design](docs/02-architecture/api-design.md), [Data architecture](docs/02-architecture/data-architecture.md), [Lab deployment profile](docs/02-architecture/lab-deployment-profile.md), [Lab integration profile](docs/05-integrations/lab-integration-profile.md), [Execution semantics](docs/02-architecture/execution-semantics.md), [Response decision matrix](docs/04-playbooks/response-decision-matrix.md), [Data-source coverage](docs/01-requirements/data-source-coverage.md) และ [Provider capability matrix](docs/05-integrations/provider-capability-matrix.md)

Machine-readable API contract: [api/openapi.yaml](api/openapi.yaml)

Playbook ที่อิงแหล่งจริงอยู่ใน [source-derived playbooks](docs/04-playbooks/source-derived-playbooks.md) และตรวจด้วย `validation/validate-source-traceability.sh`
