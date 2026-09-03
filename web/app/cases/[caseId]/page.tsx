import Link from "next/link";
import { getCase, getCaseTimeline, getCaseEvidence, ApiProblemError } from "@/lib/api-client/client";
import { SeverityBadge, VerdictBadge, IntegrityBadge } from "@/components/badges";
import { Timeline } from "@/components/Timeline";
import { ProblemBanner } from "@/components/ProblemBanner";
import { CaseHeaderActions, ActionControl } from "./status-control";

export const dynamic = "force-dynamic";

export default async function CaseDetailPage({ params }: { params: Promise<{ caseId: string }> }) {
    const { caseId } = await params;

    let caseItem, timeline, evidence;
    try {
        [caseItem, timeline, evidence] = await Promise.all([
            getCase(caseId),
            getCaseTimeline(caseId),
            getCaseEvidence(caseId),
        ]);
    } catch (err) {
        if (err instanceof ApiProblemError) {
            return <ProblemBanner problem={err.problem} fallbackTitle="Could not load this case" />;
        }
        throw err;
    }

    const hasSummaryContent =
        Boolean(caseItem.hypothesis) ||
        (caseItem.facts?.length ?? 0) > 0 ||
        (caseItem.limitations?.length ?? 0) > 0;

    return (
        <div className="flex flex-col gap-6">
            {/* Top Navigation & Case Header */}
            <header className="flex flex-col gap-3 border-b border-border pb-5">
                {/* Back Link */}
                <Link
                    href="/"
                    aria-label="Back to Alert Queue"
                    className="group -ml-1 inline-flex w-fit items-center gap-1.5 rounded-md px-2 py-1 text-[12px] font-medium text-text-tertiary transition-colors hover:bg-surface-raised hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
                >
                    <svg
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth={2}
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="h-3.5 w-3.5 transition-transform group-hover:-translate-x-0.5"
                        aria-hidden="true"
                    >
                        <polyline points="15 18 9 12 15 6" />
                    </svg>
                    Alert Queue
                </Link>

                {/* Case Identity + Primary Actions — one row on desktop so
                    status/ticket/export/close read as a toolbar next to the
                    case they act on, not a separate stacked form further
                    down the page. */}
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div className="flex flex-col gap-1.5">
                        <div className="flex flex-wrap items-center gap-2">
                            <span className="rounded bg-surface-raised px-2 py-0.5 font-mono text-[12px] font-medium text-text-secondary border border-border-strong/40">
                                {caseItem.id}
                            </span>
                            <SeverityBadge severity={caseItem.severity} size="lg" />
                        </div>

                        <h1 className="text-xl font-semibold tracking-tight text-text-primary sm:text-2xl">
                            {caseItem.title}
                        </h1>

                        <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-text-secondary">
                            <div className="flex items-center gap-1.5">
                                <span className="text-text-tertiary">Owner:</span>
                                <span className="font-medium text-text-primary">{caseItem.owner}</span>
                            </div>

                            <span aria-hidden="true" className="text-text-tertiary/50">•</span>

                            <div className="flex items-center gap-1">
                                <span className="text-text-tertiary">Opened:</span>
                                <time dateTime={caseItem.opened_at} className="font-mono text-[11.5px] text-text-secondary">
                                    {new Date(caseItem.opened_at).toLocaleString()}
                                </time>
                            </div>

                            {caseItem.updated_at && (
                                <>
                                    <span aria-hidden="true" className="text-text-tertiary/50">•</span>
                                    <div className="flex items-center gap-1">
                                        <span className="text-text-tertiary">Updated:</span>
                                        <time dateTime={caseItem.updated_at} className="font-mono text-[11.5px] text-text-secondary">
                                            {new Date(caseItem.updated_at).toLocaleString()}
                                        </time>
                                    </div>
                                </>
                            )}
                        </div>
                    </div>

                    <CaseHeaderActions caseItem={caseItem} />
                </div>
            </header>

            {/* Main Responsive Grid Layout */}
            <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-[1fr_320px]">
                {/* Left Column: Forensic Findings, Timeline & Evidence */}
                <main className="flex flex-col gap-6">
                    {/* Summary Section */}
                    {hasSummaryContent && (
                        <section
                            aria-labelledby="summary-heading"
                            className="rounded-lg border border-border bg-surface p-5 shadow-xs"
                        >
                            <h2 id="summary-heading" className="mb-3 text-[13px] font-semibold text-text-primary">
                                Executive Summary
                            </h2>

                            {caseItem.hypothesis && (
                                <div className="mb-4 rounded-md border-l-2 border-accent bg-surface-raised/60 p-3 text-[13px] text-text-secondary">
                                    <p className="font-medium text-text-primary mb-1 text-[11px] uppercase tracking-wider">
                                        Working Hypothesis
                                    </p>
                                    <p className="italic leading-relaxed">&ldquo;{caseItem.hypothesis}&rdquo;</p>
                                </div>
                            )}

                            <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
                                {(caseItem.facts?.length ?? 0) > 0 && (
                                    <div className="flex flex-col gap-1.5">
                                        <h3 className="text-[11px] font-semibold tracking-wider text-text-tertiary uppercase">
                                            Confirmed Facts ({caseItem.facts!.length})
                                        </h3>
                                        <ul className="flex flex-col gap-1.5 text-[12.5px] text-text-secondary">
                                            {caseItem.facts!.map((fact, idx) => (
                                                <li key={idx} className="flex items-start gap-2">
                                                    <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-accent" aria-hidden="true" />
                                                    <span>{fact}</span>
                                                </li>
                                            ))}
                                        </ul>
                                    </div>
                                )}

                                {(caseItem.limitations?.length ?? 0) > 0 && (
                                    <div className="flex flex-col gap-1.5">
                                        <h3 className="text-[11px] font-semibold tracking-wider text-text-tertiary uppercase">
                                            Investigation Limitations
                                        </h3>
                                        <ul className="flex flex-col gap-1.5 text-[12.5px] text-text-tertiary">
                                            {caseItem.limitations!.map((limitation, idx) => (
                                                <li key={idx} className="flex items-start gap-2">
                                                    <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-border-strong" aria-hidden="true" />
                                                    <span>{limitation}</span>
                                                </li>
                                            ))}
                                        </ul>
                                    </div>
                                )}
                            </div>
                        </section>
                    )}

                    {/* Timeline Section */}
                    <section
                        aria-labelledby="timeline-heading"
                        className="rounded-lg border border-border bg-surface p-5 shadow-xs"
                    >
                        <div className="mb-3.5 flex items-center justify-between">
                            <h2 id="timeline-heading" className="text-[13px] font-semibold text-text-primary">
                                Activity Timeline
                            </h2>
                            <span className="rounded-full bg-surface-raised px-2 py-0.5 text-[11px] text-text-tertiary">
                                {timeline.items.length} events
                            </span>
                        </div>
                        <Timeline entries={timeline.items} />
                    </section>

                    {/* Evidence Section */}
                    {evidence.items.length > 0 && (
                        <section
                            aria-labelledby="evidence-heading"
                            className="rounded-lg border border-border bg-surface p-5 shadow-xs"
                        >
                            <div className="mb-3.5 flex items-center justify-between">
                                <h2 id="evidence-heading" className="text-[13px] font-semibold text-text-primary">
                                    Artifact Evidence
                                </h2>
                                <span className="rounded-full bg-surface-raised px-2 py-0.5 text-[11px] text-text-tertiary">
                                    {evidence.items.length} artifacts
                                </span>
                            </div>

                            <div className="flex flex-col divide-y divide-border/70">
                                {evidence.items.map((item) => (
                                    <article key={item.id} className="flex flex-col gap-2 py-3.5 first:pt-0 last:pb-0">
                                        <div className="flex flex-wrap items-center justify-between gap-2">
                                            <div className="flex items-center gap-2 text-[13px] font-medium text-text-primary">
                                                <span>{item.artifact_type}</span>
                                                <span className="rounded bg-surface-raised px-1.5 py-0.5 font-mono text-[11px] text-text-tertiary">
                                                    {item.source}
                                                </span>
                                            </div>
                                            <IntegrityBadge status={item.integrity_status} />
                                        </div>

                                        {/* Formatted Tag Metadata */}
                                        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-[11.5px] text-text-tertiary">
                                            {item.tool_name && (
                                                <span>
                                                    Tool: <strong className="font-normal text-text-secondary">{item.tool_name}{item.tool_version && ` v${item.tool_version}`}</strong>
                                                </span>
                                            )}
                                            {item.tool_name && <span>•</span>}
                                            <span>
                                                Analyst: <strong className="font-normal text-text-secondary">{item.analyst}</strong>
                                            </span>
                                            <span>•</span>
                                            <span className="capitalize">Sensitivity: {item.sensitivity}</span>
                                            <span>•</span>
                                            <span>Retention: {item.retention_class}</span>
                                        </div>

                                        {item.purpose && (
                                            <p className="rounded bg-surface-raised/40 px-2.5 py-1.5 text-[12px] italic text-text-secondary">
                                                &ldquo;{item.purpose}&rdquo;
                                            </p>
                                        )}

                                        {(item.limitations?.length ?? 0) > 0 && (
                                            <p className="text-[11.5px] text-text-tertiary">
                                                <span className="font-medium text-text-secondary">Limitations:</span>{" "}
                                                {item.limitations!.join("; ")}
                                            </p>
                                        )}
                                    </article>
                                ))}
                            </div>
                        </section>
                    )}
                </main>

                {/* Right Column: Case Overview, Containment & Observables
                    (Sticky on Desktop) — status/close/ticket/export moved to
                    the header action bar above; this column is now a flat
                    metadata panel plus the one control that's genuinely
                    stateful enough to earn its own card (network
                    containment), not five stacked bordered mini-forms. */}
                <aside
                    aria-label="Case overview and controls"
                    className="flex flex-col gap-5 lg:sticky lg:top-6"
                >
                    {/* Overview Card — always present, so this column never
                        goes empty just because there's nothing to action
                        right now (a short-timeline case previously left this
                        whole column as one form-shaped box regardless). */}
                    <div className="rounded-lg border border-border bg-surface p-4 shadow-xs">
                        <h2 className="mb-3 text-[11px] font-semibold tracking-wider text-text-tertiary uppercase">
                            Case Overview
                        </h2>
                        <dl className="flex flex-col divide-y divide-border/60 text-[12.5px]">
                            <div className="flex items-center justify-between py-1.5 first:pt-0">
                                <dt className="text-text-tertiary">Category</dt>
                                <dd className="font-medium text-text-primary capitalize">{caseItem.category}</dd>
                            </div>
                            <div className="flex items-center justify-between py-1.5">
                                <dt className="text-text-tertiary">Confidence</dt>
                                <dd className="font-mono text-text-primary">{Math.round(caseItem.confidence * 100)}%</dd>
                            </div>
                            {caseItem.correlation_key && (
                                <div className="flex items-center justify-between gap-2 py-1.5">
                                    <dt className="shrink-0 text-text-tertiary">Correlation</dt>
                                    <dd className="truncate font-mono text-[11px] text-text-secondary" title={caseItem.correlation_key}>
                                        {caseItem.correlation_key}
                                    </dd>
                                </div>
                            )}
                            {caseItem.incident_commander && (
                                <div className="flex items-center justify-between py-1.5">
                                    <dt className="text-text-tertiary">Incident commander</dt>
                                    <dd className="font-medium text-text-primary">{caseItem.incident_commander}</dd>
                                </div>
                            )}
                            {caseItem.service_owner && (
                                <div className="flex items-center justify-between py-1.5">
                                    <dt className="text-text-tertiary">Service owner</dt>
                                    <dd className="font-medium text-text-primary">{caseItem.service_owner}</dd>
                                </div>
                            )}
                            {caseItem.closure_reason && (
                                <div className="flex flex-col gap-1 py-1.5 last:pb-0">
                                    <dt className="text-text-tertiary">Closure reason</dt>
                                    <dd className="text-text-secondary">{caseItem.closure_reason}</dd>
                                </div>
                            )}
                        </dl>
                    </div>

                    <ActionControl caseItem={caseItem} />

                    {/* Observables Card */}
                    {(caseItem.observables?.length ?? 0) > 0 && (
                        <section
                            aria-labelledby="observables-heading"
                            className="rounded-lg border border-border bg-surface p-4 shadow-xs"
                        >
                            <div className="mb-3 flex items-center justify-between">
                                <h2 id="observables-heading" className="text-[12px] font-semibold tracking-wider text-text-tertiary uppercase">
                                    Observables
                                </h2>
                                <span className="rounded-full bg-surface-raised px-1.5 py-0.2 font-mono text-[10.5px] text-text-tertiary">
                                    {caseItem.observables!.length}
                                </span>
                            </div>

                            <div className="flex flex-col gap-2">
                                {caseItem.observables!.map((obs, idx) => (
                                    <div
                                        key={idx}
                                        className="flex items-center justify-between gap-2 rounded-md border border-border/80 bg-surface-raised/50 p-1.5 px-2 transition-colors hover:bg-surface-raised"
                                    >
                                        <code className="truncate font-mono text-[11.5px] text-text-secondary" title={obs.value}>
                                            {obs.value}
                                        </code>
                                        {obs.verdict && <VerdictBadge verdict={obs.verdict} />}
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