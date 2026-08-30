import type { Case, Severity } from "@/lib/api-client/types";
import { CaseCard } from "./CaseCard";

const SEVERITY_ORDER: Severity[] = ["critical", "high", "medium", "low", "informational"];

const DOT_CLASSES: Record<Severity, string> = {
  critical: "bg-severity-critical",
  high: "bg-severity-high",
  medium: "bg-severity-medium",
  low: "bg-severity-low",
  informational: "bg-severity-informational",
};

/** Groups cases by severity — chunking, not one flat list (see badges.tsx for the same principle applied to a single row). */
export function CaseListGroup({ cases }: { cases: Case[] }) {
  const groups = SEVERITY_ORDER.map((severity) => ({
    severity,
    items: cases.filter((c) => c.severity === severity),
  })).filter((g) => g.items.length > 0);

  return (
    <div className="flex flex-col gap-[18px]">
      {groups.map((group) => (
        <div key={group.severity}>
          <div className="mb-2 flex items-center gap-2 pl-0.5 text-[11.5px] font-semibold tracking-wider text-text-tertiary uppercase">
            <span className={`h-[7px] w-[7px] rounded-full ${DOT_CLASSES[group.severity]}`} />
            {group.severity}
            <span className="font-mono font-normal text-text-tertiary">{group.items.length}</span>
          </div>
          <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border">
            {group.items.map((item) => (
              <CaseCard key={item.id} item={item} />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
