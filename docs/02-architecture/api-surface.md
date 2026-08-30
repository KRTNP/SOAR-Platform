# API surface

## Resource groups

| Resource | Read | Write | High-risk |
|---|---|---|---|
| `/events` | scoped | ingest | no |
| `/alerts` | scoped | triage/update | no |
| `/cases` | scoped | lifecycle/task | no |
| `/observables` | scoped | record/enrich/annotate | no |
| `/schedules` | scoped | create/update/disable | scheduler dispatch |
| `/playbooks` | reviewable | validate/promote | promotion |
| `/executions` | scoped | run/cancel/replay | cancel/execute |
| `/approvals` | approver-scoped | approve/reject | yes |
| `/actions` | case-scoped | request/execute | yes |
| `/evidence` | case-scoped | manifest/export reference | no direct original write |
| `/connectors` | restricted read | health/test | connector operation |
| `/audit` | restricted read | append service-only | no delete |

## API rules

- ทุก request มี server-generated หรือ client-supplied `request_id`, authenticated principal และ tenant context
- mutation ใช้ idempotency key เมื่อ retry ได้
- pagination เป็น cursor-based และมี maximum page size
- error ใช้ stable error code แยก validation, authz, transient, provider และ policy
- API response ไม่คืน secret หรือ unredacted evidence โดย default
- action endpoint รับ intent schema เท่านั้น ไม่รับ provider command
