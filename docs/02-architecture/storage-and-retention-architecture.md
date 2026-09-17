# Storage and retention architecture

The filesystem paths below are relative to the `soar-platform/` project root.

| Store | ข้อมูล | คุณสมบัติ |
|---|---|---|
| Domain DB | case, alert, task, decision, state | transactional, tenant-scoped |
| Queue | execution job และ retry metadata | durable, priority, dead-letter |
| Evidence store | lab: local filesystem; production: restricted object storage | original read-only, derived/export separated, hashed and access-controlled |
| Audit store | append-only action/approval trace | restricted write, tamper evidence |
| Search index | observable และ execution lookup | derived, rebuildable |
| Secret manager | credentials | no plaintext application access |

ห้ามใช้ search index เป็น source of truth ของ evidence หรือ approval และห้ามให้ evidence store อยู่ใน network zone เดียวกับ untrusted file parser. ใน lab ให้ใช้ `../dfir/` boundary ตาม `dfir/AGENTS.md` ของ workspace `dfir/` ซึ่งอยู่นอก repository นี้; object storage เป็น production option ไม่ใช่ lab requirement

## Retention flow

`active → review_due → legal_hold หรือ eligible_for_delete → deleted_with_audit`

การลบ derived data ต้องไม่ลบ audit/custody record ที่ policy กำหนดให้เก็บ
