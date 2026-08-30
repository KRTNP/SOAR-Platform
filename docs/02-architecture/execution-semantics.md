# Workflow execution semantics

## Execution state

`queued → running → waiting_approval → resumed → succeeded`

terminal alternatives คือ `dismissed`, `failed`, `cancelled`, `expired` และ `dead_letter`

## Step semantics

- step มี immutable input reference และ output reference
- retry ได้เฉพาะ error class ที่ประกาศว่า transient
- retry ต้องใช้ idempotency key เดิม
- parallel branch ต้องประกาศว่ารวมผลแบบ all, any หรือ quorum
- approval pause ต้อง durable และ resume ได้หลัง worker restart
- cancel จะหยุด step ที่ยังไม่เริ่ม; step ที่ provider รับไปแล้วต้อง reconcile state
- workflow ห้ามรายงาน success หาก verification ไม่ผ่าน

## Concurrency

- ใช้ per-target lock กับ action ที่ชนกันได้
- deduplicate จาก `(tenant, case, intent, target, policy_version)`
- ป้องกัน stale approval ด้วย action hash และ expiry
- จำกัด fan-out ต่อ tenant/provider เพื่อป้องกัน quota exhaustion

