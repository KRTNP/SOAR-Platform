import type { HTMLAttributes } from "react";

function SkeletonPulse({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      aria-hidden="true"
      className={`animate-pulse rounded bg-surface-hover/80 motion-reduce:animate-none ${className ?? ""}`}
      {...props}
    />
  );
}

// Settings makes three independent server calls; without this the route gave
// no busy state at all while the other routes did.
function SkeletonTable({ rows }: { rows: number }) {
  return (
    <div className="flex flex-col gap-3">
      <SkeletonPulse className="h-3.5 w-32" />
      <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="flex items-center justify-between gap-4 bg-surface px-3 py-2.5">
            <SkeletonPulse className="h-3 w-48" />
            <SkeletonPulse className="h-3 w-20" />
          </div>
        ))}
      </div>
    </div>
  );
}

export default function SettingsLoading() {
  return (
    <div className="flex flex-col gap-6" role="status" aria-busy="true" aria-label="Loading settings">
      <header className="flex flex-col gap-2">
        <SkeletonPulse className="h-6 w-32" />
        <SkeletonPulse className="h-3 w-full max-w-xl" />
      </header>
      <SkeletonTable rows={2} />
      <SkeletonTable rows={3} />
      <SkeletonTable rows={2} />
    </div>
  );
}
