# Functional requirements

## Ingestion

- รับ event ผ่าน webhook พร้อม request authentication
- รับ alert จาก REST API แบบ paginated
- รับ schedule สำหรับ enrichment หรือ health checks
- เก็บ raw payload แบบ redacted และ checksum
- รองรับ schema version และ source metadata

## Investigation

- ค้น asset, user, process, network, sign-in และ related case
- query TI โดยเก็บ provider, timestamp, verdict และ confidence
- correlate alert ตาม key และ time window
- สร้าง task ให้ analyst พร้อม evidence reference

## Case management

- สร้าง/รวม/แยก/ปิด case
- มี severity, status, owner, SLA, tags และ related observables
- เก็บ decision, approval, escalation และ communication
- export case manifest โดยไม่เปิดเผย secret

## Playbook execution

- trigger แบบ webhook, schedule, manual และ case event
- condition, loop, parallel branch, subflow และ pause/resume
- timeout, retry, dead-letter และ compensation
- dry-run และ replay ด้วย fixture
- หยุดเมื่อ policy, scope หรือ evidence precondition ไม่ผ่าน

## Response

- action มี risk class และ required permission
- high-impact action ต้องมี approval record
- verify post-condition หลัง action
- rollback หรือ compensating action เมื่อทำได้
- แสดงผลลัพธ์ให้ analyst ตรวจสอบก่อน close case

## Audit and reporting

- บันทึก execution graph และ input/output reference
- บันทึก actor, approver, connector version และ policy version
- สร้าง metrics และ report ตาม case/playbook/connector

