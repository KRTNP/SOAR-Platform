# Troubleshooting

| อาการ | ตรวจ | การตอบสนอง |
|---|---|---|
| alert ไม่เข้า | signature, gateway log, queue | retry source และเปิด ingestion incident |
| case ซ้ำ | idempotency key, correlation window | merge case และหยุด duplicate action |
| enrichment ช้า | provider latency/rate limit | bounded retry หรือ partial result |
| action success แต่ target ไม่เปลี่ยน | provider state/post-condition | mark failed และ escalate |
| approval ค้าง | owner, expiry, notification | escalate ไม่ auto-approve |
| evidence hash ไม่ตรง | source copy, transfer log | freeze case และ preserve copies |

