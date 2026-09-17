import Link from "next/link";
import type { Case } from "@/lib/api-client/types";
import { StatusBadge } from "./badges";
import { RelativeTime } from "./RelativeTime";
import { NewMarker } from "./NewSinceLastVisit";
import { ChevronRight } from "lucide-react";
import { truncateCaseId } from "@/app/(app)/approvals/format";

function ConfidenceMeter({ confidence }: { confidence: number }) {
  const pct = Math.round(confidence * 100);
  return (
    <div className="flex items-center gap-1.5" role="img" aria-label={`Confidence ${pct}%`}>
      <div aria-hidden="true" className="h-1 w-11 overflow-hidden rounded-full bg-surface-raised">
        <div className="h-full rounded-full bg-text-secondary" style={{ width: `${pct}%` }} />
      </div>
      <span aria-hidden="true" className="font-mono text-2xs text-text-tertiary">{pct}%</span>
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
          <span className="flex items-center gap-1.5 text-sm font-medium">
            <NewMarker iso={item.updated_at ?? item.opened_at} />
            <span className="min-w-0 truncate">{item.title}</span>
          </span>
          <span className="truncate text-2xs text-text-tertiary">{item.category}</span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <StatusBadge status={item.status} />
            <ConfidenceMeter confidence={item.confidence} />
          </div>
          <RelativeTime iso={item.opened_at} className="shrink-0 font-mono text-2xs text-text-tertiary" />
        </div>
      </div>

      <div className="hidden grid-cols-[1fr_100px_128px_180px_20px] items-center gap-4 md:grid">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="flex items-center gap-1.5 text-sm font-medium">
            <NewMarker iso={item.updated_at ?? item.opened_at} />
            <span className="min-w-0 truncate">{item.title}</span>
          </span>
          <span className="truncate text-2xs text-text-tertiary">{item.category}</span>
        </div>
        <div>
          <StatusBadge status={item.status} />
        </div>
        <ConfidenceMeter confidence={item.confidence} />
        <span className="truncate text-right font-mono text-2xs text-text-tertiary">
          {/* truncateCaseId, not CSS `truncate`: a ULID's leading characters
              are a shared timestamp prefix, so clipping from the right — which
              is what the CSS was doing here — leaves a column of near-identical
              strings for anything opened in the same period. The approvals
              table already keeps the tail; this now matches it. */}
          <RelativeTime iso={item.opened_at} /> <span className="text-text-tertiary/50">&middot;</span>{" "}
          <span title={item.id}>{truncateCaseId(item.id)}</span>
        </span>
        <ChevronRight className="h-3.5 w-3.5 text-text-tertiary" aria-hidden="true" />
      </div>
    </Link>
  );
}
