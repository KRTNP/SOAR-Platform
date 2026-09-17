"use client";

import { useMemo, useState } from "react";
import type { CaseTimelineEntry } from "@/lib/api-client/types";

// Every kind services/api/cases writes today. A kind missing from this map
// used to render as raw snake_case, which meant the containment events —
// exactly the ones an evidence review is looking for — read as machine
// strings while the routine ones read as English.
const KIND_LABELS: Record<string, string> = {
  case_opened: "Case opened",
  status_changed: "Status changed",
  case_closed: "Case closed",
  alert_linked: "Alert linked",
  field_updated: "Field updated",
  block_requested: "Block requested",
  ticket_opened: "Ticket opened",
  case_exported: "Case exported",
  priority_escalated: "Priority escalated",
};

function humanize(kind: string): string {
  return KIND_LABELS[kind] ?? kind.replaceAll("_", " ").replace(/^./, (c) => c.toUpperCase());
}

function formatDetail(entry: CaseTimelineEntry): string | null {
  const d = entry.detail;
  if (!d) return null;

  switch (entry.kind) {
    case "status_changed":
      return d.from && d.to ? `${d.from} → ${d.to}${d.reason ? ` — ${d.reason}` : ""}` : null;
    case "case_opened":
      return d.event_id ? `from ${d.source ?? "event"} ${d.event_id}` : null;
    case "case_closed":
      return d.closure_reason ? String(d.closure_reason) : null;
    case "block_requested":
      return d.target ? `target ${d.target}` : null;
    case "ticket_opened":
      return d.ticket_ref ? `${d.ticket_ref} (local mock ticket — adr/0010)` : null;
    case "case_exported":
      return d.redaction_profile ? `redaction: ${d.redaction_profile}` : null;
    case "priority_escalated":
      return d.repeat_count ? `${d.repeat_count} repeats${d.priority ? ` → ${d.priority}` : ""}` : null;
    default:
      return null;
  }
}

const dayKey = (iso: string) => new Date(iso).toDateString();

// services/api/cases writes exactly two actors to the timeline: "analyst" and
// "system". For an operator who is every role at once, the only question at
// review time is which of these they decided and which the pipeline did on
// its own, so provenance gets a dedicated channel rather than being buried in
// a "by …" suffix.
const isHuman = (actor: string) => actor !== "system" && actor !== "scheduler";

export function Timeline({ entries }: { entries: CaseTimelineEntry[] }) {
  const [humanOnly, setHumanOnly] = useState(false);

  const humanCount = useMemo(() => entries.filter((e) => isHuman(e.actor)).length, [entries]);

  const rows = useMemo(() => {
    const visible = humanOnly ? entries.filter((e) => isHuman(e.actor)) : entries;
    // Day separators computed up front rather than by mutating a cursor
    // inside the map: entries showing time-of-day alone make a case spanning
    // two days unreadable.
    return visible.map((entry, i) => ({
      entry,
      detail: formatDetail(entry),
      human: isHuman(entry.actor),
      startsNewDay: i === 0 || dayKey(entry.occurred_at) !== dayKey(visible[i - 1].occurred_at),
    }));
  }, [entries, humanOnly]);

  if (entries.length === 0) {
    return <p className="text-sm text-text-tertiary">No timeline entries yet.</p>;
  }

  return (
    <div className="flex flex-col gap-3">
      {humanCount > 0 && humanCount < entries.length && (
        <button
          type="button"
          onClick={() => setHumanOnly((v) => !v)}
          aria-pressed={humanOnly}
          className={`inline-flex h-7 w-fit items-center rounded-sm border px-2.5 text-2xs font-medium transition-colors ${
            humanOnly
              ? "border-accent bg-accent/10 text-accent"
              : "border-border-strong text-text-secondary hover:text-text-primary"
          }`}
        >
          {humanOnly ? `Showing ${humanCount} human action${humanCount === 1 ? "" : "s"}` : "Human actions only"}
        </button>
      )}

      <ol className="flex flex-col gap-3">
        {rows.map(({ entry, detail, human, startsNewDay }) => (
          <li key={entry.id} className="flex flex-col gap-2">
            {startsNewDay && (
              <div className="flex items-center gap-2 pt-1 first:pt-0">
                <span className="font-mono text-2xs tracking-wide text-text-tertiary uppercase">
                  {new Date(entry.occurred_at).toLocaleDateString([], {
                    month: "short",
                    day: "numeric",
                    year: "numeric",
                  })}
                </span>
                <span aria-hidden="true" className="h-px flex-1 bg-border" />
              </div>
            )}

            <div className="flex items-start gap-3">
              {/* Provenance channel: a filled accent mark for something the
                  operator decided, a hollow one for something the pipeline
                  did by itself. */}
              <span
                aria-hidden="true"
                className={`mt-1.5 h-2 w-2 shrink-0 rounded-full border ${
                  human ? "border-accent bg-accent" : "border-border-strong bg-transparent"
                }`}
              />
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <div className="flex flex-wrap items-baseline gap-x-2">
                  <span className="text-sm font-medium">{humanize(entry.kind)}</span>
                  <span className="text-2xs text-text-tertiary">
                    {human ? `by ${entry.actor}` : `${entry.actor} (automatic)`}
                  </span>
                </div>
                {detail && <span className="text-xs break-words text-text-secondary">{detail}</span>}
              </div>
              <time className="shrink-0 font-mono text-2xs text-text-tertiary" dateTime={entry.occurred_at}>
                {new Date(entry.occurred_at).toLocaleTimeString([], {
                  hour: "2-digit",
                  minute: "2-digit",
                  second: "2-digit",
                })}
              </time>
            </div>
          </li>
        ))}
      </ol>
    </div>
  );
}
