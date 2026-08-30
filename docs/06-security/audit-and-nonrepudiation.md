# Audit and non-repudiation

ทุก state transition ต้องมี audit record ที่ประกอบด้วย:

`event_id, request_id, actor, actor_type, case_id, execution_id, action, target_hash, policy_version, connector_version, decision, reason, timestamp, result_hash`

Audit store ต้อง append-only, จำกัดผู้ลบ, มี retention, export และ integrity verification การแก้ไข case note ไม่ลบค่าก่อนหน้า แต่สร้าง revision ใหม่

