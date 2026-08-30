# Case model

case คือ aggregate ที่รวม alert, observables, tasks, decisions, evidence และ actions

## Lifecycle

```text
new → triage → investigating → contained → eradication → recovery
                                      ↘ dismissed
ทุกสถานะสามารถ → escalated → closed หลังมี closure reason
```

## Required case data

- case ID, tenant, title, category, severity, confidence
- opened/updated/closed time พร้อม timezone
- owner, incident commander และ service owner เมื่อเกี่ยวข้อง
- source alert IDs และ correlation key
- asset/user/identity scope
- hypothesis, facts, limitations และ current assessment
- task list และ dependency
- action/approval/audit references
- evidence manifest และ retention class
- closure reason, root cause และ lessons learned

