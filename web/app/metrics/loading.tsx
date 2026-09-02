function Tile() {
  return <div className="h-[86px] animate-pulse rounded-md border border-border bg-surface-hover" />;
}

export default function MetricsLoading() {
  return (
    <div className="flex flex-col gap-[22px]">
      <div className="h-6 w-28 animate-pulse rounded bg-surface-hover" />
      <div className="h-[74px] animate-pulse rounded-md border border-border bg-surface-hover" />
      <div className="grid grid-cols-1 gap-[18px] sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Tile key={i} />
        ))}
      </div>
      <div className="h-[74px] animate-pulse rounded-md border border-border bg-surface-hover" />
      <div className="h-[130px] animate-pulse rounded-md border border-border bg-surface-hover" />
    </div>
  );
}
