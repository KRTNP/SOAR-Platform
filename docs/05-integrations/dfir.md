# DFIR integration

SOAR ทำหน้าที่สร้างและประสาน forensic task ไม่แทนที่ forensic tool

workflow ต้อง:

1. ตรวจ written scope, case ID, timezone และ target
2. อ้าง working copy/read-only source
3. สั่ง collection ผ่าน launcher ที่บันทึก command/version ได้
4. hash input/output และผูก evidence manifest
5. สร้าง timeline/correlation task
6. ส่งผลกลับเป็น evidence reference ไม่ใช่คัดลอกข้อมูลลับลง chat

สอดคล้องกับกฎของ workspace `dfir/` ซึ่งอยู่นอก repository นี้ — `dfir/playbooks/README.md` และ `dfir/AGENTS.md`

