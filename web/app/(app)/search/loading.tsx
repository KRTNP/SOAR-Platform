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

export default function SearchLoading() {
  return (
    <div className="flex flex-col gap-5" role="status" aria-busy="true" aria-label="Searching">
      <header className="flex flex-col gap-2">
        <SkeletonPulse className="h-6 w-24" />
        <SkeletonPulse className="h-3 w-full max-w-lg" />
      </header>
      <SkeletonPulse className="h-8 w-full max-w-md" />
      <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex flex-col gap-2 bg-surface px-3 py-3">
            <SkeletonPulse className="h-3 w-40" />
            <SkeletonPulse className="h-3 w-full max-w-xl" />
          </div>
        ))}
      </div>
    </div>
  );
}
