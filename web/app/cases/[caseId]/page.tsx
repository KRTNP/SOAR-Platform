import Link from "next/link";
import { getCase, getCaseTimeline, getCaseEvidence, ApiProblemError } from "@/lib/api-client/client";
import { SeverityBadge, StatusBadge, VerdictBadge, IntegrityBadge } from "@/components/badges";
import { Timeline } from "@/components/Timeline";
import { ProblemBanner } from "@/components/ProblemBanner";
import { StatusControl, CloseCaseControl, TicketControl, ActionControl, ExportControl } from "./status-control";

export const dynamic = "force-dynamic";

export default async function CaseDetailPage({ params }: { params: Promise<{ caseId: string }> }) {
  const { caseId } = await params;

  let caseItem, timeline, evidence;
  try {
    [caseItem, timeline, evidence] = await Promise.all([getCase(caseId), getCaseTimeline(caseId), getCaseEvidence(caseId)]);
  } catch (err) {
    if (err instanceof ApiProblemError) {
      return <ProblemBanner problem={err.problem} fallbackTitle="Could not load this case" />;
    }
    throw err;
  }

  return (
    <div className="flex flex-col gap-[22px]">
      <div>
        <Link href="/" className="mb-4 flex w-fit items-center gap-2 text-[12.5px] text-text-tertiary hover:text-text-secondary">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" className="h-3.5 w-3.5">
            <polyline points="15 18 9 12 15 6" />
          </svg>
          Alert Queue
        </Link>

        <div className="mb-2.5 flex flex-wrap items-center gap-x-3 gap-y-1.5">
          <span className="font-mono text-[13px] whitespace-nowrap text-text-tertiary">{caseItem.id}</span>
          <SeverityBadge severity={caseItem.severity} size="lg" />
          <StatusBadge status={caseItem.status} />
        </div>

        <h1 className="mb-2.5 text-[22px] font-semibold tracking-tight">{caseItem.title}</h1>

        <div className="flex flex-wrap items-center gap-2.5 border-b border-border pb-[22px] text-[12.5px] text-text-secondary">
          <span>
            Owner <strong className="font-semibold text-text-primary">{caseItem.owner}</strong>
          </span>
          <span className="text-text-tertiary">&middot;</span>
          <span>
            Opened <span className="font-mono">{new Date(caseItem.opened_at).toLocaleString()}</span>
          </span>
          {caseItem.updated_at && (
            <>
              <span className="text-text-tertiary">&middot;</span>
              <span>
                Updated <span className="font-mono">{new Date(caseItem.updated_at).toLocaleString()}</span>
              </span>
            </>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-5 md:grid-cols-[1fr_320px]">
        <div className="flex flex-col gap-[18px]">
          {(caseItem.hypothesis || (caseItem.facts?.length ?? 0) > 0 || (caseItem.limitations?.length ?? 0) > 0) && (
            <section className="rounded-md border border-border bg-surface p-[18px_20px]">
              <h2 className="mb-3 text-[13px] font-semibold">Summary</h2>
              {caseItem.hypothesis && <p className="mb-3.5 text-[13.5px] text-text-secondary italic">&ldquo;{caseItem.hypothesis}&rdquo;</p>}
              <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
                {(caseItem.facts?.length ?? 0) > 0 && (
                  <div>
                    <h3 className="mb-1.5 text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">Facts</h3>
                    <ul className="flex list-disc flex-col gap-1 pl-4 text-[12.5px] text-text-secondary">
                      {caseItem.facts!.map((f, i) => (
                        <li key={i}>{f}</li>
                      ))}
                    </ul>
                  </div>
                )}
                {(caseItem.limitations?.length ?? 0) > 0 && (
                  <div>
                    <h3 className="mb-1.5 text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">Limitations</h3>
                    <ul className="flex list-disc flex-col gap-1 pl-4 text-[12.5px] text-text-tertiary">
                      {caseItem.limitations!.map((l, i) => (
                        <li key={i}>{l}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            </section>
          )}

          <section className="rounded-md border border-border bg-surface p-[18px_20px]">
            <h2 className="mb-3 text-[13px] font-semibold">Timeline</h2>
            <Timeline entries={timeline.items} />
          </section>

          {evidence.items.length > 0 && (
            <section className="rounded-md border border-border bg-surface p-[18px_20px]">
              <h2 className="mb-3 text-[13px] font-semibold">Evidence</h2>
              <div className="flex flex-col gap-2">
                {evidence.items.map((e) => (
                  <div key={e.id} className="flex items-center justify-between gap-2 text-[12.5px]">
                    <span className="text-text-secondary">
                      {e.artifact_type} <span className="text-text-tertiary">({e.source})</span>
                    </span>
                    <IntegrityBadge status={e.integrity_status} />
                  </div>
                ))}
              </div>
            </section>
          )}
        </div>

        <aside className="flex flex-col gap-4">
          <StatusControl caseItem={caseItem} />
          <CloseCaseControl caseItem={caseItem} />
          <TicketControl caseItem={caseItem} />
          <ActionControl caseItem={caseItem} />
          <ExportControl caseItem={caseItem} />

          {(caseItem.observables?.length ?? 0) > 0 && (
            <section className="rounded-md border border-border bg-surface p-[18px_20px]">
              <h2 className="mb-3 text-[13px] font-semibold">Observables</h2>
              <div className="flex flex-col gap-2">
                {caseItem.observables!.map((o, i) => (
                  <div key={i} className="flex items-center justify-between gap-2">
                    <span className="rounded-[3px] border border-border bg-surface-raised px-2.5 py-1 font-mono text-[11.5px] text-text-secondary">
                      {o.value}
                    </span>
                    {o.verdict && <VerdictBadge verdict={o.verdict} />}
                  </div>
                ))}
              </div>
            </section>
          )}
        </aside>
      </div>
    </div>
  );
}
