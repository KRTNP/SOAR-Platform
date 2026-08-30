# Alert model

alert เป็นการประกาศว่า event หรือกลุ่ม event ตรงเงื่อนไข detection และต้องมี triage

ต้องเก็บ:

- detection rule ID, version และ source
- event references และ correlation window
- severity ก่อน/หลัง enrichment
- false-positive rationale ถ้าถูก dismiss
- playbook mapping และ trigger reason
- triage state: new, acknowledged, investigating, escalated, resolved, false_positive
- analyst notes แบบมี author/time
- related cases และ duplicate-of reference

Alert ไม่ควรเปลี่ยนเป็น incident อัตโนมัติเพียงเพราะ severity สูง หากยังไม่ผ่าน correlation หรือ policy ที่กำหนดไว้

