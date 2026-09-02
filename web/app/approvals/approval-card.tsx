import Link from "next/link";
import type { Approval, PendingAction } from "@/lib/api-client/types";
import { ApprovalStatusBadge, SeverityBadge } from "@/components/badges";
import { ApprovalRowActions } from "./approval-actions";
import { formatExpiry } from "./format";

// Below md this replaces ApprovalTable's row with a stacked card — the same
// "two compositions, not one shrunk" split web/components/CaseCard.tsx
// already established. A table whose Approve/Reject sit in the rightmost of
// eight columns is exactly the "button that can't actually be pressed"
// failure on a narrow screen: reachable only by scrolling a nested
// overflow-x-auto container with no visible affordance to do so.
export function ApprovalCard({ approval, action, showActions }: { approval: Approval; action?: PendingAction; showActions: boolean }) {
  return (
    <div className="flex flex-col gap-2 border-b border-border bg-surface px-4 py-3 last:border-b-0">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-1.5">
          <ApprovalStatusBadge status={approval.status} />
          {action && <SeverityBadge severity={action.risk} />}
        </div>
        {action?.case_id && (
          <Link href={`/cases/${action.case_id}`} className="shrink-0 truncate font-mono text-[11px] text-accent hover:underline">
            {action.case_id}
          </Link>
        )}
      </div>
      <div className="flex flex-col gap-0.5">
        <span className="font-mono text-[13px] font-medium text-text-primary">{action?.intent ?? "—"}</span>
        <span className="truncate font-mono text-[11.5px] text-text-tertiary">{action?.target.value ?? "—"}</span>
      </div>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-text-tertiary">
        <span>by {action?.requested_by ?? "—"}</span>
        <span>expires {approval.expires_at ? formatExpiry(approval.expires_at) : "—"}</span>
      </div>
      {showActions && action && (
        <div className="mt-1">
          <ApprovalRowActions approvalId={approval.id} actionId={action.id} caseId={action.case_id} size="md" />
        </div>
      )}
    </div>
  );
}
