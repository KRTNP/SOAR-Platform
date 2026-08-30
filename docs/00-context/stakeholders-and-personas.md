# Stakeholders and personas

| Persona | เป้าหมาย | สิทธิ์หลัก |
|---|---|---|
| Tier-1 analyst | triage และ enrich alert | read, run investigation, propose action |
| Tier-2 investigator | วิเคราะห์ incident และปรับ scope | read/write case, run approved collection |
| Incident commander | ตัดสินใจ containment และสื่อสาร | approve high-impact action |
| DFIR analyst | รักษาหลักฐานและทำ timeline | evidence, custody, forensic workflow |
| Detection engineer | สร้าง rule และ mapping | detection/playbook authoring |
| Platform operator | ดูแล availability และ connector | deployment, health, no case approval by default |
| Auditor | ตรวจ traceability และ policy | read-only audit/evidence |
| Service owner | ยืนยัน criticality และ business impact | asset context, approval when service affected |

หลักการแยกหน้าที่: ผู้สร้าง playbookไม่ควรเป็นผู้อนุมัติ action ของ playbook เดียวกันใน production และผู้ดูแล platform ไม่ควรได้สิทธิ์อ่าน evidence ทั้งหมดโดยอัตโนมัติ

