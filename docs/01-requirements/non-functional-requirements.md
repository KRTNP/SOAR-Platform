# Non-functional requirements

| Area | Requirement |
|---|---|
| Availability | ingestion และ case API ต้องแยกจาก worker เพื่อให้ queue ยังรับงานได้เมื่อ worker บางตัวล้ม |
| Reliability | ทุก external call มี timeout, bounded retry และ failure state ที่มองเห็นได้ |
| Security | default deny, least privilege, encrypted transport และ secret manager |
| Auditability | execution และ policy decision ต้องค้นย้อนหลังได้ตาม case/action/request ID |
| Integrity | evidence manifest และ audit record ตรวจ hash ได้ |
| Privacy | data minimization, field redaction, tenant boundary และ retention policy |
| Performance | enrichment หนึ่งรายการไม่ควรบล็อก workflow ทั้งหมด; ใช้ async fan-out เมื่อเหมาะสม |
| Recovery | backup/restore ต้องทดสอบได้ และไม่ทำลาย evidence ต้นฉบับ |
| Operability | health, queue lag, connector error และ playbook failure ต้องมี dashboard/alert |
| Portability | core model ไม่ผูกกับ vendor และ connector contract มี version |

