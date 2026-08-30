# Approval model

## Decision inputs

- action risk
- asset criticality
- scope size/blast radius
- confidence and evidence quality
- maintenance window
- service owner policy
- current incident severity

## Approval states

`not_required → pending → approved → executing → verified`

ทางเลือกอื่นคือ `rejected`, `expired`, `cancelled`, `failed` และ `escalated`

Approval record ต้องผูกกับ exact action hash, target set hash, policy version, approver identity, reason, time, expiry และ execution ID เพื่อป้องกันการเปลี่ยนคำขอหลังอนุมัติ

กรณีฉุกเฉินต้องมี break-glass role แยกต่างหาก, เหตุผลบังคับ, TTL สั้น และ post-incident review

