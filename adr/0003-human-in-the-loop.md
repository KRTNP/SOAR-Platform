# ADR-0003: Human-in-the-loop for high-impact response

## Decision

block, isolate, disable, revoke privileged access และ action ที่ blast radius สูงต้องผ่าน approval ที่ผูกกับ exact action hash

## Rationale

งานวิจัยการใช้ SOAR พบความเสี่ยงจาก over-automation และคุณภาพ ticket ที่ลดลงเมื่อ automation ไม่ช่วยการตัดสินใจของ analyst

## Consequences

response บางอย่างช้าลง แต่ลด false-positive harm และทำให้การอนุมัติตรวจสอบย้อนหลังได้

