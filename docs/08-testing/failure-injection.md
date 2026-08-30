# Failure injection

จำลอง failure ต่อไปนี้ใน lab:

- queue unavailable
- worker crash หลัง provider call ก่อนบันทึกผล
- provider timeout/rate-limit
- duplicate webhook
- stale approval
- target เปลี่ยน state ก่อน action
- evidence store unavailable
- policy service unavailable

เกณฑ์ผ่านคือ system fail closed, ไม่ทำ action ซ้ำโดยไม่ตั้งใจ, แจ้ง operator และเก็บ recovery context ได้

