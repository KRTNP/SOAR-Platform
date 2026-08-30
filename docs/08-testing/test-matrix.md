# Test matrix

| Area | Happy path | Negative path | Evidence |
|---|---|---|---|
| Ingestion | valid signed event | replay/oversize/unknown tenant | request/audit ID |
| Normalization | complete Wazuh event | missing timezone/unknown field | mapping result |
| Dedup | same event once | duplicate storm | idempotency record |
| Enrichment | provider verdict | timeout/rate limit | provider response/status |
| Policy | allow low-risk | deny out-of-scope | policy decision |
| Approval | approve exact hash | reject/expire/stale | approval record |
| Response | sandbox target changes | postcondition mismatch | action + verification |
| Evidence | hash matches | mismatch/unavailable | custody event |
| Recovery | worker resumes | queue/store outage | recovery log |
| AI assist | schema-valid suggestion | injection/unsafe tool request | prompt/output audit |

