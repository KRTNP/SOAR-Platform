# Integration error taxonomy

| Error class | Retry | Case behavior |
|---|---:|---|
| validation | no | reject and audit |
| authentication | no/after credential rotation | disable connector and alert owner |
| authorization | no | escalate scope/permission |
| not_found | no | mark target unresolved |
| rate_limit | yes with backoff | keep pending and expose retry time |
| timeout | bounded | retry then partial/fail |
| provider_5xx | bounded | circuit breaker after threshold |
| postcondition_mismatch | no automatic repeat | mark failed and reconcile |
| policy_denied | no | record denied decision |
| evidence_integrity | no | freeze dependent workflow |

ห้ามใช้ HTTP 200 เพียงอย่างเดียวเป็น success; ต้องแยก transport, provider และ post-condition status

