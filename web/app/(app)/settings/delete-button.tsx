"use client";

import { useState, useTransition } from "react";
import { Modal } from "@/components/Modal";

// This app's destructive-action UI outside the case detail page — reuses the
// same Modal/danger-button confirm pattern as CloseCaseModal rather than
// inventing a new one.
//
// `consequence` and `confirmText` exist because the ceremony here used to be
// inverted: closing a case demanded a typed reason, while deleting a tenant —
// which invalidates its API key and orphans everything keyed to it — was one
// click behind "This cannot be undone." The heavier consequence now carries
// the heavier gate, and it names what breaks rather than only asserting
// permanence.
export function DeleteButton({
  label,
  onDelete,
  consequence,
  confirmText,
}: {
  label: string;
  onDelete: () => Promise<{ error?: string }>;
  consequence?: string;
  confirmText?: string;
}) {
  const [open, setOpen] = useState(false);
  const [typed, setTyped] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  const gated = Boolean(confirmText);
  const armed = !gated || typed.trim() === confirmText;

  function close() {
    setOpen(false);
    setTyped("");
    setError(null);
  }

  function submit() {
    if (!armed) return;
    setError(null);
    startTransition(async () => {
      try {
        const { error: problem } = await onDelete();
        if (problem) {
          setError(problem);
          return;
        }
        close();
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
        className="rounded-sm px-2 py-1 text-2xs font-medium text-danger/80 transition-colors hover:bg-danger/10 hover:text-danger"
      >
        Delete
      </button>
      {open && (
        <Modal title={`Delete ${label}?`} onClose={close} tone="danger" dismissible={!isPending}>
          <p className="mb-3 text-xs leading-relaxed text-text-secondary">
            {consequence ?? "This cannot be undone."}
          </p>

          {gated && (
            <label className="mb-3 flex flex-col gap-1.5">
              <span className="text-2xs text-text-tertiary">
                Type <code className="font-mono text-text-secondary">{confirmText}</code> to confirm
              </span>
              <input
                autoFocus
                data-autofocus
                value={typed}
                onChange={(e) => setTyped(e.target.value)}
                disabled={isPending}
                autoComplete="off"
                className="h-8 rounded-sm border border-border-strong bg-surface px-2.5 font-mono text-xs text-text-primary focus:border-danger focus:outline-none"
              />
            </label>
          )}

          {error && (
            <p role="alert" className="mb-2 text-2xs font-medium text-danger">
              {error}
            </p>
          )}

          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={close}
              disabled={isPending}
              className="h-8 rounded-sm px-3 text-xs text-text-tertiary hover:text-text-primary"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={submit}
              disabled={isPending || !armed}
              className="inline-flex h-8 items-center gap-1.5 rounded-sm bg-danger px-3 text-xs font-medium text-bg transition-all hover:bg-danger/90 disabled:cursor-not-allowed disabled:opacity-40"
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
