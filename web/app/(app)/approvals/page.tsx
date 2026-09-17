import Link from "next/link";
import { listApprovals, getAction, ApiProblemError } from "@/lib/api-client/client";
import type { Approval, PendingAction } from "@/lib/api-client/types";
import { ApprovalStatusBadge, ModeBadge, SeverityBadge } from "@/components/badges";
import { EmptyState } from "@/components/EmptyState";
import { ProblemBanner } from "@/components/ProblemBanner";
import { ApprovalCard } from "./approval-card";
import { ApprovalRowActions } from "./approval-actions";
import { formatExpiry, truncateCaseId } from "./format";

export const dynamic = "force-dynamic";

// Decided approvals are reference material, not work. Bounded so one page
// render cannot fan out one getAction per historical row.
const DECIDED_LIMIT = 25;

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

  // The Approval schema deliberately omits intent/target/risk, so each row
  // costs its own getAction round trip. Pending rows all get one — they are
  // the actionable set and there are never many. Decided history is bounded:
  // it is a reference list, and resolving every historical approval would
  // turn one page render into an unbounded fan-out against an 8s-per-request
  // budget.
  const pendingApprovals = page.items.filter((a) => a.status === "pending");
  const decidedApprovals = page.items.filter((a) => a.status !== "pending");
  const decidedShown = decidedApprovals.slice(0, DECIDED_LIMIT);

  const resolve = (approval: Approval): Promise<ApprovalRow> =>
    getAction(approval.action_id)
      .then((action) => ({ approval, action }))
      .catch(() => ({ approval }));

  const [pending, decided] = await Promise.all([
    Promise.all(pendingApprovals.map(resolve)),
    Promise.all(decidedShown.map(resolve)),
  ]);

  const decidedHidden = decidedApprovals.length - decidedShown.length;

  return (
    <div className="flex flex-col gap-6">
      {/* One statement of the counts, not a title pill plus a metric bar
          repeating the same two numbers three feet apart. */}
      <header className="flex flex-col gap-1 border-b border-border pb-4">
        <h1 className="text-2xl font-semibold tracking-tight text-text-primary">Approvals</h1>
        <p className="text-xs text-text-tertiary">
          {pending.length > 0 ? (
            <span className="font-medium text-warn">{pending.length} awaiting your decision</span>
          ) : (
            "Nothing awaiting your decision"
          )}{" "}
          &middot; {decidedHidden > 0 ? `${decided.length + decidedHidden} decided` : `${decided.length} decided`}
        </p>
      </header>

      {/* Main Content Area */}
      {page.items.length === 0 ? (
        <EmptyState
          title="No approvals yet"
          body="A response action that requires approval shows up here until it's approved, rejected, or expires."
          action={{ label: "Back to case queue", href: "/" }}
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
            <div className="flex flex-col gap-2">
              <ApprovalTable
                title="Decided History"
                rows={decided}
                badgeColor="bg-text-tertiary"
                showActions={false}
                isHistorical
              />
              {decidedHidden > 0 && (
                <p className="text-2xs text-text-tertiary">
                  Showing the {DECIDED_LIMIT} most recent decisions; {decidedHidden} older{" "}
                  {decidedHidden === 1 ? "one is" : "ones are"} not listed.
                </p>
              )}
            </div>
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
        <div className="flex items-center gap-2 text-xs font-semibold tracking-wider text-text-secondary uppercase">
          <span aria-hidden="true" className={`h-2 w-2 rounded-full ${badgeColor}`} />
          <h2 id={`table-title-${title.toLowerCase().replace(/\s+/g, "-")}`}>{title}</h2>
          <span className="rounded-full bg-surface-raised px-1.5 py-px font-mono text-2xs font-normal text-text-tertiary">
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
        <table className="w-full border-collapse text-left text-xs">
          <caption className="sr-only">{title} queue list</caption>
          <thead>
            <tr className="border-b border-border bg-surface-raised/60 text-2xs font-semibold tracking-wider text-text-tertiary uppercase">
              <th scope="col" className="px-3.5 py-2.5">Status</th>
              {!isHistorical && <th scope="col" className="px-3.5 py-2.5">Risk</th>}
              <th scope="col" className="px-3.5 py-2.5">Intent</th>
              <th scope="col" className="px-3.5 py-2.5">Target</th>
              <th scope="col" className="px-3.5 py-2.5">Case</th>
              {/* A decided row's useful facts are who decided and why — the
                  reason the operator was required to type. Its risk and its
                  expiry are spent. */}
              <th scope="col" className="px-3.5 py-2.5">{isHistorical ? "Decided" : "Requested By"}</th>
              <th scope="col" className="px-3.5 py-2.5">{isHistorical ? "Reason" : "Expires"}</th>
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
                {!isHistorical && (
                  <td className="px-3.5 py-2.5 whitespace-nowrap">
                    {action?.risk ? <SeverityBadge severity={action.risk} /> : <span className="text-text-tertiary">—</span>}
                  </td>
                )}
                <td className="max-w-[220px] px-3.5 py-2.5 text-text-secondary" title={action?.intent}>
                  {action ? (
                    <div className="flex flex-col items-start gap-1">
                      <span className="max-w-full truncate">{action.intent.replaceAll("_", " ")}</span>
                      <ModeBadge mode={action.mode} />
                    </div>
                  ) : (
                    "—"
                  )}
                </td>
                <td className="max-w-[220px] truncate px-3.5 py-2.5 font-mono text-2xs text-text-secondary" title={action?.target.value}>
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
                      className="font-mono text-2xs font-medium text-accent hover:underline focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent"
                      title={action.case_id}
                    >
                      {truncateCaseId(action.case_id)}
                    </Link>
                  ) : (
                    <span className="text-text-tertiary">—</span>
                  )}
                </td>
                {isHistorical ? (
                  <>
                    <td className="px-3.5 py-2.5 whitespace-nowrap text-text-secondary">
                      {approval.decided_at ? (
                        <span className="flex flex-col">
                          <span className="font-mono text-2xs text-text-tertiary">
                            {formatExpiry(approval.decided_at)}
                          </span>
                          {approval.approved_by && <span>{approval.approved_by}</span>}
                        </span>
                      ) : (
                        <span className="text-text-tertiary">—</span>
                      )}
                    </td>
                    <td className="max-w-[260px] px-3.5 py-2.5 text-text-secondary" title={approval.reason}>
                      {approval.reason ? (
                        <span className="line-clamp-2">{approval.reason}</span>
                      ) : (
                        <span className="text-text-tertiary">—</span>
                      )}
                    </td>
                  </>
                ) : (
                  <>
                    <td className="px-3.5 py-2.5 whitespace-nowrap text-text-secondary">{action?.requested_by ?? "—"}</td>
                    <td className="px-3.5 py-2.5 font-mono text-2xs whitespace-nowrap text-text-tertiary">
                      {approval.expires_at ? formatExpiry(approval.expires_at) : "—"}
                    </td>
                  </>
                )}
                {showActions && (
                  <td className="px-3.5 py-2.5 text-right whitespace-nowrap">
                    {action ? (
                      <ApprovalRowActions
                        approvalId={approval.id}
                        action={action}
                        expiresAt={approval.expires_at}
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