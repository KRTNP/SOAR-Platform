import { listCases, ApiProblemError } from "@/lib/api-client/client";
import { CaseListGroup } from "@/components/CaseListGroup";
import { CaseCard } from "@/components/CaseCard";
import { EmptyState } from "@/components/EmptyState";
import { ProblemBanner } from "@/components/ProblemBanner";

export const dynamic = "force-dynamic";

export default async function QueuePage() {
  // Caught here rather than left to app/error.tsx: Next.js redacts details
  // from an error thrown out of a Server Component before it reaches a
  // client error boundary, so a Problem's code/detail would be lost.
  let page;
  try {
    page = await listCases({ limit: 100 });
  } catch (err) {
    if (err instanceof ApiProblemError) {
      return <ProblemBanner problem={err.problem} fallbackTitle="Could not load the alert queue" />;
    }
    throw err;
  }

  const openCases = page.items.filter((c) => c.status !== "closed" && c.status !== "dismissed");
  const resolvedCases = page.items.filter((c) => c.status === "closed" || c.status === "dismissed");

  return (
    <div className="flex flex-col gap-6">
      {/* Header: Overview & Triage Metrics */}
      <header className="flex flex-col gap-3 border-b border-border pb-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2.5">
            <h1 className="text-xl font-semibold tracking-tight text-text-primary sm:text-2xl">
              Alert Queue
            </h1>
            {openCases.length > 0 ? (
              <span className="inline-flex items-center rounded-full bg-accent/15 px-2.5 py-0.5 text-[11.5px] font-medium text-accent">
                {openCases.length} active
              </span>
            ) : (
              <span className="inline-flex items-center rounded-full bg-surface-raised px-2.5 py-0.5 text-[11.5px] font-medium text-text-tertiary">
                Queue empty
              </span>
            )}
          </div>
          <p className="text-xs text-text-tertiary">
            Active security incidents sorted by risk severity &middot; investigate or transition status inline.
          </p>
        </div>

        {/* Quick Status Metric Pills */}
        <div className="flex items-center gap-2 text-xs">
          <div className="flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1 text-text-secondary shadow-2xs">
            <span className="font-semibold text-text-primary">{openCases.length}</span> Open
          </div>
          <div className="flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1 text-text-secondary shadow-2xs">
            <span className="font-semibold text-text-primary">{resolvedCases.length}</span> Resolved
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      {openCases.length === 0 && resolvedCases.length === 0 ? (
        <EmptyState
          title="No cases yet"
          body="Ingested events that produce a case will show up here, grouped by severity."
        />
      ) : (
        <main className="flex flex-col gap-8">
          {/* Active Cases Section */}
          {openCases.length > 0 && (
            <section aria-label="Open Cases Grouped by Severity" className="flex flex-col gap-3">
              <CaseListGroup cases={openCases} />
            </section>
          )}

          {/* Resolved History Section */}
          {resolvedCases.length > 0 && (
            <section
              aria-labelledby="resolved-cases-heading"
              className="flex flex-col gap-2.5 pt-2"
            >
              <div className="flex items-center justify-between pl-0.5">
                <div className="flex items-center gap-2 text-[12px] font-semibold tracking-wider text-text-secondary uppercase">
                  <span aria-hidden="true" className="h-2 w-2 rounded-full bg-text-tertiary" />
                  <h2 id="resolved-cases-heading">Resolved & Dismissed</h2>
                  <span className="rounded-full bg-surface-raised px-1.5 py-0.2 font-mono text-[11px] font-normal text-text-tertiary">
                    {resolvedCases.length}
                  </span>
                </div>
              </div>

              {/* Subdued History List (Accessible Alternative to opacity-60) */}
              <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border/80 bg-surface/50 shadow-2xs">
                {resolvedCases.map((item) => (
                  <div
                    key={item.id}
                    className="transition-colors duration-150 hover:bg-surface-raised/40"
                  >
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