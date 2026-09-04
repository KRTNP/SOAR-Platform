"use client";

import { useState, useTransition } from "react";
import { Modal } from "@/components/Modal";

// This app's first destructive-action UI outside the case detail page —
// reuses the same Modal/danger-button confirm pattern as CloseCaseModal
// (app/cases/[caseId]/status-control.tsx) rather than inventing a new one.
export function DeleteButton({ label, onDelete }: { label: string; onDelete: () => Promise<void> }) {
  const [open, setOpen] = useState(false);
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  function submit() {
    setError(null);
    startTransition(async () => {
      try {
        await onDelete();
        setOpen(false);
      } catch {
        setError("Could not delete. Try again.");
      }
    });
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={`Delete ${label}`}
        className="rounded-[4px] px-2 py-1 text-[11.5px] font-medium text-danger/80 transition-colors hover:bg-danger/10 hover:text-danger"
      >
        Delete
      </button>
      {open && (
        <Modal title={`Delete ${label}?`} onClose={() => setOpen(false)}>
          <p className="mb-3 text-[12px] leading-relaxed text-text-secondary">This cannot be undone.</p>
          {error && (
            <p role="alert" aria-live="polite" className="mb-2 text-[11px] font-medium text-danger">
              {error}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setOpen(false)}
              disabled={isPending}
              className="h-8 rounded-[4px] px-3 text-[12.5px] text-text-tertiary hover:text-text-primary"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={submit}
              disabled={isPending}
              className="inline-flex h-8 items-center gap-1.5 rounded-[4px] bg-danger px-3 text-[12.5px] font-medium text-bg transition-all hover:bg-danger/90 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {isPending ? (
                <span className="inline-block h-3 w-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
              ) : (
                "Confirm delete"
              )}
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}
