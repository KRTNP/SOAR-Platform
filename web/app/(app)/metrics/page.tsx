import { getMetrics, listEvents, ApiProblemError } from "@/lib/api-client/client";
import type { Severity } from "@/lib/api-client/types";
import { ProblemBanner } from "@/components/ProblemBanner";
import { EmptyState } from "@/components/EmptyState";
import { SeverityBadge } from "@/components/badges";
import { RelativeTime } from "@/components/RelativeTime";

export const dynamic = "force-dynamic";

const EVENT_LIMIT = 15;

// events.Summary.severity is a plain string on the wire; the badge takes the
// closed domain vocabulary. Anything outside it renders as informational
// rather than crashing on an unexpected value from a new source.
const SEVERITIES = new Set(["informational", "low", "medium", "high", "critical"]);
function normalizeSeverity(raw: string): Severity {
  return SEVERITIES.has(raw) ? (raw as Severity) : "informational";
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${(seconds / 3600).toFixed(1)}h`;
}

function StatTile({ label, value, caption }: { label: string; value: string; caption?: string }) {
  return (
    <div className="flex flex-col gap-1.5 rounded-md border border-border bg-surface px-5 py-4">
      <span className="text-2xs font-semibold tracking-wider text-text-tertiary uppercase">{label}</span>
      <span className="font-mono text-2xl font-semibold tracking-tight text-text-primary">{value}</span>
      {caption && <span className="text-xs text-text-tertiary">{caption}</span>}
    </div>
  );
}

export default async function MetricsPage() {
  let snap;
  try {
    snap = await getMetrics();
  } catch (err) {
    if (err instanceof ApiProblemError) {
      return <ProblemBanner problem={err.problem} fallbackTitle="Could not load metrics" />;
    }
    throw err;
  }

  // null distinguishes "could not read the event stream" from "no events" —
  // the section says which, rather than letting an unreachable API look like
  // an idle pipeline.
  const events = await listEvents(EVENT_LIMIT)
    .then((page) => page.items)
    .catch(() => null);

  const evidenceRatio = snap.total_cases > 0 ? Math.round((snap.cases_with_verified_evidence / snap.total_cases) * 100) : null;
  const totalReleases = snap.auto_release_count + snap.manual_release_count;

  return (
    <div className="flex flex-col gap-5">
      <header className="flex flex-col gap-1 border-b border-border pb-4">
        <h1 className="text-2xl font-semibold tracking-tight">Metrics</h1>
        <p className="text-xs text-text-tertiary">
          Point-in-time KPI snapshot, not a historical baseline &middot; computed{" "}
          <span className="font-mono">{new Date(snap.computed_at).toLocaleString()}</span>
        </p>
      </header>

      <section className="rounded-md border border-border bg-surface px-5 py-4">
        <h2 className="mb-3 text-sm font-semibold">Active connector</h2>
        <div className="flex flex-wrap items-center gap-2.5 text-xs">
          <span className="rounded-sm border border-border-strong bg-surface-raised px-2.5 py-1 font-mono text-text-secondary">
            {snap.connector_target}
          </span>
          <span className="text-text-tertiary">&middot;</span>
          <span className="font-mono text-text-secondary">{snap.connector_version}</span>
        </div>
      </section>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatTile label="Total cases" value={String(snap.total_cases)} />
        <StatTile
          label="Evidence verified"
          value={evidenceRatio === null ? "—" : `${evidenceRatio}%`}
          caption={`${snap.cases_with_verified_evidence} of ${snap.total_cases} cases`}
        />
        <StatTile label="Analyst overrides" value={String(snap.analyst_override_count)} caption="rejected approvals" />
        <StatTile
          label="Mean time to containment"
          value={snap.mean_time_to_containment_seconds != null ? formatDuration(snap.mean_time_to_containment_seconds) : "—"}
          caption={snap.mean_time_to_containment_seconds == null ? "no allowed executions yet" : "case open to first allowed execute"}
        />
      </div>

      <section className="rounded-md border border-border bg-surface px-5 py-4">
        <h2 className="mb-3 text-sm font-semibold">Rollback</h2>
        {totalReleases === 0 ? (
          <p className="text-xs text-text-tertiary">No blocks released yet.</p>
        ) : (
          <div className="flex items-center gap-2 text-xs">
            <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-raised">
              <div
                className="h-full rounded-full bg-accent"
                style={{ width: `${Math.round((snap.auto_release_count / totalReleases) * 100)}%` }}
              />
            </div>
            <span className="font-mono text-text-secondary">
              {snap.auto_release_count} auto &middot; {snap.manual_release_count} manual
            </span>
          </div>
        )}
      </section>

      <section className="rounded-md border border-border bg-surface px-5 py-4">
        <h2 className="mb-3 text-sm font-semibold">Connector reliability</h2>
        {snap.connector_reliability.length === 0 ? (
          <EmptyState
            title="No executions recorded yet"
            body="Allowed/denied counts appear here once actions have been executed."
            action={{ label: "Back to case queue", href: "/" }}
          />
        ) : (
          <div className="flex flex-col divide-y divide-border">
            {snap.connector_reliability.map((o) => (
              <div key={o.connector_version} className="flex items-center justify-between gap-3 py-2 text-xs">
                <span className="truncate font-mono text-text-secondary">{o.connector_version}</span>
                <span className="flex shrink-0 items-center gap-3 font-mono text-text-tertiary">
                  <span className="text-safe">{o.allowed} allowed</span>
                  <span className="text-danger">{o.denied} denied</span>
                </span>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Ingest evidence. A case list answers "what correlated"; this answers
          "what arrived", which is the prior question after a lab scenario and
          the only place an event that produced no case is visible at all. */}
      <section className="rounded-md border border-border bg-surface px-5 py-4">
        <h2 className="mb-3 text-sm font-semibold">Recent events</h2>
        {events === null ? (
          <p className="text-xs text-warn">Event stream unreachable — ingest activity unknown.</p>
        ) : events.length === 0 ? (
          <EmptyState title="No events ingested yet" body="Events appear here as soon as a source posts one, whether or not they correlate into a case." />
        ) : (
          <div className="flex flex-col divide-y divide-border">
            {events.map((e) => (
              <div key={e.id} className="flex flex-wrap items-center justify-between gap-2 py-2 text-xs">
                <div className="flex min-w-0 items-center gap-2">
                  <SeverityBadge severity={normalizeSeverity(e.severity)} />
                  <span className="truncate text-text-secondary">{e.event_class}</span>
                  <span className="shrink-0 font-mono text-2xs text-text-tertiary">{e.source}</span>
                </div>
                <RelativeTime iso={e.received_at} className="shrink-0 font-mono text-2xs text-text-tertiary" />
              </div>
            ))}
          </div>
        )}
        <p className="mt-3 text-2xs text-text-tertiary">
          The {EVENT_LIMIT} most recent, newest first. An event listed here has been ingested; whether it opened a case is a
          separate question answered on the queue.
        </p>
      </section>

      <p className="text-2xs text-text-tertiary">
        MTTT (alert-to-triage), false-positive harm and automation coverage are not shown — this system does not yet record the
        underlying data (a triage timestamp, a manual outcome tag) needed to compute them. See adr/0015-kpi-snapshot.md.
      </p>
    </div>
  );
}
