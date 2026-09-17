import type { Problem } from "@/lib/api-client/types";

export function ProblemBanner({ problem, fallbackTitle }: { problem?: Problem; fallbackTitle: string }) {
  return (
    <div className="flex flex-col gap-1 rounded-md border border-danger/40 bg-danger/[0.08] px-4 py-3.5 text-sm">
      <p className="font-medium text-text-primary">{problem?.title ?? fallbackTitle}</p>
      {problem?.detail && <p className="text-sm text-text-secondary">{problem.detail}</p>}
      {problem && (
        <p className="mt-1 font-mono text-2xs text-text-tertiary">
          {problem.code} &middot; request {problem.request_id}
        </p>
      )}
    </div>
  );
}
