# Safe response controls

ก่อน action ที่เปลี่ยนระบบต้องผ่าน gate ต่อไปนี้:

1. target อยู่ใน authorized scope
2. target identity ถูก resolve แบบไม่กำกวม
3. asset criticality และ business owner ถูกโหลดแล้ว
4. indicator/verdict มี confidence และ evidence เพียงพอ
5. allowlist/exception ไม่ห้าม action
6. blast radius อยู่ใน threshold
7. approval ตรงกับ exact action hash
8. TTL/expiry และ rollback ถูกกำหนด
9. connector health ผ่าน
10. post-condition verification พร้อม

หากข้อใดไม่ผ่าน ให้สร้าง escalation task และไม่ execute

