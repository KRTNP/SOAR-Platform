# Mission and scope

## Mission

สร้างระบบกลางที่รับ security signal จากหลายแหล่ง, รวมเหตุการณ์ที่เกี่ยวข้อง, เพิ่ม context, ประสานเครื่องมือ, ขอการตัดสินใจจากคนเมื่อจำเป็น และสร้าง audit/evidence ที่ตรวจสอบย้อนหลังได้

## In scope

- Webhook, REST API และ scheduled ingestion
- Alert normalization และ deduplication
- Case management และ observable tracking
- Threat-intelligence enrichment
- Playbook execution แบบ sync/async
- Approval, escalation และ pause/resume
- Connector framework พร้อม health check และ versioning
- Evidence metadata, hashes และ immutable audit trail
- Metrics, execution history, replay และ dry-run
- Wazuh, DFIR, ticketing และ generic HTTP เป็น baseline integrations

## Out of scope for the first release

- Autonomous destructive response
- การเก็บ raw log ปริมาณมหาศาลแทน SIEM/data lake
- การแทนที่ EDR, firewall, IAM หรือ forensic tool
- การฝึกโมเดล AI จาก client data โดยอัตโนมัติ
- การรัน offensive simulation ใน production

## Success definition

ระบบรุ่นแรกต้องรับ Wazuh alert จริง, normalize, deduplicate, enrich, เปิด case, ขออนุมัติ และบันทึกผลได้ใน lab โดยไม่ทำลายหลักฐานและสามารถ replay เหตุการณ์เดิมได้
