# Metrics and KPIs

| Metric | ความหมาย | ใช้ระวังอะไร |
|---|---|---|
| MTTT | เวลา alert ถึง triage | แยกตาม severity/source |
| MTTC | เวลา alert ถึง containment | นับเฉพาะ verified containment |
| workflow success | run จบตาม expected post-condition | ไม่ใช่แค่ HTTP 200 |
| automation coverage | สัดส่วนงานที่ลด manual step | ไม่ควรเพิ่มโดยลดคุณภาพ |
| false-positive harm | action ผิดที่เกิดจาก automation | metric ความปลอดภัยหลัก |
| analyst override | คนแก้/ปฏิเสธผลระบบ | ใช้ปรับ rule/playbook |
| evidence completeness | case ที่มี artifact/chain ครบ | สำคัญต่อ DFIR |
| connector reliability | success/timeout/error ต่อ provider | ใช้ตัดสิน circuit breaker |

ต้องรายงานทั้ง speed, quality, safety และ business impact ไม่ใช่ speed อย่างเดียว

