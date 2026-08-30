# Ticketing and communication integration

ticket ต้องมี case ID, severity, owner, SLA, summary, evidence links, action history และ closure reason

ห้ามส่ง raw credential, sensitive artifact หรือ full log ไปใน notification channel โดย default ให้ส่ง redacted summary และ link ที่มี authorization

การส่งแจ้งเตือนต้อง idempotent ด้วย case/event key เพื่อไม่สร้าง ticket ซ้ำเมื่อ webhook ถูก retry

