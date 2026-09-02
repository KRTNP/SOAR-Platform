import Link from "next/link";
import { listApprovals, getAction, ApiProblemError } from "@/lib/api-client/client";
import type { Approval, PendingAction } from "@/lib/api-client/types";
import { ApprovalStatusBadge, SeverityBadge } from "@/components/badges";
import { EmptyState } from "@/components/EmptyState";
import { ProblemBanner } from "@/components/ProblemBanner";
import { ApprovalCard } from "./approval-card";
import { ApprovalRowActions } from "./approval-actions";
import { formatExpiry, truncateCaseId } from "./format";

export const dynamic = "force-dynamic";

interface ApprovalRow {
  approval: Approval;
  action?: PendingAction;
}

export default async function ApprovalsPage() {
  let page;
  try {
    page = await listApprovals();
  } catch (err) {
    if (err instanceof ApiProblemError) {
      return <ProblemBanner problem={err.problem} fallbackTitle="Could not load approvals" />;
    }
    throw err;
  }

  // The Approval schema deliberately omits intent/target/risk (see the type's
  // comment); resolve each one's action separately. A single action lookup
  // failing (e.g. it was since deleted) drops that row's action detail
  // rather than breaking the whole queue.
  const rows: ApprovalRow[] = await Promise.all(
    page.items.map(async (approval) => {
      try {
        return { approval, action: await getAction(approval.action_id) };
      } catch {
        return { approval };
      }
    }),
  );

  const pending = rows.filter((r) => r.approval.status === "pending");
  const decided = rows.filter((r) => r.approval.status !== "pending");

  return (
    <div className="flex flex-col gap-[22px]">
      <header>
        <h1 className="text-xl font-semibold tracking-tight">Approvals</h1>
        <p className="text-xs text-text-tertiary">
          {pending.length} pending, {decided.length} decided &middot; approve or reject inline, or open the case for full context
        </p>
      </header>

      {rows.length === 0 ? (
        <EmptyState
          title="No approvals yet"
          body="A response action that requires approval shows up here until it's approved, rejected, or expires."
        />
      ) : (
        <div className="flex flex-col gap-[18px]">
          {pending.length > 0 && <ApprovalTable title="Pending" rows={pending} />}
          {decided.length > 0 && <ApprovalTable title="Decided" rows={decided} muted />}
        </div>
      )}
    </div>
  );
}

function ApprovalTable({ title, rows, muted }: { title: string; rows: ApprovalRow[]; muted?: boolean }) {
  return (
    <div>
      <div className="mb-2 flex items-center gap-2 pl-0.5 text-[11.5px] font-semibold tracking-wider text-text-tertiary uppercase">
        <span className="h-[7px] w-[7px] rounded-full bg-text-tertiary" />
        {title}
        <span className="font-mono font-normal text-text-tertiary">{rows.length}</span>
      </div>
      <div className={`flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border md:hidden ${muted ? "opacity-60" : ""}`}>
        {rows.map(({ approval, action }) => (
          <ApprovalCard key={approval.id} approval={approval} action={action} showActions={!muted} />
        ))}
      </div>

      <div className={`hidden overflow-x-auto rounded-md border border-border md:block ${muted ? "opacity-60" : ""}`}>
        <table className={`w-full border-collapse text-[12.5px] ${muted ? "min-w-[860px]" : "min-w-[980px]"}`}>
          <thead>
            <tr className="border-b border-border bg-surface text-left text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">
              <th className="px-3 py-2.5 font-semibold">Status</th>
              <th className="px-3 py-2.5 font-semibold">Risk</th>
              <th className="px-3 py-2.5 font-semibold">Intent</th>
              <th className="px-3 py-2.5 font-semibold">Target</th>
              <th className="px-3 py-2.5 font-semibold">Case</th>
              <th className="px-3 py-2.5 font-semibold">Requested by</th>
              <th className="px-3 py-2.5 font-semibold">Expires</th>
              {!muted && <th className="px-3 py-2.5 font-semibold">Decide</th>}
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {rows.map(({ approval, action }) => (
              <tr key={approval.id} className="bg-surface hover:bg-surface-hover">
                <td className="px-3 py-2.5">
                  <ApprovalStatusBadge status={approval.status} />
                </td>
                <td className="px-3 py-2.5">
                  {action ? <SeverityBadge severity={action.risk} /> : <span className="text-text-tertiary">—</span>}
                </td>
                <td className="px-3 py-2.5 font-mono text-text-primary whitespace-nowrap">{action?.intent ?? "—"}</td>
                <td className="px-3 py-2.5 font-mono text-text-secondary whitespace-nowrap">{action?.target.value ?? "—"}</td>
                <td className="px-3 py-2.5 whitespace-nowrap">
                  {action?.case_id ? (
                    <Link href={`/cases/${action.case_id}`} className="font-mono text-accent hover:underline" title={action.case_id}>
                      {truncateCaseId(action.case_id)}
                    </Link>
                  ) : (
                    <span className="text-text-tertiary">—</span>
                  )}
                </td>
                <td className="px-3 py-2.5 text-text-secondary whitespace-nowrap">{action?.requested_by ?? "—"}</td>
                <td className="px-3 py-2.5 font-mono text-text-tertiary whitespace-nowrap">
                  {approval.expires_at ? formatExpiry(approval.expires_at) : "—"}
                </td>
                {!muted && (
                  <td className="px-3 py-2.5">
                    {action ? (
                      <ApprovalRowActions approvalId={approval.id} actionId={action.id} caseId={action.case_id} />
                    ) : (
                      <span className="text-text-tertiary">—</span>
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
