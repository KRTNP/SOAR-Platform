# Deployment model

## Lab baseline

- reverse proxy/API gateway
- ingestion and case API
- PostgreSQL สำหรับ domain state
- queue/broker สำหรับ workflow jobs
- worker pool แยกตาม connector risk/egress
- local filesystem สำหรับ lab evidence โดยแยก original/derived/export และ manifest
- secret manager
- metrics/log/trace backend

## Production boundary

วาง ingestion/API ใน network zone ที่รับจาก source ได้, worker investigation ใน egress-controlled zone, response worker ใน zone ที่แยก credential และ policy gate, และ evidence store ใน restricted zone

ไม่ให้ worker ที่ประมวลผล untrusted email/file มี network access ไปยัง control plane โดยตรง

## Environment separation

| Environment | Response policy | Data |
|---|---|---|
| dev | mock only | synthetic fixtures |
| lab | controlled target allowlist | real lab telemetry plus sanitized/replayed test data; no production data |
| staging | dry-run หรือ approval ทุก action | masked data |
| production | policy-driven | authorized operational data |
