"use client";

import { ProblemBanner } from "@/components/ProblemBanner";

// Next.js strips details (including a custom Error subclass's fields) from
// anything thrown out of a Server Component before it reaches this client
// boundary, so this only ever renders the generic fallback. A Problem's
// code/detail/request_id are shown by catching ApiProblemError directly in
// the page instead (see app/page.tsx) — this boundary is the last resort
// for a genuinely unexpected exception.
export default function QueueError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex flex-col gap-3">
      <ProblemBanner fallbackTitle="Something went wrong loading the alert queue" />
      <button
        onClick={reset}
        className="w-fit rounded-[4px] border border-border-strong px-3 py-1.5 text-[13px] text-text-secondary hover:bg-surface-hover"
      >
        Try again
      </button>
    </div>
  );
}
