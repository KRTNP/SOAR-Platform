# Test strategy

แบ่ง test เป็น contract, unit, integration, replay, security, resilience และ human exercise

เครื่องมือหลักตาม design baseline คือ Go testing สำหรับ unit/เป็นฐานของ integration,
Testcontainers สำหรับ dependency integration, Playwright สำหรับ UI flow และ OpenAPI
contract tests สำหรับตรวจ client/server compatibility

หลักฐานทุก test ต้องระบุ input fixture, versions, expected outcome, actual outcome และ whether target state changed

## Automation status ปัจจุบัน

Go testing/Testcontainers/Playwright/OpenAPI contract test ข้างบนคือ target stack เมื่อเริ่มเขียน implementation (ตาม [ADR-0005](../../adr/0005-selected-software-stack.md)) — ยังไม่มีอยู่จริงในขั้น documentation-first นี้ สิ่งที่ runnable จริงตอนนี้คือ `validation/*.sh` 5 ไฟล์ (`validate-json.sh`, `validate-playbooks.sh`, `validate-diagrams.sh`, `validate-links.sh`, `validate-source-traceability.sh`) ซึ่งตรวจ required-field/enum ผิวเผินและ cross-reference ระหว่างเอกสาร ไม่ใช่ behavioral test ตาม test-matrix/workflow-testing/replay-testing/connector-testing/failure-injection ที่เอกสารชุดนี้อธิบาย — เอกสารเหล่านั้นยังเป็น design target รอ implementation เช่นเดียวกับ ADR ทุกฉบับ

production-like test ต้องใช้ sandbox/test tenant หรือ dry-run เท่านั้น การจำลอง attack ต้องอยู่ใน lab ที่ได้รับอนุญาต
