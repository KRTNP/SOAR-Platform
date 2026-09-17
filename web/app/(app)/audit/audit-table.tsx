"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import type { AuditRecord } from "@/lib/api-client/types";
import { DecisionBadge, VerifiedBadge } from "@/components/badges";
import { truncateCaseId } from "@/app/(app)/approvals/format";

// services/api/audit hardcodes List(…, 100) and GET /v1/audit takes no
// paging parameters, so this is the hard ceiling on what any single read can
// return — not a number this client chose.
const AUDIT_CAP = 100;

const WINDOWS = [
  { label: "All time", minutes: 0 },
  { label: "Last 15m", minutes: 15 },
  { label: "Last hour", minutes: 60 },
  { label: "Last 24h", minutes: 60 * 24 },
];

// Filtering happens here rather than server-side because GET /v1/audit takes
// no query parameters (api/openapi.yaml) — this narrows the records the API
// already returned, and the count line says so rather than implying the
// whole log was searched.
export function AuditTable({ records }: { records: AuditRecord[] }) {
  const [query, setQuery] = useState("");
  // The cutoff is pinned when the window is chosen rather than recomputed on
  // every render: a moving boundary would silently drop rows out from under
  // you while you were reading them, and Date.now() in render is impure.
  const [timeFilter, setTimeFilter] = useState<{ minutes: number; cutoff: number | null }>({
    minutes: 0,
    cutoff: null,
  });
  const [problemsOnly, setProblemsOnly] = useState(false);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const cutoff = timeFilter.cutoff;

    return records.filter((r) => {
      if (cutoff !== null && new Date(r.timestamp).getTime() < cutoff) return false;
      if (problemsOnly && r.decision !== "denied" && r.verified) return false;
      if (!q) return true;
      return [r.action, r.actor, r.actor_type, r.case_id, r.reason, r.decision, r.connector_version, r.policy_version]
        .filter(Boolean)
        .some((field) => String(field).toLowerCase().includes(q));
    });
  }, [records, query, timeFilter, problemsOnly]);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Filter by action, actor, case, reason…"
          aria-label="Filter audit records"
          className="h-8 min-w-[220px] flex-1 rounded-sm border border-border-strong bg-surface px-2.5 text-xs text-text-primary placeholder:text-text-tertiary focus:border-accent focus:outline-none"
        />

        <select
          value={timeFilter.minutes}
          onChange={(e) => {
            const minutes = Number(e.target.value);
            setTimeFilter({ minutes, cutoff: minutes > 0 ? Date.now() - minutes * 60_000 : null });
          }}
          aria-label="Time window"
          className="h-8 rounded-sm border border-border-strong bg-surface px-2 text-xs text-text-secondary focus:border-accent focus:outline-none"
        >
          {WINDOWS.map((w) => (
            <option key={w.minutes} value={w.minutes}>
              {w.label}
            </option>
          ))}
        </select>

        {/* One toggle, not two: "denied" and "tampered" are both "a row that
            needs a human to look at it", and they are the only reason to
            scan this log in a hurry. */}
        <button
          type="button"
          onClick={() => setProblemsOnly((v) => !v)}
          aria-pressed={problemsOnly}
          className={`h-8 rounded-sm border px-2.5 text-xs font-medium transition-colors ${
            problemsOnly
              ? "border-danger bg-danger/10 text-danger"
              : "border-border-strong bg-surface text-text-secondary hover:text-text-primary"
          }`}
        >
          Denied or tampered
        </button>
      </div>

      {/* At exactly the cap, "of N records" reads as a total. This is the one
          surface whose whole job is completeness, so it says which it is. */}
      <p className="text-2xs text-text-tertiary">
        Showing {filtered.length} of{" "}
        {records.length >= AUDIT_CAP
          ? `the ${AUDIT_CAP} most recent records — the API returns no more per request`
          : `${records.length} record${records.length === 1 ? "" : "s"} returned by the API`}
        {filtered.length !== records.length && " (filtered in the browser)"}
      </p>

      {filtered.length === 0 ? (
        <p className="rounded-md border border-dashed border-border-strong px-4 py-8 text-center text-sm text-text-tertiary">
          No records match this filter.
        </p>
      ) : (
        <div>
          <p className="mb-1.5 text-2xs text-text-tertiary md:hidden">
            Scroll sideways for decision, actor, case, connector, and reason
          </p>
          <div className="overflow-x-auto rounded-md border border-border">
            <table className="w-full min-w-[880px] border-collapse text-xs">
              <thead>
                <tr className="border-b border-border bg-surface text-left text-2xs font-semibold tracking-wider text-text-tertiary uppercase">
                  <th scope="col" className="px-2.5 py-2.5 font-semibold">Timestamp</th>
                  <th scope="col" className="px-2.5 py-2.5 font-semibold">Action</th>
                  <th scope="col" className="px-2.5 py-2.5 font-semibold">Decision &amp; policy</th>
                  <th scope="col" className="px-2.5 py-2.5 font-semibold">Actor</th>
                  <th scope="col" className="px-2.5 py-2.5 font-semibold">Case</th>
                  <th scope="col" className="px-2.5 py-2.5 font-semibold">Integrity</th>
                  <th scope="col" className="px-2.5 py-2.5 font-semibold">Reason</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {filtered.map((r) => (
                  <tr key={r.id} className="bg-surface hover:bg-surface-hover">
                    <td
                      className="px-2.5 py-2.5 font-mono whitespace-nowrap text-text-tertiary"
                      title={new Date(r.timestamp).toLocaleString()}
                    >
                      {new Date(r.timestamp).toLocaleDateString([], { month: "numeric", day: "numeric" })},{" "}
                      {new Date(r.timestamp).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })}
                    </td>
                    <td className="px-2.5 py-2.5 font-mono whitespace-nowrap text-text-primary">
                      {r.action}
                      {/* Connector version used to be its own mostly-empty
                          column — shown here only when present, next to the
                          action it belongs to. */}
                      {r.connector_version && (
                        <span className="ml-1.5 text-text-tertiary">
                          {r.connector_version.replace(/^connector-/, "")}
                        </span>
                      )}
                    </td>
                    {/* The policy version is the answer to "why was this
                        permitted?" — the product's stated differentiator. It
                        was fetched and typed but rendered nowhere. It sits
                        with the decision it explains rather than taking an
                        eighth column. */}
                    <td className="px-2.5 py-2.5 whitespace-nowrap">
                      <div className="flex items-center gap-1.5">
                        <DecisionBadge decision={r.decision} />
                        <span className="font-mono text-2xs text-text-tertiary" title="Policy version in force">
                          {r.policy_version}
                        </span>
                      </div>
                    </td>
                    <td className="px-2.5 py-2.5 whitespace-nowrap text-text-secondary">
                      {r.actor} <span className="text-text-tertiary">({r.actor_type})</span>
                    </td>
                    <td className="px-2.5 py-2.5 whitespace-nowrap">
                      {r.case_id ? (
                        <Link
                          href={`/cases/${r.case_id}`}
                          className="font-mono text-accent hover:underline"
                          title={r.case_id}
                        >
                          {truncateCaseId(r.case_id)}
                        </Link>
                      ) : (
                        <span className="text-text-tertiary">—</span>
                      )}
                    </td>
                    <td className="px-2.5 py-2.5">
                      <VerifiedBadge verified={r.verified} />
                    </td>
                    <td className="max-w-[220px] px-2.5 py-2.5 text-text-tertiary">{r.reason ?? "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
