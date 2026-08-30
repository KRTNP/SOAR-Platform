# Playbook standard

playbook ต้องอธิบายทั้ง “ทำอะไร” และ “ห้ามทำเมื่อไร” โดยใช้รูปแบบที่มนุษย์อ่านได้และ serialize เป็น YAML/JSON ได้

## Required sections

| Section | เนื้อหา |
|---|---|
| Metadata | id, name, version, owner, type, status, references |
| Trigger | source, event class, matching condition |
| Scope | tenant, asset, user, time และ jurisdiction |
| Preconditions | data, permission, service health และ approval |
| Workflow | `steps` ที่มี typed `kind` สำหรับ validate, deduplicate, investigate, enrich, decision, approval, action, verify, evidence, notify, escalate และ close |
| Evidence | artifact, hash, source, custody และ retention |
| Failure | `on_failure` ของแต่ละ step, retry/dead-letter, manual task และ escalation |
| Safety | stop conditions, blast radius และ rollback |
| Metrics | duration, success, quality, false positive |
| Test | fixtures, mocks, replay และ tabletop reference |

## Playbook type

`type` มีค่าที่เป็นไปได้ 4 แบบ ใช้บอกลักษณะ default output ของ playbook ไม่ใช่ automation level (ดู [automation levels](automation-levels.md) สำหรับ L0-L5 แยกต่างหาก):

| Type | ความหมาย |
|---|---|
| `investigation` | ผลลัพธ์คือ enriched case/verdict; ไม่มี write action ที่เปลี่ยน state ปลายทาง |
| `mitigation` | มี write action ที่จำกัด blast radius ทันที (revoke, isolate, block) ผ่าน approval + action step |
| `remediation` | แก้ต้นเหตุแบบถาวรกว่า mitigation (patch, restore, recovery) มักผ่าน change-management path |
| `notification` | ผลลัพธ์หลักคือ coordination/communication แม้จะมี approval step ปนอยู่ก็ตาม |

## Approval and action steps

`kind: approval` ต้องไม่ผูก logic การเรียก connector ไว้ใน step เดียวกัน — step หนึ่งขออนุมัติเท่านั้น (purpose อธิบาย target, scope, action hash ที่ขอ) แล้วต้องตามด้วย step แยกที่ `kind: action` เป็นคนสั่งเรียก connector จริง ก่อนถึง `kind: verify` เพื่อรักษาขอบเขตของ [ADR-0003 human-in-the-loop](../../adr/0003-human-in-the-loop.md) ให้ตรวจสอบย้อนหลังได้ว่าอนุมัติกับสิ่งที่ execute จริงคือคำขอเดียวกัน

## Playbook quality rules

- ขั้นตอนหนึ่งควรมีวัตถุประสงค์และ output ที่ตรวจสอบได้
- ไม่ผูก logic สำคัญกับชื่อ field ของ vendor
- แยก read/enrich จาก write/contain
- ใช้ explicit condition แทน implicit truthiness
- action ที่ไม่ idempotent ต้องระบุ lock หรือ compensation
- เมื่อ confidence ต่ำให้สร้าง analyst task แทนการตอบสนอง
- ทุกเส้นทางต้องจบด้วย success, dismissed, escalated หรือ failed ที่มีเหตุผล
