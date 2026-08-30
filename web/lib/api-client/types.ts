// Hand-written types mirroring api/openapi.yaml's M1 schemas (Case, Problem,
// the timeline entry shape). A full Orval/oapi-codegen pass across the whole
// 30-endpoint spec is deferred until more of it is actually implemented —
// see the M1 plan's note on scoped codegen. Keep these in sync with
// api/openapi.yaml and services/api/genapi/types.gen.go by hand for now.

export type Severity = "informational" | "low" | "medium" | "high" | "critical";

export type CaseStatus =
  | "new"
  | "triage"
  | "investigating"
  | "contained"
  | "eradication"
  | "recovery"
  | "escalated"
  | "dismissed"
  | "closed";

export interface Case {
  id: string;
  tenant_id: string;
  title: string;
  category: string;
  severity: Severity;
  confidence: number;
  status: CaseStatus;
  opened_at: string;
  updated_at?: string;
  closed_at?: string;
  owner: string;
  incident_commander?: string;
  service_owner?: string;
  closure_reason?: string;
  hypothesis?: string;
  facts?: string[];
  limitations?: string[];
  correlation_key?: string;
  root_cause?: string;
  lessons_learned?: string;
  observables?: CaseObservable[];
}

export interface CaseObservable {
  type: string;
  value: string;
}

export interface CasePage {
  items: Case[];
  next_cursor?: string;
}

export interface CaseTimelineEntry {
  id: number;
  occurred_at: string;
  kind: string;
  actor: string;
  detail?: Record<string, unknown>;
}

export interface Problem {
  type: string;
  title: string;
  status: number;
  code: string;
  request_id: string;
  retryable: boolean;
  detail?: string;
}
