# Milestones

| Milestone | ผลลัพธ์ | Gate | Status |
|---|---|---|---|
| M0 | design repository พร้อมเอกสาร/diagram/schema | link/schema validation | Done |
| M1 | synthetic alert เข้า case ได้ | ingestion/replay tests | Done — [`services/api`](../../services/api) (Go/Gin/pgx, no auth/NATS/Temporal yet) + [`web`](../../web) (Next.js Alert Queue and Case Detail); gate proven by `services/api/integration/ingestion_test.go` (Testcontainers) |
| M2 | enrichment และ ticket mock | connector contract tests | Not started |
| M3 | approval/deny/expiry | no-action-before-approval test | Not started |
| M4 | evidence/audit export | hash/redaction tests | Not started |
| M5 | lab connector | sandbox + tabletop approval | Not started |
| M6 | controlled production pilot | owner sign-off, rollback, KPI baseline | Not started |

M1's scope and explicit simplifications (no auth stub beyond a fixed tenant, synchronous single-transaction ingestion instead of NATS/Temporal, trivial 1:1 event→alert→case) are recorded in the implementation plan this session produced; see the M1 code's package-level doc comments (`services/api/main.go`, `services/api/events/event.go`) for the same rationale inline.

