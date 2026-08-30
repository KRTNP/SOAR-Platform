# Evidence and custody

The filesystem paths below are relative to the `soar-platform/` project root.

SOAR เก็บ reference และ metadata เป็นหลัก ส่วน original evidence อยู่ในระบบที่กำหนดโดย DFIR policy

สำหรับ lab นี้ mapping ที่ใช้จริงคือ original `../dfir/evidence/<case>/`, derived
work `../dfir/cases/<case>/work/` และ export `../dfir/exports/<case>/`; SOAR จะไม่เขียน
ทับ evidence tree และจะเก็บ manifest/custody reference ใน domain store

ทุก transfer ต้องมี:

- from/to
- actor
- purpose
- timestamp/timezone
- source/destination hash
- authorization
- case ID

หาก hash mismatch, scope ไม่ชัด หรือ source กำลังเปลี่ยนแปลง ให้หยุด collection/response ที่พึ่ง evidence นั้นและ escalate
