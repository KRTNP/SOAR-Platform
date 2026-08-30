# Playbook lifecycle and versioning

```text
draft → review → lab_tested → staged → approved → active
  ↘ rejected        ↘ deprecated ← superseded
```

ทุก revision ต้องมี immutable ID, author, created/modified timestamp, change summary, test result, policy compatibility และ rollback revision

การ promote ไป environment ถัดไปต้องตรวจ:

- schema compatibility
- connector version
- permission diff
- target scope diff
- new action risk
- fixture replay result
- approval from owner and security reviewer

หากเปลี่ยนความหมายของ action หรือเพิ่ม blast radius ให้ถือเป็น breaking change แม้ YAML จะยัง parse ได้

