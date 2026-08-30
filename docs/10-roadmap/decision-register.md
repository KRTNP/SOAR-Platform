# Decision register

This register is the durable record of the decisions made during the SOAR design
conversation. It is the compact index; detailed contracts live in the linked design
documents. All entries are **accepted for lab design**, not permission to touch a
production system.

## Product and deployment decisions

| # | Topic | Decision |
|---:|---|---|
| 1 | MVP coverage | Cover alert triage, case management, automated response, DFIR/evidence and Wazuh integration in the baseline |
| 2 | Deployment | On-premise/self-hosted |
| 3 | Tenancy | Single-tenant; retain `tenant_id` for future extensibility |
| 4 | Lab packaging | Docker Compose |
| 5 | Architecture shape | Modular monolith plus separate worker and connector workers |
| 6 | Backend | Go; use Gin for the HTTP layer |
| 7 | Primary data/search | PostgreSQL source of truth plus OpenSearch derived search projection |
| 8 | Workflow | Temporal with Go SDK |
| 9 | Frontend | Next.js, React and TypeScript |
| 10 | Identity | Keycloak with OIDC/OAuth2 |
| 11 | Authorization | RBAC plus resource scope |
| 12 | API style | REST plus OpenAPI |
| 13 | Live updates | Server-Sent Events |
| 14 | Evidence bytes | Local filesystem for lab, separated original/derived/export paths |
| 15 | Secrets | HashiCorp Vault |
| 16 | Event transport | NATS JetStream |
| 17 | Observability | OpenTelemetry, Prometheus, Grafana and Loki |
| 18 | PostgreSQL access | pgx plus sqlc |
| 19 | Migration | Atlas |
| 20 | UI system | Tailwind CSS, shadcn/ui and Radix UI |
| 21 | Frontend state | TanStack Query plus Zustand |
| 22 | API generation | oapi-codegen for Go and Orval for TypeScript |
| 23 | Search indexing | Transactional outbox, NATS and idempotent indexer worker |
| 24 | Audit | Append-only PostgreSQL hash chain plus read-only export |
| 25 | Test stack | Go testing, Testcontainers, Playwright and OpenAPI contract tests |
| 26 | Integration policy | Use real providers when available; mock only unavailable providers |
| 27 | Wazuh source | Existing Wazuh instance; fixture remains for replay/negative tests |

## Lab topology and capacity decisions

| # | Topic | Decision |
|---:|---|---|
| 28 | Network | Isolated host-only/internal network; no bridged attacker/victim interface |
| 29 | Hypervisor | KVM/libvirt |
| 30 | Current machine | Ubuntu host, 2 CPU, 4 GB RAM and 50 GB storage is diagnostic-only, not full-stack capacity |
| 31 | Recommended host | 8 vCPU, 32 GB RAM and 500 GB SSD when Wazuh central is external |
| 32 | Victim order | Ubuntu Server victim first, Windows victim second; Kali is the authorized attacker VM |
| 33 | First scenario | SSH brute-force against Ubuntu victim |

## Response policy decisions

| # | Topic | Decision |
|---:|---|---|
| 34 | Automation tiers | Low-risk actions auto-run; high-impact actions require approval; unknown state fails closed |
| 35 | Low-risk automation | Create/update case, enrich IP/user/asset and notify analyst only |
| 36 | Lab approval | One authorized incident commander |
| 37 | First write action | Temporary source-IP block |
| 38 | Block duration | 10 minutes |
| 39 | Firewall | OPNsense lab VM |
| 40 | Service under test | SSH |
| 41 | Detection | 5 failed SSH logins within 1 minute from one source IP |
| 42 | Trusted sources | Management/admin IP allowlist; allowlist changes expire and are audited |
| 43 | Initial severity | Medium |
| 44 | Escalation | Raise priority for privileged user or critical asset |
| 45 | Analyst SLA | 15 minutes for Medium case review |
| 46 | OPNsense mechanism | Dedicated `SOAR_TEMP_BLOCK` alias and static firewall rule |
| 47 | Post-condition | Verify alias membership and actual blocked SSH traffic; verify rollback too |
| 48 | Rollback failure | Retry boundedly, retain block, create escalation; never remove all rules |
| 49 | Rollback retry | 3 attempts at 5, 15 and 30 seconds |
| 50 | Case closure | Block succeeded, cleanup/rollback verified, no repeated alert, analyst review recorded |
| 51 | Wazuh connector boundary | Two investigation-only manifests (`connector-wazuh-manager-lab`, `connector-wazuh-indexer-lab`) instead of one, per [ADR-0008](../../adr/0008-wazuh-manager-and-indexer-connector-split.md); no Wazuh write/active-response intent is exposed yet |

## Authority and change rule

The user delegated remaining design choices after decision 50 to the assistant. The
choices above therefore represent the selected recommendation and are recorded as
design defaults. Any change to response scope, target network, risk thresholds,
credentials, retention or write capability requires a new ADR and an updated lab test
before execution.

