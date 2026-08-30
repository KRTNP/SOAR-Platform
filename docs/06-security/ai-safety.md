# AI safety

LLM ใช้ได้กับ:

- สรุป alert/case จากข้อมูลที่อนุญาต
- แนะนำ observable ที่ควร enrich
- ร่าง analyst note และ query template
- แนะนำ playbook/module ที่ใกล้เคียง

LLM ห้ามเป็นผู้ตัดสินสุดท้ายสำหรับ block, isolate, disable, delete หรือ export evidence

ข้อกำหนด:

- แยก untrusted incident text ออกจาก system/policy instruction
- validate output ด้วย schema และ policy engine ซ้ำ
- จำกัด tools ตาม role และ case scope
- บันทึก prompt context ที่ไม่ sensitive, model/version และ output hash
- มี confidence/abstain path
- ทดสอบ prompt injection, data exfiltration, tool abuse และ poisoned document
- ระบบต้องทำงานได้เมื่อ LLM unavailable

