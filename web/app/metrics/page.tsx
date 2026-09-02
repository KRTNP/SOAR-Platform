import { getMetrics, ApiProblemError } from "@/lib/api-client/client";
import { ProblemBanner } from "@/components/ProblemBanner";
import { EmptyState } from "@/components/EmptyState";

export const dynamic = "force-dynamic";

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${(seconds / 3600).toFixed(1)}h`;
}

function StatTile({ label, value, caption }: { label: string; value: string; caption?: string }) {
  return (
    <div className="flex flex-col gap-1.5 rounded-md border border-border bg-surface p-[18px_20px]">
      <span className="text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">{label}</span>
      <span className="font-mono text-2xl font-semibold tracking-tight text-text-primary">{value}</span>
      {caption && <span className="text-[12px] text-text-tertiary">{caption}</span>}
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

  const evidenceRatio = snap.total_cases > 0 ? Math.round((snap.cases_with_verified_evidence / snap.total_cases) * 100) : null;
  const totalReleases = snap.auto_release_count + snap.manual_release_count;

  return (
    <div className="flex flex-col gap-[22px]">
      <header>
        <h1 className="text-xl font-semibold tracking-tight">Metrics</h1>
        <p className="text-xs text-text-tertiary">
          Point-in-time KPI snapshot, not a historical baseline &middot; computed{" "}
          <span className="font-mono">{new Date(snap.computed_at).toLocaleString()}</span>
        </p>
      </header>

      <section className="rounded-md border border-border bg-surface p-[18px_20px]">
        <h2 className="mb-3 text-[13px] font-semibold">Active connector</h2>
        <div className="flex flex-wrap items-center gap-2.5 text-[12.5px]">
          <span className="rounded-[3px] border border-border-strong bg-surface-raised px-2.5 py-1 font-mono text-text-secondary">
            {snap.connector_target}
          </span>
          <span className="text-text-tertiary">&middot;</span>
          <span className="font-mono text-text-secondary">{snap.connector_version}</span>
        </div>
      </section>

      <div className="grid grid-cols-4 gap-[18px]">
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

      <section className="rounded-md border border-border bg-surface p-[18px_20px]">
        <h2 className="mb-3 text-[13px] font-semibold">Rollback</h2>
        {totalReleases === 0 ? (
          <p className="text-[12.5px] text-text-tertiary">No blocks released yet.</p>
        ) : (
          <div className="flex items-center gap-2 text-[12.5px]">
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

      <section className="rounded-md border border-border bg-surface p-[18px_20px]">
        <h2 className="mb-3 text-[13px] font-semibold">Connector reliability</h2>
        {snap.connector_reliability.length === 0 ? (
          <EmptyState title="No executions recorded yet" body="Allowed/denied counts appear here once actions have been executed." />
        ) : (
          <div className="flex flex-col divide-y divide-border">
            {snap.connector_reliability.map((o) => (
              <div key={o.connector_version} className="flex items-center justify-between gap-3 py-2 text-[12.5px]">
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

      <p className="text-[11.5px] text-text-tertiary">
        MTTT (alert-to-triage), false-positive harm and automation coverage are not shown — this system does not yet record the
        underlying data (a triage timestamp, a manual outcome tag) needed to compute them. See adr/0015-kpi-snapshot.md.
      </p>
    </div>
  );
}
