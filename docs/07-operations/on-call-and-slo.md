# On-call and SLO

## Ownership

- platform operator: availability, queue, worker และ secrets
- connector owner: provider auth, quota, API behavior
- detection owner: rule quality, mapping และ false positive
- incident commander: high-impact decision
- DFIR owner: evidence/custody

## Suggested service objectives

| Service | Objective |
|---|---|
| Ingestion acceptance | 99.9% ของ authenticated requests ได้รับ durable ID |
| Case persistence | 99.9% ของ accepted alerts มี case/duplicate decision |
| Investigation execution | 99% ของ low-risk jobs จบหรือมี visible failure ภายใน policy timeout |
| Audit completeness | 100% ของ high-impact action มี approval/action/post-condition record |
| Recovery | restore control plane ตาม RPO/RTO ที่ owner อนุมัติ |

SLO เป็นข้อเสนอเริ่มต้น ต้อง calibrate จาก baseline จริงและไม่ควรแลก audit/evidence integrity กับ latency

