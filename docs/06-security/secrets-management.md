# Secrets management

- secret อยู่ใน external secret manager ไม่อยู่ใน Git, YAML, database plaintext หรือ logs
- worker ขอ secret เฉพาะตอน execute และเฉพาะ connector/action ที่ต้องใช้
- redact token, cookie, password และ private key จาก input/output/error
- rotate credential ตาม owner และ expiry
- ตรวจ secret leakage ใน CI และ execution output
- credential ที่ใช้ containment ต้องมี scope, TTL และ revocation path

หาก connector ไม่รองรับ scoped credential ให้จัดเป็น high-risk integration และบังคับ approval ทุก action

