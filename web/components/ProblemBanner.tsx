import type { Problem } from "@/lib/api-client/types";

export function ProblemBanner({
  problem,
  fallbackTitle,
  hint,
}: {
  problem?: Problem;
  fallbackTitle: string;
  // Caller-supplied context the API cannot know — e.g. that a 503 from
  // /v1/search means the indexer is not running in this environment rather
  // than that the query was wrong. Without somewhere to put this, such
  // explanations end up in a source comment the operator never sees.
  hint?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1 rounded-md border border-danger/40 bg-danger/[0.08] px-4 py-3.5 text-sm">
      <p className="font-medium text-text-primary">{problem?.title ?? fallbackTitle}</p>
      {problem?.detail && <p className="text-sm text-text-secondary">{problem.detail}</p>}
      {hint && <p className="text-sm text-text-secondary">{hint}</p>}
      {problem && (
        <p className="mt-1 font-mono text-2xs text-text-tertiary">
          {problem.code} &middot; request {problem.request_id}
        </p>
      )}
    </div>
  );
}
