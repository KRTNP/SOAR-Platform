# Incident management for the SOAR platform

"Incident" ในเอกสารนี้หมายถึงตัวแพลตฟอร์ม SOAR เอง (ingestion, workflow, connector) มีปัญหา — คนละความหมายกับ "Incident" ใน [glossary](../00-context/glossary.md) ที่หมายถึง escalated case ของฝั่ง security operations เอกสารนี้เป็น design-baseline runbook ยังไม่ใช่ operational SLA ที่ทดสอบจริง (ดู [on-call and SLO](on-call-and-slo.md) สำหรับสถานะ SLO ที่เป็น proposal)

หาก SOAR เองมีปัญหา ให้ใช้ manual fallback:

1. freeze high-impact automation
2. preserve platform audit/logs
3. notify incident commander และ connector owners
4. switch to approved manual runbooks
5. assess actions ที่ค้างหรือทำซ้ำ
6. reconcile external target state
7. restore จาก known-good version
8. review และปรับ playbook/controls

