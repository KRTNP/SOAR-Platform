# Detection engineering and alert quality

SOAR จะขยายความเสียหายของ detection ที่ผิดพลาด ดังนั้น detection quality เป็น precondition ของ automation ไม่ใช่งานแยกภายหลัง

## Detection lifecycle

```text
hypothesis → data-source mapping → rule/query → historical replay
           → false-positive tuning → analyst review → production
           → outcome feedback → rule revision
```

ทุก detection ต้องระบุ:

- threat hypothesis และ ATT&CK technique ที่เกี่ยวข้อง
- data source, field mapping, retention และ expected latency
- severity/confidence logic
- known false positives และ exclusions
- test fixture/atomic test ที่ได้รับอนุญาตใน lab
- playbook mapping และ safe default
- owner, version, review date และ deprecation condition

ไม่ควร trigger destructive response จาก rule เดียวที่ยังไม่มี historical precision/recall หรือ analyst review

