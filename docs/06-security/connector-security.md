# Connector security

connector เป็น trust boundary และต้องผ่าน review ก่อนเปิดใช้งาน

## Review checklist

- input schema ไม่รับ arbitrary command/URL
- destination allowlist และ TLS verification
- auth scope ต่ำสุด
- timeout/retry/rate-limit ชัดเจน
- response ถูก validate และจำกัดขนาด
- error ไม่เผย secret
- idempotency และ duplicate behavior ชัดเจน
- health check ไม่เปลี่ยน state
- dry-run หรือ sandbox มีจริงเมื่อ action risk สูง
- version/pin ของ dependency และ image ตรวจได้

