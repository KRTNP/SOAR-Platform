import type { ApprovalStatus, CaseObservableVerdict, CaseStatus, IntegrityStatus, Severity } from "@/lib/api-client/types";

// Tailwind's JIT scanner needs literal class strings, not interpolated ones,
// so severity/status -> class is a lookup table rather than a template.
const SEVERITY_CLASSES: Record<Severity, string> = {
  critical: "text-severity-critical border-severity-critical/40 bg-severity-critical/[0.08]",
  high: "text-severity-high border-severity-high/40 bg-severity-high/[0.08]",
  medium: "text-severity-medium border-severity-medium/40 bg-severity-medium/[0.08]",
  low: "text-severity-low border-border-strong",
  informational: "text-severity-informational border-border-strong",
};

export function SeverityBadge({ severity, size = "sm" }: { severity: Severity; size?: "sm" | "lg" }) {
  return (
    <span
      className={`inline-block rounded-[3px] border font-medium whitespace-nowrap ${SEVERITY_CLASSES[severity]} ${
        size === "lg" ? "px-2.5 py-1 text-xs tracking-wide" : "px-2 py-[3px] text-[11px]"
      }`}
    >
      {size === "lg" ? severity.toUpperCase() : severity}
    </span>
  );
}

const STATUS_CLASSES: Partial<Record<CaseStatus, string>> = {
  investigating: "text-accent border-accent/40",
  triage: "text-warn border-warn/40",
  closed: "text-safe border-safe/35",
  dismissed: "text-text-tertiary border-border-strong",
};

export function StatusBadge({ status }: { status: CaseStatus }) {
  const cls = STATUS_CLASSES[status] ?? "text-text-secondary border-border-strong";
  return (
    <span className={`inline-block rounded-[3px] border px-2 py-[3px] text-[11px] font-medium whitespace-nowrap capitalize ${cls}`}>
      {status}
    </span>
  );
}

// Mock-TI verdict from ADR-0010's enrichment connector. "unknown" only ever
// appears for a type the mock provider covers but found no match for — a
// type it doesn't cover at all (user, host, ...) carries no verdict, so
// there is no badge to render for it.
const VERDICT_CLASSES: Record<CaseObservableVerdict, string> = {
  malicious: "text-danger border-danger/40 bg-danger/[0.08]",
  suspicious: "text-warn border-warn/40 bg-warn/[0.08]",
  benign: "text-safe border-safe/35",
  unknown: "text-text-tertiary border-border-strong",
};

export function VerdictBadge({ verdict }: { verdict: CaseObservableVerdict }) {
  return (
    <span className={`inline-block rounded-[3px] border px-2 py-[3px] text-[11px] font-medium whitespace-nowrap capitalize ${VERDICT_CLASSES[verdict]}`}>
      {verdict}
    </span>
  );
}

// M4's evidence.Get re-verifies the recorded hash on every read (ADR-0012);
// "mismatch" means that re-check just failed, not that it hasn't run yet.
const INTEGRITY_CLASSES: Record<IntegrityStatus, string> = {
  verified: "text-safe border-safe/35",
  pending: "text-text-tertiary border-border-strong",
  mismatch: "text-danger border-danger/40 bg-danger/[0.08]",
  unavailable: "text-warn border-warn/40",
};

export function IntegrityBadge({ status }: { status: IntegrityStatus }) {
  return (
    <span className={`inline-block rounded-[3px] border px-2 py-[3px] text-[11px] font-medium whitespace-nowrap capitalize ${INTEGRITY_CLASSES[status]}`}>
      {status}
    </span>
  );
}

// Audit records carry a free-text "decision" (services/api/audit's
// auditAppend callers use "allowed"/"denied"/"completed" today, per
// services/api/cases/handler.go and services/api/actions/handler.go) rather
// than a closed enum, so anything outside this vocabulary falls back to the
// same quiet plain-text treatment as the routine cases below.
//
// A full colored pill is reserved for "denied" — the one value an analyst
// scanning this log actually needs to catch. "allowed"/"completed" is what
// nearly every row says (it's the expected outcome), so giving it the same
// bordered/bg pill treatment as "denied" just paints most of the column
// green and drowns out the rows that matter; plain text carries it instead.
export function DecisionBadge({ decision }: { decision: string }) {
  if (decision === "denied") {
    return (
      <span className="inline-block rounded-[3px] border border-danger/40 bg-danger/[0.08] px-2 py-[3px] text-[11px] font-medium whitespace-nowrap text-danger capitalize">
        {decision}
      </span>
    );
  }
  return <span className="text-[11px] whitespace-nowrap text-text-tertiary capitalize">{decision}</span>;
}

// "pending" is genuinely actionable (it's why the row is in the Pending
// table at all) and "rejected" is the outcome an analyst reviewing history
// needs to spot, so both keep a full colored pill. "approved" is the
// majority outcome in Decided History — same quiet-by-default treatment as
// DecisionBadge/VerifiedBadge, a small check instead of a matching pill so
// it doesn't visually compete with "rejected". "expired" is background
// information, not a decision anyone made, so it stays plain muted text.
export function ApprovalStatusBadge({ status }: { status: ApprovalStatus }) {
  if (status === "approved") {
    return (
      <span className="inline-flex items-center gap-1 text-[11px] text-text-tertiary capitalize">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" className="h-3 w-3 text-safe">
          <polyline points="5 12.5 9.5 17 19 6.5" />
        </svg>
        {status}
      </span>
    );
  }
  if (status === "expired") {
    return <span className="text-[11px] whitespace-nowrap text-text-tertiary capitalize">{status}</span>;
  }
  const cls = status === "rejected" ? "text-danger border-danger/40 bg-danger/[0.08]" : "text-warn border-warn/40";
  return (
    <span className={`inline-block rounded-[3px] border px-2 py-[3px] text-[11px] font-medium whitespace-nowrap capitalize ${cls}`}>
      {status}
    </span>
  );
}

// Reflects audit.Record.Verified — a read-time recomputation of the row's
// own result_hash (services/api/audit's List), not a persisted flag. false
// means this specific row's hash no longer matches its stored content.
//
// "verified" is the expected outcome on every row (a real tamper event
// should be rare), so it gets a small quiet checkmark rather than a full
// bordered pill — the pill is reserved for "tampered", the one state an
// analyst scanning this column must never miss.
export function VerifiedBadge({ verified }: { verified: boolean }) {
  if (!verified) {
    return (
      <span className="inline-block rounded-[3px] border border-danger/40 bg-danger/[0.08] px-2 py-[3px] text-[11px] font-medium whitespace-nowrap text-danger">
        tampered
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 text-[11px] text-text-tertiary" title="Hash recomputed and matched on read">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" className="h-3 w-3 text-safe">
        <polyline points="5 12.5 9.5 17 19 6.5" />
      </svg>
      verified
    </span>
  );
}
