# Event model

event คือ telemetry หรือ notification ที่เกิดขึ้นจาก source ส่วน alert คือผลการประเมิน event โดย rule/anomaly

## Canonical fields

| Field | ความหมาย |
|---|---|
| id | unique event ID จาก SOAR |
| source | vendor/system ที่สร้าง event |
| source_event_id | ID จาก source ถ้ามี |
| observed_at | เวลาเกิดเหตุใน UTC |
| received_at | เวลาที่ SOAR รับ |
| tenant_id | ขอบเขตเจ้าของข้อมูล |
| event_class | authentication, network, process, file, email, vulnerability ฯลฯ |
| severity | informational/low/medium/high/critical |
| confidence | 0.0 ถึง 1.0 พร้อม method |
| observables | references ไป IP/domain/hash/user/host |
| raw_ref | reference และ hash ของ payload ที่ redacted |
| labels | MITRE/Sigma/asset/risk labels |
| schema_version | version ของ canonical model |

Raw vendor fields เก็บใน `extensions` ได้ แต่ไม่ควรให้ core workflow อ้าง field vendor โดยตรง

