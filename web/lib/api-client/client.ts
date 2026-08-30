import "server-only";

import type { Case, CasePage, CaseTimelineEntry, CaseStatus, EvidencePage, Problem } from "./types";

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
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
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

export function rejectApproval(approvalId: string, reason: string): Promise<unknown> {
  return request(`/v1/approvals/${encodeURIComponent(approvalId)}/reject`, {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
    body: JSON.stringify({ reason }),
  });
}
