# Contributing

## Design changes

การเปลี่ยนแปลงที่กระทบ domain model, trust boundary, response permission, schema หรือ external API ต้องมีเอกสาร ADR ใน `adr/` และอัปเดตเอกสารที่เกี่ยวข้องใน commit เดียวกัน

## Playbook changes

ทุก playbook ต้องมี owner, trigger, scope/preconditions, stop conditions, evidence output, approval policy, rollback และ test case ห้ามเพิ่ม action ที่มีผลกระทบสูงโดยไม่มี dry-run และ approval path

## Schema changes

เพิ่ม field แบบ backward-compatible เป็นค่าเริ่มต้น หากลบหรือเปลี่ยนความหมายของ field ให้เพิ่ม schema version และ migration note ใน `docs/03-domain-model/normalization-strategy.md`

## Safety

ห้ามใส่ token, password, private URL, client data หรือ incident artifact จริงในตัวอย่าง เอกสาร และ test fixture ใช้ค่าจำลองเท่านั้น
