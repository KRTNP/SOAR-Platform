export function EmptyState({ title, body }: { title: string; body?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-1.5 rounded-md border border-dashed border-border-strong px-6 py-14 text-center">
      <p className="text-sm font-medium text-text-secondary">{title}</p>
      {body && <p className="max-w-sm text-[13px] text-text-tertiary">{body}</p>}
    </div>
  );
}
