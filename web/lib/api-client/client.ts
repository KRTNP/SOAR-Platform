import "server-only";

import type { Case, CasePage, CaseTimelineEntry, CaseStatus, Problem } from "./types";

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
