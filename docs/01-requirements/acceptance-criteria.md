# Acceptance criteria

## MVP end-to-end

- รับ fixture alert ที่มี source, event time, severity และ observable อย่างน้อยหนึ่งรายการ
- ปฏิเสธ payload ที่ไม่มี signature หรืออยู่นอก tenant/scope
- event เดิมซ้ำต้องอ้าง execution เดิมหรือถูก mark เป็น duplicate โดยไม่เปิด case ซ้ำ
- enrichment failure ต้องบันทึก partial result และไปเส้นทาง retry/escalation
- case ต้องมี owner, status, severity, evidence reference และ execution ID
- action risk สูงต้องหยุดที่ approval และห้ามเรียก target ก่อน approve
- approve แล้วต้องเรียก connector ด้วย idempotency key และบันทึก result
- connector timeout ต้องกลายเป็น actionable failure ไม่ใช่ success
- export ต้อง redaction secret และตรวจ manifest hash ได้
- replay fixture ต้องให้ decision เดิมเมื่อ policy/schema version เดิม

