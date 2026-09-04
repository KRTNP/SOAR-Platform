import "server-only";

import { auth } from "@/lib/auth";
import type {
  Case,
  CasePage,
  CaseTimelineEntry,
  CaseStatus,
  EvidencePage,
  Problem,
  ReleaseAccepted,
  AuditPage,
  MetricsSnapshot,
  CaseExportRequest,
  CaseExportAccepted,
  PendingAction,
  ApprovalPage,
  SearchPage,
  Tenant,
  TenantPage,
  TrustedSourcePage,
  CriticalTargetPage,
} from "./types";

const API_BASE = process.env.API_BASE_URL ?? "http://localhost:8080";

/** Thrown when the API responds with an application/problem+json body. */
export class ApiProblemError extends Error {
  readonly problem: Problem;

  constructor(problem: Problem) {
    super(problem.title);
    this.problem = problem;
    this.name = "ApiProblemError";
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  // adr/0024 — forwards the signed-in user's Keycloak access token so the
  // Go API's auth.Verifier can authenticate the caller; session is null
  // (and this header omitted) whenever AUTH_ENABLE_REAL is off server-side,
  // matching that flag's own default-off, zero-behavior-change posture.
  const session = await auth();
  const headers: HeadersInit = { "Content-Type": "application/json", ...init?.headers };
  if (session?.accessToken) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${session.accessToken}`;
  }
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  });

  if (!res.ok) {
    const contentType = res.headers.get("content-type") ?? "";
    if (contentType.includes("application/problem+json")) {
      throw new ApiProblemError(await res.json());
    }
    throw new Error(`API request failed: ${res.status} ${res.statusText}`);
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

export function listCases(params?: { cursor?: string; limit?: number }): Promise<CasePage> {
  const query = new URLSearchParams();
  if (params?.cursor) query.set("cursor", params.cursor);
  if (params?.limit) query.set("limit", String(params.limit));
  const qs = query.toString();
  return request<CasePage>(`/v1/cases${qs ? `?${qs}` : ""}`);
}

export function getCase(caseId: string): Promise<Case> {
  return request<Case>(`/v1/cases/${encodeURIComponent(caseId)}`);
}

export function getCaseTimeline(caseId: string): Promise<{ items: CaseTimelineEntry[] }> {
  return request<{ items: CaseTimelineEntry[] }>(`/v1/cases/${encodeURIComponent(caseId)}/timeline`);
}

export function getCaseEvidence(caseId: string): Promise<EvidencePage> {
  return request<EvidencePage>(`/v1/cases/${encodeURIComponent(caseId)}/evidence`);
}

export function transitionCase(caseId: string, status: CaseStatus, reason?: string): Promise<Case> {
  return request<Case>(`/v1/cases/${encodeURIComponent(caseId)}`, {
    method: "PATCH",
    body: JSON.stringify({ operation: "transition", status, reason }),
  });
}

export function closeCase(caseId: string, closureReason: string): Promise<Case> {
  return request<Case>(`/v1/cases/${encodeURIComponent(caseId)}`, {
    method: "PATCH",
    body: JSON.stringify({ operation: "close", closure_reason: closureReason }),
  });
}

export function openTicket(caseId: string): Promise<Case> {
  return request<Case>(`/v1/cases/${encodeURIComponent(caseId)}`, {
    method: "PATCH",
    body: JSON.stringify({ operation: "open_ticket" }),
  });
}

export function requestBlock(caseId: string, targetValue: string): Promise<Case> {
  return request<Case>(`/v1/cases/${encodeURIComponent(caseId)}`, {
    method: "PATCH",
    body: JSON.stringify({ operation: "request_block", target_value: targetValue }),
  });
}

// approveAction and executeAction are two separate M3 API calls (the
// approval gate must be a real HTTP round trip an analyst cannot skip), but
// the UI only offers one "Approve" action — see approveActionAction in
// app/cases/[caseId]/actions.ts.
export function approveAction(approvalId: string): Promise<unknown> {
  return request(`/v1/approvals/${encodeURIComponent(approvalId)}/approve`, {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
  });
}

export function executeAction(actionId: string): Promise<unknown> {
  return request(`/v1/actions/${encodeURIComponent(actionId)}/execute`, {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
  });
}

// GET /v1/actions/{actionId} — used by the approvals queue page to resolve
// an Approval's intent/target/risk/case, which the Approval schema itself
// deliberately does not carry (see the Approval type's comment).
export function getAction(actionId: string): Promise<PendingAction> {
  return request<PendingAction>(`/v1/actions/${encodeURIComponent(actionId)}`);
}

// GET /v1/approvals — every approval visible to the tenant, pending first
// (services/api/actions.ListApprovals), newest first within each group.
export function listApprovals(): Promise<ApprovalPage> {
  return request<ApprovalPage>(`/v1/approvals`);
}

export function rejectApproval(approvalId: string, reason: string): Promise<unknown> {
  return request(`/v1/approvals/${encodeURIComponent(approvalId)}/reject`, {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
    body: JSON.stringify({ reason }),
  });
}

// M6's rollback endpoint (adr/0014) — same manual path the TTL scheduler
// itself uses automatically once block_ttl_seconds elapses; this lets an
// analyst release a still-executed block early instead of waiting on it.
export function releaseAction(actionId: string): Promise<ReleaseAccepted> {
  return request<ReleaseAccepted>(`/v1/actions/${encodeURIComponent(actionId)}/release`, {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
  });
}

// GET /v1/audit — restricted audit records, newest first. No auth/RBAC
// exists yet (see adr/0011), so this is not actually access-restricted in
// this lab build, only in the design intent it implements.
export function getAudit(): Promise<AuditPage> {
  return request<AuditPage>(`/v1/audit`);
}

// GET /v1/metrics — a point-in-time KPI snapshot (adr/0015), not a
// historical baseline or time series.
export function getMetrics(): Promise<MetricsSnapshot> {
  return request<MetricsSnapshot>(`/v1/metrics`);
}

// GET /v1/search — a cross-entity OpenSearch query (adr/0022), only mounted
// server-side when OPENSEARCH_ENABLE_REAL=true; a Problem here (e.g. 503)
// means the search indexer isn't running, not that q was invalid.
export function search(q: string): Promise<SearchPage> {
  return request<SearchPage>(`/v1/search?q=${encodeURIComponent(q)}`);
}

// GET/POST /v1/tenants (adr/0022). None of the three functions below need
// an Idempotency-Key header — unlike requestBlock/approveAction/etc., the
// backend handlers for tenants/allowlist/critical-targets don't require one.
export function listTenants(): Promise<TenantPage> {
  return request<TenantPage>(`/v1/tenants`);
}

export function createTenant(name: string): Promise<Tenant> {
  return request<Tenant>(`/v1/tenants`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export function deleteTenant(id: string): Promise<void> {
  return request<void>(`/v1/tenants/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function listAllowlist(): Promise<TrustedSourcePage> {
  return request<TrustedSourcePage>(`/v1/allowlist`);
}

export function addAllowlistEntry(value: string, reason: string, expiresAt?: string): Promise<unknown> {
  return request(`/v1/allowlist`, {
    method: "POST",
    body: JSON.stringify({ value, reason, expires_at: expiresAt || undefined }),
  });
}

export function deleteAllowlistEntry(id: string): Promise<void> {
  return request<void>(`/v1/allowlist/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function listCriticalTargets(): Promise<CriticalTargetPage> {
  return request<CriticalTargetPage>(`/v1/critical-targets`);
}

export function addCriticalTarget(value: string, reason: string): Promise<unknown> {
  return request(`/v1/critical-targets`, {
    method: "POST",
    body: JSON.stringify({ value, reason }),
  });
}

export function deleteCriticalTarget(id: string): Promise<void> {
  return request<void>(`/v1/critical-targets/${encodeURIComponent(id)}`, { method: "DELETE" });
}

// csv is spec'd (api/openapi.yaml) but not implemented server-side yet
// (services/api/cases/export.go rejects it with a Problem), so the request
// type here is narrowed to "json" only rather than offering a dead option.
export function exportCase(caseId: string, req: CaseExportRequest): Promise<CaseExportAccepted> {
  return request<CaseExportAccepted>(`/v1/cases/${encodeURIComponent(caseId)}/export`, {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
    body: JSON.stringify(req),
  });
}
