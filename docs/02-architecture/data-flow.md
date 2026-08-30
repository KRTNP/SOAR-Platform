# Data flow

## Alert-to-response flow

1. Source ส่ง signed webhook หรือ authenticated API request
2. Ingestion ตรวจขนาด, content type, timestamp, source และ tenant
3. Raw payload ถูก redacted และเก็บ hash/reference
4. Normalizer สร้าง canonical event และ observable
5. Correlator ค้น duplicate/campaign ภายใน time window
6. Case service สร้างหรือ update case
7. Workflow engine เรียก investigation connectors แบบ parallel ที่ปลอดภัย
8. Policy engine คำนวณ severity, confidence, blast radius และ required approval
9. Response connector ทำงานเฉพาะเมื่อ policy อนุญาต
10. Verification ตรวจ post-condition และบันทึกผล
11. Evidence/audit เก็บทุก transition และส่ง metrics

## Failure flow

```text
connector error → classify transient/permanent/policy → retry if bounded
               → dead-letter if exhausted
               → create task + notify owner
               → never mark success without post-condition
```

