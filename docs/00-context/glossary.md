# Glossary

| Term | ความหมาย |
|---|---|
| Event | เหตุการณ์ดิบจาก sensor หรือระบบธุรกิจ |
| Alert | ผลจาก rule/anomaly ที่บอกว่าควรตรวจสอบ |
| Incident | Case ที่ถูก escalate ผ่าน threshold ที่กำหนด (status `escalated` ใน case lifecycle) ไม่ใช่ schema entity แยกต่างหาก; "incident" ที่หมายถึงตัวแพลตฟอร์ม SOAR เองล่มเป็นความหมายคนละบริบท ดู [incident management](../07-operations/incident-management.md) |
| Case | record กลางที่รวม alert, observable, evidence, task และ decision |
| Observable | สิ่งที่ตรวจสอบได้ เช่น IP, domain, hash, user, host |
| Enrichment | การเติม context จากแหล่งข้อมูลอื่น |
| Playbook | workflow ที่กำหนดขั้นตอนตรวจสอบ/ตอบสนอง |
| Connector | adapter ที่เรียก API หรือเครื่องมือภายนอก |
| Intent | ความสามารถหนึ่งอย่างที่ connector ประกาศไว้ล่วงหน้า (เช่น `get_alert`, `health`) ที่ playbook step เรียกใช้ได้ |
| Action | การเรียกใช้ intent หนึ่งครั้งพร้อมผลลัพธ์จริงของ connector |
| Idempotency | ส่งคำสั่งซ้ำแล้วผลลัพธ์ไม่เปลี่ยนซ้ำจนเกิดผลเสีย |
| Evidence | artifact หรือ metadata ที่ใช้สนับสนุนข้อสรุป |
| Dry run | ประเมินคำสั่งโดยไม่เปลี่ยนระบบปลายทาง |
| HITL | Human-in-the-loop จุดที่คนต้องอนุมัติ/ปฏิเสธ |
| Blast radius | ขอบเขตผลกระทบหาก action ผิดพลาด |
| TTP | tactic, technique และ procedure ของผู้โจมตี |

