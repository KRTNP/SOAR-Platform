# Problem statement

SOC ต้องสลับไปมาระหว่าง SIEM, endpoint, firewall, identity, TI และ ticketing ทำให้เกิดความล่าช้าและข้อมูลหลุดระหว่าง handoff ปัญหาหลักไม่ใช่การขาดเครื่องมือ แต่คือไม่มี process และ data contract กลางที่เชื่อมเครื่องมือเหล่านั้น

## Failure modes ที่ต้องแก้

| Failure mode | ผลกระทบ | กลไกแก้ใน SOAR |
|---|---|---|
| Alert ซ้ำจำนวนมาก | analyst เสียเวลา | deduplication และ correlation |
| Field ของแต่ละ vendor ต่างกัน | playbook ใช้ซ้ำไม่ได้ | normalization layer |
| Enrichment ทำมือ | triage ช้า | reusable investigation actions |
| Action ไม่มีผู้อนุมัติ | false positive ทำระบบล่ม | risk tier และ approval gate |
| หลักฐานกระจัดกระจาย | สืบสวน/รายงานยาก | case/evidence model |
| Connector เสียเงียบ ๆ | response ไม่เกิด | health check, timeout และ alerting |
| Playbook แก้แล้วไม่มี version | ผลลัพธ์ทำซ้ำไม่ได้ | immutable revision และ test |

## Design hypothesis

หากเรามี common event model, playbook ที่มี stop condition และ approval, connector ที่มี contract ชัดเจน และ evidence/audit ที่ครบ จะลดเวลางานซ้ำโดยไม่เพิ่มความเสี่ยงจาก automation ที่ควบคุมไม่ได้

