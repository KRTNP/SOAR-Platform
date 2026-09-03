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
  ticket_ref?: string;
  ticket_status?: TicketStatus;
  pending_action?: PendingAction;
}

export type ActionStatus = "pending_approval" | "approved" | "rejected" | "expired" | "executed" | "released";

export interface PendingAction {
  id: string;
  approval_id?: string;
  case_id: string;
  intent: string;
  target: { type: string; value: string };
  risk: "low" | "medium" | "high" | "critical";
  mode: "dry_run" | "execute";
  status: ActionStatus;
  requested_by: string;
  idempotency_key: string;
  expires_at?: string;
  reason?: string;
}

export type TicketStatus = "open" | "closed";

export type CaseObservableVerdict = "malicious" | "suspicious" | "benign" | "unknown";

export interface CaseObservable {
  type: string;
  value: string;
  verdict?: CaseObservableVerdict;
  ti_confidence?: number;
  source_reliability?: string;
}

export type IntegrityStatus = "verified" | "pending" | "mismatch" | "unavailable";

export interface Evidence {
  id: string;
  case_id: string;
  artifact_type: string;
  source: string;
  sha256: string;
  size: number;
  collected_at: string;
  integrity_status: IntegrityStatus;
  // Captured by services/api/evidence (schemas/evidence.schema.json) and
  // already returned by GET /v1/cases/{caseId}/evidence — previously
  // fetched but not typed or rendered here.
  tool_name?: string;
  tool_version?: string;
  analyst: string;
  purpose: string;
  sensitivity: "public" | "internal" | "confidential" | "restricted";
  retention_class: string;
  limitations?: string[];
}

export interface EvidencePage {
  items: Evidence[];
}

export interface CasePage {
  items: Case[];
  next_cursor?: string;
}

// GET /v1/search's result shape (adr/0022) — a thin pass-through of
// whatever OpenSearch document matched, not a typed domain object; "source"
// varies by which index ("soar-cases-v1" vs "soar-events-v1") the hit came
// from.
export interface SearchResult {
  index: string;
  id: string;
  source: Record<string, unknown>;
}

export interface SearchPage {
  items: SearchResult[];
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

export interface ReleaseAccepted {
  action_id: string;
  status: "released";
}

export type ApprovalStatus = "pending" | "approved" | "rejected" | "expired";

// Mirrors services/api/genapi.Approval's JSON shape (GET /v1/approvals,
// POST .../approve|reject) — deliberately thin (no intent/target/risk):
// those live on the linked Action, fetched separately via getAction.
export interface Approval {
  id: string;
  action_id: string;
  status: ApprovalStatus;
  approved_by?: string;
  reason?: string;
  decided_at?: string;
  expires_at: string;
}

export interface ApprovalPage {
  items: Approval[];
}

// Mirrors services/api/audit.Record's JSON shape (GET /v1/audit). Nullable
// fields are genuinely absent for many rows (e.g. an ingestion-time record
// has no case_id yet) rather than omitted.
export interface AuditRecord {
  id: string;
  event_id?: string | null;
  request_id?: string | null;
  actor: string;
  actor_type: string;
  case_id?: string | null;
  execution_id?: string | null;
  action: string;
  target_hash?: string | null;
  policy_version: string;
  connector_version?: string | null;
  decision: string;
  reason?: string | null;
  timestamp: string;
  result_hash: string;
  verified: boolean;
}

export interface AuditPage {
  items: AuditRecord[];
}

// Mirrors services/api/metrics.ConnectorOutcome / Snapshot (GET /v1/metrics)
// — a point-in-time read, not a historical baseline. See
// adr/0015-kpi-snapshot.md.
export interface ConnectorOutcome {
  connector_version: string;
  allowed: number;
  denied: number;
}

export interface MetricsSnapshot {
  computed_at: string;
  connector_target: string;
  connector_version: string;
  connector_reliability: ConnectorOutcome[];
  analyst_override_count: number;
  auto_release_count: number;
  manual_release_count: number;
  mean_time_to_containment_seconds?: number | null;
  cases_with_verified_evidence: number;
  total_cases: number;
}

export type RedactionProfile = "standard" | "full";

export interface CaseExportRequest {
  format: "json";
  include_evidence_references?: boolean;
  redaction_profile?: RedactionProfile;
  purpose?: string;
  recipient?: string;
}

export interface CaseExportAccepted {
  export_id: string;
  status: "accepted" | "completed" | "failed";
  manifest_sha256?: string;
  redaction_profile?: string;
  manifest?: Record<string, unknown>;
}
