# Replay testing

นำ sanitized historical alert หรือ synthetic fixture เข้า ingestion แบบไม่ต่อ target จริง ตรวจว่า:

- normalization เหมือนเดิมเมื่อ schema version เดิม
- dedup/correlation ให้ผลตาม expected
- policy decision reproducible
- connector call ถูก mock และไม่เกิด external side effect
- evidence/audit output ครบ

เปลี่ยน rule, schema, policy หรือ connector เมื่อใดต้องรัน replay regression ที่เกี่ยวข้อง

