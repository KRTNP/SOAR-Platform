function SkeletonRow() {
  return <div className="h-[45px] animate-pulse bg-surface-hover" />;
}

export default function AuditLoading() {
  return (
    <div className="flex flex-col gap-[22px]">
      <div className="h-6 w-32 animate-pulse rounded bg-surface-hover" />
      <div className="flex flex-col divide-y divide-border overflow-hidden rounded-md border border-border">
        {Array.from({ length: 6 }).map((_, i) => (
          <SkeletonRow key={i} />
        ))}
      </div>
    </div>
  );
}
