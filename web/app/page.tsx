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
    <div className="flex flex-col gap-[22px]">
      <header>
        <h1 className="text-xl font-semibold tracking-tight">Alert Queue</h1>
        <p className="text-xs text-text-tertiary">
          {openCases.length} open, {resolvedCases.length} resolved &middot; sorted by severity
        </p>
      </header>

      {openCases.length === 0 && resolvedCases.length === 0 ? (
        <EmptyState
          title="No cases yet"
          body="Ingested events that produce a case will show up here, grouped by severity."
        />
      ) : (
        <div className="flex flex-col gap-[18px]">
          {openCases.length > 0 && <CaseListGroup cases={openCases} />}

          {resolvedCases.length > 0 && (
            <div>
              <div className="mb-2 flex items-center gap-2 pl-0.5 text-[11.5px] font-semibold tracking-wider text-text-tertiary uppercase">
                <span className="h-[7px] w-[7px] rounded-full bg-text-tertiary" />
                Resolved
                <span className="font-mono font-normal text-text-tertiary">{resolvedCases.length}</span>
              </div>
              <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border opacity-60">
                {resolvedCases.map((item) => (
                  <CaseCard key={item.id} item={item} />
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
