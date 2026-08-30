# Research and best practices

แหล่งอ้างอิงแบบมี source ID, section/page ที่ตรวจ และ mapping ไปยัง design อยู่ใน [source catalog](source-catalog.md) และ [source-to-design traceability](../09-governance/source-to-design-traceability.md)

เอกสารนี้สรุปแหล่งอ้างอิงที่ใช้กำหนด design baseline และแยกข้อกำหนดจากมาตรฐานออกจากข้อเสนอเชิงออกแบบของโปรเจกต์

## Standards and government guidance

| Source | สิ่งที่นำมาใช้ |
|---|---|
| [NIST SP 800-61 Rev. 3](https://csrc.nist.gov/pubs/sp/800/61/r3/final) | integrate incident response กับ CSF 2.0, preparation, response, recovery และ improvement |
| [CISA Incident/Vulnerability Playbooks](https://www.cisa.gov/sites/default/files/2024-08/Federal_Government_Cybersecurity_Incident_and_Vulnerability_Response_Playbooks_508C.pdf) | standardized checklist, threshold, role, coordination, remediation และ tracking |
| [OASIS CACAO 2.0](https://docs.oasis-open.org/cacao/security-playbooks/v2.0/security-playbooks-v2.0.html) | metadata, workflow, agents/targets, versioning, markings และ signatures |
| [OCSF](https://ocsf.io/) | vendor-neutral security event schema และ normalization |
| [OASIS TAXII 2.1](https://docs.oasis-open.org/cti/taxii/v2.1/taxii-v2.1.html) | CTI exchange ผ่าน RESTful collections/channels ที่ออกแบบมารองรับ STIX |
| [Sigma Correlation](https://sigmahq.io/sigma-specification/specification/sigma-correlation-rules-specification.html) | event/value count, temporal correlation, grouping, time window และ false positives |

## Research findings

### SOAR usability and over-automation

[Testing SOAR Tools in Use](https://arxiv.org/abs/2208.06075) ทดลองกับผู้ใช้ 24 คนและ SOAR 6 ตัว พบว่า configuration มีผลมาก, SOAR ลด context switching และเพิ่ม efficiency แต่ ticket accuracy/completeness ลดลง และ over-automation เป็น concern ของ analyst อาวุโส

**Design consequence:** ระบบต้องมี decision support, readable evidence, approval และ quality metric ไม่วัดเฉพาะจำนวน automation

### Operations-aware playbooks

[Operations-informed Incident Response Playbooks](https://orca.cardiff.ac.uk/id/eprint/165303/) เสนอให้ playbook เชื่อม process model กับ model ของ operation และ critical threshold เพื่อสะท้อนผลกระทบต่อธุรกิจ/บริการ

**Design consequence:** ก่อน isolate, shutdown หรือ patch ต้องโหลด asset criticality, service owner และ business dependency

### Investigative vs response playbooks

[IC-SECURE](https://arxiv.org/abs/2311.03825) แยก investigative playbook ที่เก็บ context ออกจาก response playbook ที่ทำ mitigation และมอง playbook เป็น directed graph ของ modules/conditions

**Design consequence:** investigation default เป็น read-only; response ใช้ risk tier, approval และ post-condition

### Integration/API mapping

[APIRO](https://arxiv.org/abs/2201.07959) และ [IRP2API](https://ieeexplore.ieee.org/document/10123645) ชี้ว่าการ map ขั้นตอน IR ไปยัง API ของ security tools เป็นภาระสำคัญเพราะ API มีรูปแบบและ semantics ต่างกัน

**Design consequence:** ใช้ normalized action intent และ connector contract แทนการให้ playbook เรียก vendor API โดยตรง

### Automation maintenance

[SANS State of Automation in Security Operations](https://www.sans.org/white-papers/state-automation-security-operations) รายงานว่าความพยายามในการ deploy และ maintain automation เป็นอุปสรรคใหญ่ของ SOAR

**Design consequence:** ต้องมี ownership, versioning, health checks, replay regression, dead-letter และ maintenance metrics ตั้งแต่ MVP

## Derived best practices

1. เริ่มจาก repetitive, low-risk, high-volume workflow
2. เขียน playbook จาก IR policy และ business impact ไม่ใช่เริ่มจาก connector ที่มีอยู่
3. normalize ก่อน correlate และอย่าให้ core อ้าง vendor field
4. แยก investigation, decision, approval, action และ verification
5. ทุก action ต้องมี scope, idempotency, timeout, retry, post-condition และ audit
6. ใช้ fail-closed เมื่อ identity, scope, evidence หรือ policy ไม่ชัด
7. ทดสอบด้วย replay, mock, tabletop และ failure injection
8. เก็บ metrics ด้าน speed, quality, safety และ business impact
9. ใช้ LLM เป็นผู้ช่วยร่าง/สรุป ไม่ใช่ผู้บังคับ policy หรือผู้ยิง destructive tool
10. lessons learned ต้องย้อนกลับไปแก้ detection, playbook, policy และ training
