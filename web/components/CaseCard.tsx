import Link from "next/link";
import type { Case } from "@/lib/api-client/types";
import { StatusBadge } from "./badges";

function relativeTime(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const minutes = Math.round(diffMs / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

function ConfidenceMeter({ confidence }: { confidence: number }) {
  return (
    <div className="flex items-center gap-1.5">
      <div className="h-1 w-11 overflow-hidden rounded-full bg-surface-raised">
        <div className="h-full rounded-full bg-text-secondary" style={{ width: `${Math.round(confidence * 100)}%` }} />
      </div>
      <span className="font-mono text-[11px] text-text-tertiary">{Math.round(confidence * 100)}%</span>
    </div>
  );
}

// Two compositions of the same data, not one grid shrunk to fit: below md
// this is a stacked card (title first, then status/confidence/meta
// wrapping freely); md and up is a dense table row. Tailwind's JIT scanner
// needs the "hidden"/"md:grid" etc. as literal classes on their own
// wrappers, so the row is duplicated rather than conditionally computed.
//
// No per-row severity badge: CaseListGroup already buckets cases by
// severity under one section header, so repeating it on every card inside
// that section is pure redundancy — the group header carries that signal.
// The case id moves from a leading column to trailing metadata next to the
// timestamp; it's a reference string an analyst copies, not the first
// thing that should catch the eye in a severity-sorted triage list.
export function CaseCard({ item }: { item: Case }) {
  return (
    <Link href={`/cases/${item.id}`} className="block bg-surface px-4 py-3 hover:bg-surface-hover">
      <div className="flex flex-col gap-2 md:hidden">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="text-[13.5px] font-medium">{item.title}</span>
          <span className="truncate text-[11.5px] text-text-tertiary">{item.category}</span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <StatusBadge status={item.status} />
            <ConfidenceMeter confidence={item.confidence} />
          </div>
          <span className="shrink-0 font-mono text-[11px] text-text-tertiary">{relativeTime(item.opened_at)}</span>
        </div>
      </div>

      <div className="hidden grid-cols-[1fr_100px_128px_180px_20px] items-center gap-4 md:grid">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="truncate text-[13.5px] font-medium">{item.title}</span>
          <span className="truncate text-[11.5px] text-text-tertiary">{item.category}</span>
        </div>
        <div>
          <StatusBadge status={item.status} />
        </div>
        <ConfidenceMeter confidence={item.confidence} />
        <span className="truncate text-right font-mono text-[11px] text-text-tertiary">
          {relativeTime(item.opened_at)} <span className="text-text-tertiary/50">&middot;</span> {item.id}
        </span>
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.8}
          strokeLinecap="round"
          strokeLinejoin="round"
          className="h-3.5 w-3.5 text-text-tertiary"
        >
          <polyline points="9 6 15 12 9 18" />
        </svg>
      </div>
    </Link>
  );
}
