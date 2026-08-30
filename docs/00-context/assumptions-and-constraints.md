# Assumptions and constraints

Paths in this document are relative to the `soar-platform/` project root.

## Assumptions

- SIEM/EDR/Wazuh/provider source เป็น source of truth สำหรับ raw event ของตนเอง
- SOAR ได้รับเฉพาะข้อมูลที่ scope อนุญาต
- ทุก target มี asset owner และ criticality อย่างน้อยระดับหนึ่ง
- connector ภายนอกอาจช้า, rate-limit หรือ unavailable ได้
- time zone ของ event ต้องเก็บเป็น UTC พร้อม original timestamp

## Constraints

- ห้ามแก้ไข original evidence ใน `../dfir/evidence/<case>/`
- derived data ต้องอยู่ใน `../dfir/cases/<case>/work/` หรือ `../dfir/exports/<case>/` ตาม `../dfir/AGENTS.md`
- ห้ามเก็บ secret ใน source, workflow YAML หรือ example JSON
- action ที่กระทบ production ต้องมี authorization และ policy gate
- network egress ของ worker ต้องจำกัดตาม connector allowlist
- system ต้องทำงานได้โดยไม่มี LLM
