import type { HTMLAttributes } from "react";

// Base micro-skeleton, motion-reduce aware.
function SkeletonPulse({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      aria-hidden="true"
      className={`animate-pulse rounded bg-surface-hover/80 motion-reduce:animate-none ${className ?? ""}`}
      {...props}
    />
  );
}

// Mirrors one real approval row's shape (badges, intent/target, metadata,
// action buttons) so the loading state doesn't jump when real data arrives.
function SkeletonRow() {
  return (
    <div
      aria-hidden="true"
      className="flex flex-col justify-between gap-3 bg-surface px-4 py-3.5 sm:flex-row sm:items-center sm:gap-4"
    >
      {/* Left: primary content (badges, intent, target, metadata) */}
      <div className="flex flex-1 flex-col gap-2">
        {/* Badges & Case ID Placeholder */}
        <div className="flex items-center gap-2">
          <SkeletonPulse className="h-5 w-16 rounded-full" />
          <SkeletonPulse className="h-5 w-14 rounded-full" />
          <SkeletonPulse className="h-4 w-12 rounded" />
        </div>

        {/* Action Intent & Target Value Placeholder */}
        <div className="flex flex-col gap-1.5 pt-0.5">
          <SkeletonPulse className="h-4 w-2/5 min-w-[140px]" />
          <SkeletonPulse className="h-3.5 w-3/5 min-w-[180px] rounded" />
        </div>

        {/* Metadata (Requester & Expiration) */}
        <div className="flex items-center gap-2 pt-0.5">
          <SkeletonPulse className="h-3 w-20" />
          <span className="text-2xs text-border">•</span>
          <SkeletonPulse className="h-3 w-24" />
        </div>
      </div>

      {/* Right: action buttons (approve / reject) */}
      <div className="flex items-center gap-1.5 pt-2 sm:pt-0">
        <SkeletonPulse className="h-7 w-16 rounded-sm" />
        <SkeletonPulse className="h-7 w-16 rounded-sm" />
      </div>
    </div>
  );
}

export default function ApprovalsLoading() {
  return (
    <section
      role="status"
      aria-busy="true"
      aria-label="Loading approval queue"
      className="flex flex-col gap-6"
    >
      {/* Screen Reader Only Notification */}
      <span className="sr-only">Loading approvals queue, please wait...</span>

      {/* Page Title Placeholder */}
      <div className="flex items-center justify-between">
        <SkeletonPulse className="h-6 w-32 rounded-md" />
        <SkeletonPulse className="h-4 w-16 rounded" />
      </div>

      {/* Approval Groups Placeholder */}
      <div className="flex flex-col gap-5">
        {[2, 3].map((rowCount, groupIdx) => (
          <div key={groupIdx} className="flex flex-col gap-2.5">
            {/* Group Header Label */}
            <div className="flex items-center gap-2">
              <SkeletonPulse className="h-3.5 w-24 rounded" />
              <SkeletonPulse className="h-3 w-6 rounded-full" />
            </div>

            {/* List / Table Shell */}
            <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border bg-surface shadow-xs">
              {Array.from({ length: rowCount }).map((_, rowIdx) => (
                <SkeletonRow key={rowIdx} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}