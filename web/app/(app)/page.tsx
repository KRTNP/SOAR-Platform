import Link from "next/link";
import { listCases, listApprovals, listEvents, getAction, ApiProblemError } from "@/lib/api-client/client";
import type { PendingAction } from "@/lib/api-client/types";
import { CaseListGroup } from "@/components/CaseListGroup";
import { CaseCard } from "@/components/CaseCard";
import { EmptyState } from "@/components/EmptyState";
import { ProblemBanner } from "@/components/ProblemBanner";
import { ModeBadge, SeverityBadge } from "@/components/badges";
import { NewSinceLastVisit } from "@/components/NewSinceLastVisit";
import { RelativeTime } from "@/components/RelativeTime";
import { ApprovalRowActions } from "./approvals/approval-actions";

export const dynamic = "force-dynamic";

// Matches the server-side cap in services/api/cases. Named so the header can
// say so out loud when the list is full rather than presenting a truncated
// page as a complete count.
const QUEUE_LIMIT = 100;

// How many pending decisions the landing panel resolves and renders inline.
const WAITING_PANEL_LIMIT = 8;

// Sessions here are episodic: the operator runs a lab scenario, then opens
// the console to ask "what happened, and what is waiting on me?" (PRODUCT.md
// principle 1). The queue answers the first half. This resolves the second
// so the answer is on the landing page rather than one nav click away.
type Decision = { approvalId: string; action: PendingAction; expiresAt: string };

// Returns null — not an empty list — when the approvals read fails. An empty
// list renders "nothing awaiting your decision", and saying that because the
// API was unreachable is a confident false negative in the one place this
// product promises never to guess (PRODUCT.md principle 3). `unresolved`
// counts approvals that are pending but whose action could not be fetched, so
// the page can admit it is showing fewer rows than exist.
async function pendingDecisions(): Promise<{
  rows: Decision[];
  unresolved: number;
  overflow: number;
  totalPending: number;
} | null> {
  let page;
  try {
    page = await listApprovals();
  } catch {
    return null;
  }

  const pending = page.items.filter((a) => a.status === "pending");

  // Bounded on purpose. Each row costs its own getAction round trip inside
  // this server render, so an unbounded list would turn a busy queue into
  // dozens of serial-ish fetches on the landing page. The panel is a "what
  // needs me right now" summary, not the approvals table — the overflow is
  // stated and linked rather than silently dropped.
  const shown = pending.slice(0, WAITING_PANEL_LIMIT);
  const settled = await Promise.all(
    shown.map(async (approval): Promise<Decision | null> => {
      try {
        return {
          approvalId: approval.id,
          action: await getAction(approval.action_id),
          expiresAt: approval.expires_at,
        };
      } catch {
        return null;
      }
    }),
  );

  const rows = settled.filter((r): r is Decision => r !== null);
  return {
    rows,
    unresolved: shown.length - rows.length,
    overflow: pending.length - shown.length,
    totalPending: pending.length,
  };
}

// After a lab scenario the first question is "did the pipeline see it?" —
// and until now that was unanswerable inside the console, because an event
// that arrived without correlating into a case appears nowhere. Returns null
// when the read fails, so the header can stay silent rather than imply the
// pipeline is idle.
async function pipelineFreshness(): Promise<{ lastReceivedAt: string; lastHour: number } | null> {
  try {
    const page = await listEvents(50);
    if (page.items.length === 0) return null;
    const hourAgo = Date.now() - 3_600_000;
    return {
      lastReceivedAt: page.items[0].received_at,
      lastHour: page.items.filter((e) => new Date(e.received_at).getTime() >= hourAgo).length,
    };
  } catch {
    return null;
  }
}

export default async function QueuePage() {
  // Caught here rather than left to app/error.tsx: Next.js redacts details
  // from an error thrown out of a Server Component before it reaches a
  // client error boundary, so a Problem's code/detail would be lost.
  let page;
  try {
    page = await listCases({ limit: QUEUE_LIMIT });
  } catch (err) {
    if (err instanceof ApiProblemError) {
      return <ProblemBanner problem={err.problem} fallbackTitle="Could not load the case queue" />;
    }
    throw err;
  }

  const [decisions, freshness] = await Promise.all([pendingDecisions(), pipelineFreshness()]);
  const waiting = decisions?.rows ?? [];
  const atCap = page.items.length >= QUEUE_LIMIT;
  const openCases = page.items.filter((c) => c.status !== "closed" && c.status !== "dismissed");
  const resolvedCases = page.items.filter((c) => c.status === "closed" || c.status === "dismissed");

  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-col gap-1 border-b border-border pb-4">
        <h1 className="text-2xl font-semibold tracking-tight text-text-primary">Case Queue</h1>
        <p className="text-xs text-text-tertiary">
          {/* The API caps this list; printing the counts bare would present a
              truncation as a total, in a console whose value is that its
              numbers can be trusted. */}
          {atCap ? `showing the ${QUEUE_LIMIT} most recent` : `${openCases.length} open`} &middot;{" "}
          {resolvedCases.length} resolved &middot;{" "}
          {decisions === null ? (
            <span className="font-medium text-warn">approvals unreachable — unknown whether anything needs you</span>
          ) : waiting.length > 0 ? (
            <span className="font-medium text-warn">{decisions.totalPending} awaiting your decision</span>
          ) : (
            "nothing awaiting your decision"
          )}
        </p>
        {freshness && (
          <p className="text-xs text-text-tertiary">
            Ingest: last event <RelativeTime iso={freshness.lastReceivedAt} className="text-text-secondary" />
            {" \u00b7 "}
            {freshness.lastHour} in the last hour
          </p>
        )}
        <NewSinceLastVisit timestamps={page.items.map((c) => c.updated_at ?? c.opened_at)} />
      </header>

      {decisions !== null && decisions.unresolved > 0 && (
        <p className="rounded-md border border-warn/40 bg-warn/[0.07] px-3 py-2 text-xs text-warn">
          {decisions.unresolved} pending approval{decisions.unresolved === 1 ? "" : "s"} could not be loaded and{" "}
          {decisions.unresolved === 1 ? "is" : "are"} not shown below. Open{" "}
          <Link href="/approvals" className="underline">
            Approvals
          </Link>{" "}
          to retry.
        </p>
      )}

      {waiting.length > 0 && (
        <section
          aria-labelledby="waiting-heading"
          className="flex flex-col gap-2.5 rounded-lg border border-warn/40 bg-warn/[0.05] p-3.5"
        >
          <div className="flex items-center justify-between gap-3">
            <h2 id="waiting-heading" className="text-sm font-semibold tracking-tight text-warn">
              Waiting on you
            </h2>
            <Link href="/approvals" className="text-2xs font-medium text-text-secondary hover:text-text-primary">
              {decisions && decisions.overflow > 0 ? `${decisions.overflow} more in Approvals` : "All approvals"}
            </Link>
          </div>

          <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border bg-surface">
            {waiting.map(({ approvalId, action, expiresAt }) => (
              <div
                key={approvalId}
                className="flex flex-col gap-2.5 p-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
              >
                <div className="flex min-w-0 flex-col gap-1.5">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <ModeBadge mode={action.mode} />
                    <SeverityBadge severity={action.risk} />
                    <Link
                      href={`/cases/${action.case_id}`}
                      className="font-mono text-2xs text-accent hover:underline"
                    >
                      {action.case_id}
                    </Link>
                  </div>
                  <p className="truncate text-xs text-text-secondary">
                    <span className="text-text-primary">{action.intent.replaceAll("_", " ")}</span>{" "}
                    <code className="font-mono text-2xs text-text-tertiary">{action.target.value}</code>
                  </p>
                </div>
                <div className="shrink-0">
                  <ApprovalRowActions approvalId={approvalId} action={action} expiresAt={expiresAt} size="md" />
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {openCases.length === 0 && resolvedCases.length === 0 ? (
        <EmptyState
          title="No cases yet"
          body="Ingested events that produce a case will show up here, grouped by severity."
        />
      ) : (
        <main className="flex flex-col gap-8">
          {openCases.length > 0 && (
            <section aria-label="Open cases grouped by severity" className="flex flex-col gap-3">
              <CaseListGroup cases={openCases} />
            </section>
          )}

          {resolvedCases.length > 0 && (
            <section aria-labelledby="resolved-cases-heading" className="flex flex-col gap-2.5 pt-2">
              <div className="flex items-center gap-2 pl-0.5 text-xs font-semibold tracking-wider text-text-secondary uppercase">
                <span aria-hidden="true" className="h-2 w-2 rounded-full bg-text-tertiary" />
                <h2 id="resolved-cases-heading">Resolved &amp; dismissed</h2>
                <span className="rounded-full bg-surface-raised px-1.5 py-px font-mono text-2xs font-normal text-text-tertiary">
                  {resolvedCases.length}
                </span>
              </div>

              <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border/80 bg-surface/50">
                {resolvedCases.map((item) => (
                  <div key={item.id} className="transition-colors duration-150 hover:bg-surface-raised/40">
                    <CaseCard item={item} />
                  </div>
                ))}
              </div>
            </section>
          )}
        </main>
      )}
    </div>
  );
}
