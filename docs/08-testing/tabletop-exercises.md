# Tabletop exercises

เริ่มจาก scenario ที่มีผลต่อการตัดสินใจจริง:

- phishing นำไปสู่ account compromise
- malware บน endpoint ที่เป็นระบบสำคัญ
- ransomware พร้อม third-party dependency
- actively exploited vulnerability
- SOAR/connector outage ระหว่าง incident

ผู้เข้าร่วมต้องครอบคลุม analyst, incident commander, IT/service owner, legal/communications, DFIR และ vendor ที่เกี่ยวข้อง ผลลัพธ์ต้องมี decision log, gap, owner, due date และ playbook revision

สำหรับ M5 ("lab connector") มี script ที่รันได้จริงกับ API และ OPNsense VM ของ lab แล้ว ดู
[`tabletop-m5-lab-connector.md`](tabletop-m5-lab-connector.md).

