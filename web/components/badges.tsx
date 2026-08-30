import type { CaseObservableVerdict, CaseStatus, Severity } from "@/lib/api-client/types";

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
