function SkeletonRow() {
  return <div className="h-[52px] animate-pulse bg-surface-hover" />;
}

export default function QueueLoading() {
  return (
    <div className="flex flex-col gap-5">
      <div className="h-6 w-40 animate-pulse rounded bg-surface-hover" />
      <div className="flex flex-col gap-4">
        {[3, 2].map((rows, i) => (
          <div key={i} className="flex flex-col gap-2">
            <div className="h-3 w-24 animate-pulse rounded bg-surface-hover" />
            <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border">
              {Array.from({ length: rows }).map((_, j) => (
                <SkeletonRow key={j} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
