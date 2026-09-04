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

  // The Approval schema deliberately omits intent/target/risk;
  // resolve each one's action separately.
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
    <div className="flex flex-col gap-6">
      {/* Page Header with Key Metrics */}
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between border-b border-border pb-4">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2.5">
            <h1 className="text-xl font-semibold tracking-tight text-text-primary">Approvals</h1>
            {pending.length > 0 ? (
              <span className="inline-flex items-center rounded-full bg-accent/15 px-2 py-0.5 text-[11px] font-medium text-accent">
                {pending.length} need review
              </span>
            ) : (
              <span className="inline-flex items-center rounded-full bg-surface-raised px-2 py-0.5 text-[11px] font-medium text-text-tertiary">
                All caught up
              </span>
            )}
          </div>
          <p className="text-xs text-text-tertiary">
            Review and decide on pending actions inline, or drill into each case for full audit context.
          </p>
        </div>

        {/* Quick Summary Pill Bar */}
        <div className="flex items-center gap-2 text-xs">
          <div className="flex items-center gap-1.5 rounded-md border border-border bg-surface px-2.5 py-1 text-text-secondary">
            <span className="font-semibold text-text-primary">{pending.length}</span> Pending
          </div>
          <div className="flex items-center gap-1.5 rounded-md border border-border bg-surface px-2.5 py-1 text-text-secondary">
            <span className="font-semibold text-text-primary">{decided.length}</span> Decided
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      {rows.length === 0 ? (
        <EmptyState
          title="No approvals yet"
          body="A response action that requires approval shows up here until it's approved, rejected, or expires."
        />
      ) : (
        <main className="flex flex-col gap-8">
          {pending.length > 0 && (
            <ApprovalTable
              title="Pending Approvals"
              rows={pending}
              badgeColor="bg-accent"
              showActions
            />
          )}

          {decided.length > 0 && (
            <ApprovalTable
              title="Decided History"
              rows={decided}
              badgeColor="bg-text-tertiary"
              showActions={false}
              isHistorical
            />
          )}
        </main>
      )}
    </div>
  );
}

interface ApprovalTableProps {
  title: string;
  rows: ApprovalRow[];
  badgeColor: string;
  showActions: boolean;
  isHistorical?: boolean;
}

function ApprovalTable({ title, rows, badgeColor, showActions, isHistorical }: ApprovalTableProps) {
  return (
    <section aria-labelledby={`table-title-${title.toLowerCase().replace(/\s+/g, "-")}`} className="flex flex-col gap-2.5">
      {/* Section Header */}
      <div className="flex items-center justify-between pl-0.5">
        <div className="flex items-center gap-2 text-[12px] font-semibold tracking-wider text-text-secondary uppercase">
          <span aria-hidden="true" className={`h-2 w-2 rounded-full ${badgeColor}`} />
          <h2 id={`table-title-${title.toLowerCase().replace(/\s+/g, "-")}`}>{title}</h2>
          <span className="rounded-full bg-surface-raised px-1.5 py-0.2 font-mono text-[11px] font-normal text-text-tertiary">
            {rows.length}
          </span>
        </div>
      </div>

      {/* Mobile Stacked Cards (Visible below md) */}
      <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border bg-surface md:hidden shadow-xs">
        {rows.map(({ approval, action }) => (
          <ApprovalCard
            key={approval.id}
            approval={approval}
            action={action}
            showActions={showActions}
          />
        ))}
      </div>

      {/* Desktop Responsive Table (Visible on md and above) */}
      <div className="hidden overflow-x-auto rounded-md border border-border bg-surface shadow-xs md:block">
        <table className="w-full border-collapse text-left text-[12.5px]">
          <caption className="sr-only">{title} queue list</caption>
          <thead>
            <tr className="border-b border-border bg-surface-raised/60 text-[11px] font-semibold tracking-wider text-text-tertiary uppercase">
              <th scope="col" className="px-3.5 py-2.5">Status</th>
              <th scope="col" className="px-3.5 py-2.5">Risk</th>
              <th scope="col" className="px-3.5 py-2.5">Intent</th>
              <th scope="col" className="px-3.5 py-2.5">Target</th>
              <th scope="col" className="px-3.5 py-2.5">Case</th>
              <th scope="col" className="px-3.5 py-2.5">Requested By</th>
              <th scope="col" className="px-3.5 py-2.5">Expires</th>
              {showActions && (
                <th scope="col" className="px-3.5 py-2.5 text-right">
                  Decide
                </th>
              )}
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {rows.map(({ approval, action }) => (
              <tr
                key={approval.id}
                className={`transition-colors duration-150 ${isHistorical
                    ? "bg-surface/50 text-text-secondary hover:bg-surface-raised/40"
                    : "bg-surface hover:bg-surface-raised/60"
                  }`}
              >
                <td className="px-3.5 py-2.5 whitespace-nowrap">
                  <ApprovalStatusBadge status={approval.status} />
                </td>
                <td className="px-3.5 py-2.5 whitespace-nowrap">
                  {action?.risk ? (
                    <SeverityBadge severity={action.risk} />
                  ) : (
                    <span className="text-text-tertiary">—</span>
                  )}
                </td>
                <td className="max-w-[200px] truncate px-3.5 py-2.5 text-text-secondary" title={action?.intent}>
                  {action?.intent ? action.intent.replaceAll("_", " ") : "—"}
                </td>
                <td className="max-w-[220px] truncate px-3.5 py-2.5 font-mono text-[11.5px] text-text-secondary" title={action?.target.value}>
                  {action?.target.value ? (
                    <span className="rounded bg-surface-raised px-1.5 py-0.5 border border-border-strong/40">
                      {action.target.value}
                    </span>
                  ) : (
                    "—"
                  )}
                </td>
                <td className="px-3.5 py-2.5 whitespace-nowrap">
                  {action?.case_id ? (
                    <Link
                      href={`/cases/${action.case_id}`}
                      className="font-mono text-[11.5px] font-medium text-accent hover:underline focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent"
                      title={action.case_id}
                    >
                      {truncateCaseId(action.case_id)}
                    </Link>
                  ) : (
                    <span className="text-text-tertiary">—</span>
                  )}
                </td>
                <td className="px-3.5 py-2.5 text-text-secondary whitespace-nowrap">
                  {action?.requested_by ?? "—"}
                </td>
                <td className="px-3.5 py-2.5 font-mono text-[11.5px] text-text-tertiary whitespace-nowrap">
                  {approval.expires_at ? formatExpiry(approval.expires_at) : "—"}
                </td>
                {showActions && (
                  <td className="px-3.5 py-2.5 text-right whitespace-nowrap">
                    {action ? (
                      <ApprovalRowActions
                        approvalId={approval.id}
                        actionId={action.id}
                        caseId={action.case_id}
                        size="sm"
                      />
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
    </section>
  );
}