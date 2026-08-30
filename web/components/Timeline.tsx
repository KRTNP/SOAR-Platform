import type { CaseTimelineEntry } from "@/lib/api-client/types";

const KIND_LABELS: Record<string, string> = {
  case_opened: "Case opened",
  status_changed: "Status changed",
  case_closed: "Case closed",
  alert_linked: "Alert linked",
  field_updated: "Field updated",
};

function formatDetail(entry: CaseTimelineEntry): string | null {
  if (!entry.detail) return null;
  if (entry.kind === "status_changed" && entry.detail.from && entry.detail.to) {
    const reason = entry.detail.reason ? ` — ${entry.detail.reason}` : "";
    return `${entry.detail.from} → ${entry.detail.to}${reason}`;
  }
  if (entry.kind === "case_opened" && entry.detail.event_id) {
    return `from ${entry.detail.source ?? "event"} ${entry.detail.event_id}`;
  }
  if (entry.kind === "case_closed" && entry.detail.closure_reason) {
    return String(entry.detail.closure_reason);
  }
  return null;
}

export function Timeline({ entries }: { entries: CaseTimelineEntry[] }) {
  if (entries.length === 0) {
    return <p className="text-[13px] text-text-tertiary">No timeline entries yet.</p>;
  }

  return (
    <ol className="flex flex-col gap-3">
      {entries.map((entry) => (
        <li key={entry.id} className="flex items-start gap-3">
          <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-border-strong" />
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <div className="flex items-baseline gap-2">
              <span className="text-[13px] font-medium">{KIND_LABELS[entry.kind] ?? entry.kind}</span>
              <span className="text-[11px] text-text-tertiary">by {entry.actor}</span>
            </div>
            {formatDetail(entry) && <span className="text-[12.5px] text-text-secondary">{formatDetail(entry)}</span>}
          </div>
          <time className="shrink-0 font-mono text-[11px] text-text-tertiary" dateTime={entry.occurred_at}>
            {new Date(entry.occurred_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
          </time>
        </li>
      ))}
    </ol>
  );
}
