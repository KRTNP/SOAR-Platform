# Response decision matrix

ใช้ matrix นี้ก่อนเลือก automation level โดยต้องประเมินทั้ง evidence และผลกระทบ

| Evidence | Asset criticality | Blast radius | Default decision |
|---|---|---|---|
| low | any | any | investigate only |
| medium | low | one target | require analyst review |
| high | low | one target | gated response |
| high | medium | one target | owner + analyst approval |
| high | high | one target | incident commander + service owner approval |
| high | any | multiple targets | incident commander, staged containment |
| any | unknown | any | stop and resolve context |
| any | any | irreversible | manual approval and rollback plan |

## Action defaults

| Action | Default level | Required verification |
|---|---:|---|
| create ticket/notify | L2 | ticket ID/message ID |
| query/enrich | L1 | result count/provider status |
| quarantine email | L3 | message state and affected recipients |
| revoke session/token | L3 | active session state |
| block indicator | L4 | rule ID, TTL and traffic state |
| isolate endpoint | L4 | endpoint isolation state |
| disable account | L4 | account state and owner confirmation |
| delete/overwrite evidence | L5 prohibited | manual forensic procedure only |

## Mandatory deny conditions

- target outside scope
- exact target identity unresolved
- approval expired or action hash changed
- policy service unavailable
- evidence integrity mismatch
- post-condition cannot be checked
- blast radius exceeds policy threshold

