# Automation levels

| Level | ลักษณะ | ตัวอย่าง | Control |
|---|---|---|---|
| L0 | manual checklist | analyst query SIEM | case task |
| L1 | assistive | enrich IOC, draft summary | analyst review |
| L2 | low-risk automated | create ticket, notify | allowlist + audit |
| L3 | gated response | revoke token, quarantine email | approval + post-condition |
| L4 | high-impact controlled | isolate host, block firewall | dual approval, TTL, rollback |
| L5 | prohibited autonomous | delete evidence, disable broad tenant | manual procedure only |

เลื่อนระดับได้เมื่อมีข้อมูล false positive, rollback, ownership, test coverage และธุรกิจยอมรับความเสี่ยงแล้ว

[ADR-0007](../../adr/0007-first-vertical-scenario.md) อนุมัติ single-approval (ไม่ใช่ dual) สำหรับ OPNsense temporary block ของ scenario แรก โดยชดเชยด้วย 10-minute TTL, post-condition verification ทั้งขาบล็อกและขา rollback, และ bounded retry — เป็นข้อยกเว้นที่บันทึกไว้ต่อ scenario นี้เท่านั้น ไม่ใช่ default ใหม่ของ L4

