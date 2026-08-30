# API design

The API is REST-first and described by [`api/openapi.yaml`](../../api/openapi.yaml).
The API is asynchronous for ingestion and workflow execution; a successful command
submission means the request was accepted, not that a provider action completed.

## Common rules

| Rule | Design |
|---|---|
| Base path | `/v1` |
| Authentication | OIDC bearer token issued by Keycloak |
| Authorization | RBAC plus tenant/team/case/asset/connector resource scope |
| Correlation | `X-Request-ID` accepted or generated; the server generates and propagates `X-Correlation-ID` for workflow/provider work |
| Mutations | `Idempotency-Key` required for retryable create/command operations |
| Pagination | Cursor-based; server maximum is 100 records |
| Time | RFC 3339 with UTC storage; source timezone is retained when available |
| Errors | `application/problem+json` with stable `code`, `request_id`, `retryable` and field errors |
| Secrets | Never returned; redact provider payloads and evidence metadata by policy |
| Async status | `202 Accepted` with execution/task reference for long-running work |
| Real-time | `GET /v1/stream` using SSE; reconnect with `Last-Event-ID` |

## Resource operations

| Resource | Endpoints | Important behavior |
|---|---|---|
| Events | `POST /events`, `GET /events` | Authenticated ingestion, deduplication, raw redaction and provenance |
| Alerts | `GET /alerts`, `POST /alerts/{id}/triage` | Triage is idempotent and scoped |
| Observables | `GET /observables`, `POST /observables` | Canonical IOC/entity values are scoped, deduplicated and provenance-linked |
| Cases | `POST /cases`, `GET /cases/{id}`, `PATCH /cases/{id}`, `GET /cases/{id}/timeline`, `GET/POST /cases/{id}/tasks`, `POST /cases/{id}/export` | Typed case operations cover update, transition, merge, split and close; analyst tasks and exports enforce lifecycle and owner scope |
| Schedules | `GET /schedules`, `POST /schedules`, `PATCH /schedules/{id}`, `DELETE /schedules/{id}` | Versioned playbook schedules are scoped; scheduler never bypasses policy or approval |
| Playbooks | `GET /playbooks`, `GET /playbooks/{id}`, `POST /playbooks/{id}/validate`, `POST /playbooks/{id}/promote` | Promotion checks lifecycle, reviewer, compatibility and risk diff; only validated versions can be executed |
| Executions | `POST /executions`, `GET /executions/{id}`, `POST /executions/{id}/pause`, `POST /executions/{id}/resume`, `POST /executions/{id}/cancel`, `POST /executions/{id}/replay` | Temporal-backed; cancellation and replay are audited |
| Approvals | `GET /approvals`, `POST /approvals/{id}/approve`, `POST /approvals/{id}/reject` | Approver scope, action hash and expiry are verified server-side |
| Actions | `POST /actions/preview`, `POST /actions/{id}/execute`, `GET /actions/{id}` | Preview is dry-run; execute requires policy and approval |
| Evidence | `POST /evidence/manifests`, `GET /evidence/{id}`, `GET /cases/{id}/evidence` | Original path is never writable through API |
| Connectors | `GET /connectors`, `GET /connectors/{id}/health`, `POST /connectors/{id}/test` | Capability and health are visible; credentials are not |
| Audit | `GET /audit` | Restricted read; append is service-only and no delete endpoint exists |

## First scenario API sequence

```text
Wazuh webhook → POST /v1/events
             ← 202 event_id, alert_id, request_id

Frontend → GET /v1/cases/{id}
         → GET /v1/stream

Workflow → POST /v1/approvals/{id}/approve
         → POST /v1/actions/{id}/execute
         ← 202 execution_id

Frontend → GET /v1/executions/{id}
         → SSE action.completed / action.rollback.completed
```

## Stable error taxonomy

`invalid_request`, `unauthenticated`, `forbidden`, `not_found`, `conflict`,
`idempotency_conflict`, `policy_denied`, `approval_required`, `approval_expired`,
`scope_violation`, `precondition_failed`, `provider_timeout`, `provider_rate_limited`,
`provider_unavailable`, `evidence_integrity_failure`, `workflow_paused`, and
`internal_error` are stable public codes. Provider-specific errors are mapped behind
these codes and preserved in restricted diagnostic metadata.
