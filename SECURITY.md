# Security model

ระบบนี้ถูกออกแบบให้เป็น security control plane ที่มีสิทธิ์เรียกเครื่องมืออื่น จึงต้องถือว่า SOAR เป็น high-value target

ข้อกำหนดขั้นต่ำ:

- แยก read-only connector จาก response connector
- ใช้ short-lived credentials และ secret manager
- บังคับ tenant, asset และ action allowlist
- บันทึกผู้ร้องขอ ผู้อนุมัติ ผู้ปฏิบัติ input/output และผลลัพธ์
- ป้องกัน replay ด้วย request ID และ idempotency key
- ปฏิเสธ action เมื่อ target หรือ confidence ไม่ครบ
- ไม่ส่ง evidence ไป third-party service โดยไม่มี authorization
- LLM เป็นผู้ช่วยเสนอผล ไม่ใช่ policy enforcement point

เมื่อพบ security concern ให้หยุด action, เก็บ audit event และส่งต่อ incident owner

