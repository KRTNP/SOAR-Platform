# Provider capability matrix

ก่อนเขียน connector ให้กรอก capability จาก provider documentation และทดสอบใน sandbox; ห้ามอนุมานจากชื่อ endpoint

| Capability | Wazuh | SIEM | EDR/XDR | Firewall | Identity | Mail | DFIR |
|---|---:|---:|---:|---:|---:|---:|---:|
| Webhook ingest | yes | varies | varies | varies | varies | varies | no |
| Bounded query | yes | yes | yes | yes | yes | yes | yes |
| Read-only health check | yes | yes | yes | yes | yes | yes | yes |
| Dry-run | n/a (no write intent) | adapter | provider-dependent | adapter | adapter | adapter | yes |
| Idempotent write | n/a (no write intent) | n/a | provider-dependent | provider-dependent | provider-dependent | provider-dependent | n/a |
| Post-condition API | yes | yes | yes | yes | yes | yes | yes |
| Native rollback | n/a (no write intent) | n/a | varies | varies | varies | varies | n/a |
| TTL/expiry | varies | n/a | varies | varies | varies | varies | n/a |

`yes` ในตารางเป็นความสามารถที่ design ต้องการ ไม่ใช่การรับรองว่า provider ทุกตัวรองรับโดย native; connector ต้องประกาศความจริงพร้อม version และ test evidence

