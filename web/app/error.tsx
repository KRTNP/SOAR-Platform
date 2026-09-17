"use client";

import { ProblemBanner } from "@/components/ProblemBanner";

// Next.js strips details (including a custom Error subclass's fields) from
// anything thrown out of a Server Component before it reaches this client
// boundary, so this only ever renders the generic fallback. A Problem's
// code/detail/request_id are shown by catching ApiProblemError directly in
// the page instead (see app/page.tsx) — this boundary is the last resort
// for a genuinely unexpected exception. It is the fallback for every route
// under app/ that has no more specific error.tsx of its own (today: /,
// /audit, /metrics, /approvals — only /cases/[caseId] overrides it), so its
// copy is deliberately route-agnostic rather than naming one page.
export default function RootError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex flex-col gap-3">
      <ProblemBanner fallbackTitle="Something went wrong loading this page" />
      <button
        onClick={reset}
        className="w-fit rounded-sm border border-border-strong px-3 py-1.5 text-sm text-text-secondary hover:bg-surface-hover"
      >
        Try again
      </button>
    </div>
  );
}
