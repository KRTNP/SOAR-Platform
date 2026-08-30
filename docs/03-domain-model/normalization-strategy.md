# Normalization strategy

## Mapping pipeline

```text
vendor payload → source adapter → field mapping → type coercion
               → timestamp normalization → observable extraction
               → sensitivity labeling → canonical event
```

## Mapping rules

- เก็บ original source field name ไว้ใน mapping metadata
- timestamp ที่ไม่มี timezone ต้อง mark `timezone_missing` และไม่ใช้เป็นหลักฐานลำดับเหตุการณ์จนกว่าจะยืนยัน
- IP, domain, URL, hash และ account ต้อง normalize case/format อย่างชัดเจน
- ค่าที่ไม่รู้จักให้เก็บใน `extensions` ไม่ทิ้งเงียบ ๆ
- mapping ที่เปลี่ยนความหมายต้องเพิ่ม schema version
- mapping test ต้องมี fixture ทั้ง valid, missing, duplicate และ malicious input

OCSF เป็น reference สำหรับ taxonomy/security event; internal model อาจใช้ subset ที่เหมาะกับ operational workflow แต่ต้องบันทึก mapping และไม่อ้างว่าเป็น OCSF เต็มรูปแบบหากยังไม่ครบ

