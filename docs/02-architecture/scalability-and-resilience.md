# Scalability and resilience

## Design targets

- ingestion เป็น stateless และ scale-out ได้
- queue แยก priority, tenant และ action risk
- enrichment fan-out มี concurrency limit ต่อ provider
- response action ใช้ serialized execution เมื่อ target เดียวกันเสี่ยงชนกัน
- case update ใช้ optimistic locking หรือ version check
- worker crash ต้อง resume จาก durable execution state

## Reliability controls

| Control | วิธีใช้ |
|---|---|
| Timeout | กำหนดต่อ connector และต่อ workflow step |
| Retry | exponential backoff, จำกัดจำนวนครั้ง, เฉพาะ transient error |
| Circuit breaker | หยุด provider ที่ล้มซ้ำและสร้าง operator alert |
| Dead letter | เก็บงานที่ retry หมดพร้อม reason |
| Idempotency | key จาก case/action/target/intent |
| Backpressure | จำกัด queue และ reject/slow low-priority input |
| Bulkhead | แยก worker pool ตาม connector/risk |
| Compensation | reverse หรือ mark manual recovery เมื่อ rollback ไม่ได้ |

