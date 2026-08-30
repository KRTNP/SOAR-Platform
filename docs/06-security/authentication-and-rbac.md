# Authentication and RBAC

## Roles

| Role | Read case | Run investigation | Approve response | Manage platform |
|---|---:|---:|---:|---:|
| Tier-1 | scoped | yes | no | no |
| Tier-2 | scoped | yes | limited | no |
| Incident commander | incident scope | yes | yes | no |
| DFIR analyst | evidence scope | collection | no by default | no |
| Detection engineer | metadata | test only | no | playbook draft |
| Platform operator | operational metadata | health only | no | yes |
| Auditor | read-only | no | no | no |

สิทธิ์จริงต้องประกอบด้วย tenant, case, asset และ action scope ไม่ใช่ role อย่างเดียว

## Authentication controls

- SSO/OIDC สำหรับผู้ใช้
- service identity แยกต่อ connector
- short-lived tokens เมื่อ provider รองรับ
- MFA สำหรับ approval และ break-glass
- session expiry และ re-authentication สำหรับ high-impact action
- machine-to-machine request signing

