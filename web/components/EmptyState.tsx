import Link from "next/link";

// An empty state that only says "nothing here" is a dead end. In an
// episodic lab session the useful answer is where to go next — usually
// another surface that does have something, or the thing you run to make
// this page fill up.
export function EmptyState({
  title,
  body,
  action,
}: {
  title: string;
  body?: string;
  action?: { label: string; href: string };
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border-strong px-6 py-14 text-center">
      <p className="text-sm font-medium text-text-secondary">{title}</p>
      {body && <p className="max-w-sm text-sm text-text-tertiary">{body}</p>}
      {action && (
        <Link
          href={action.href}
          className="mt-1.5 inline-flex h-8 items-center rounded-sm border border-border-strong bg-surface-raised px-3 text-xs font-medium text-text-primary transition-colors hover:bg-surface-hover"
        >
          {action.label}
        </Link>
      )}
    </div>
  );
}
