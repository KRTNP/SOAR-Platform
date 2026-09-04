export default function CaseDetailLoading() {
  return (
    <div className="flex flex-col gap-[22px]">
      <div className="flex flex-col gap-2.5 border-b border-border pb-[22px]">
        <div className="h-3 w-24 animate-pulse rounded bg-surface-hover" />
        <div className="h-5 w-40 animate-pulse rounded bg-surface-hover" />
        <div className="h-6 w-96 animate-pulse rounded bg-surface-hover" />
      </div>
      <div className="grid grid-cols-1 gap-5 md:grid-cols-[1fr_320px]">
        <div className="h-64 animate-pulse rounded-md bg-surface-hover" />
        <div className="h-40 animate-pulse rounded-md bg-surface-hover" />
      </div>
    </div>
  );
}
