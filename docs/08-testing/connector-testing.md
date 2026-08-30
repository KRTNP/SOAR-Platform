# Connector testing

ต่อ connector หนึ่งตัวให้ทดสอบ:

- valid/invalid auth
- permission denied
- malformed response
- timeout และ retry exhaustion
- provider rate-limit
- duplicate request ด้วย idempotency key เดิม
- dry-run ไม่เปลี่ยน target
- post-condition mismatch
- redaction ของ secret/error
- target outside allowlist

