"use client";

import { useEffect, useId, useRef } from "react";
import { X } from "lucide-react";

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

// A dialog for operations that need more than a click to confirm — Escape
// and a backdrop click both close it; content clicks don't propagate to the
// backdrop.
//
// Focus is trapped and restored because this dialog now carries the app's
// highest-consequence decision (authorizing a write to a live firewall).
// A keyboard user who tabs out of that panel and hits Enter on whatever is
// behind it has no way to know what they just activated.
export function Modal({
  title,
  onClose,
  children,
  tone = "neutral",
  size = "sm",
  dismissible = true,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
  tone?: "neutral" | "danger";
  size?: "sm" | "lg";
  /**
   * False while an irreversible request is in flight. Escape and the backdrop
   * are convenience exits; when the dialog is waiting on a two-call
   * approve+execute sequence, taking one of them unmounts the dialog before
   * the outcome lands and the operator never learns whether the write
   * happened. The close button is disabled by the same flag.
   */
  dismissible?: boolean;
}) {
  const panelRef = useRef<HTMLDivElement>(null);
  const restoreRef = useRef<HTMLElement | null>(null);
  // Generated, not hardcoded: two dialogs can be mounted at once (a receipt
  // still open behind a confirm), and a duplicated id makes aria-labelledby
  // resolve to whichever heading the browser found first — so the dialog
  // announces the wrong title.
  const titleId = useId();

  useEffect(() => {
    restoreRef.current = document.activeElement as HTMLElement | null;
    const panel = panelRef.current;
    // Prefer whatever the content asked for. Focusing the first tabbable
    // element unconditionally meant the close button won over the reason
    // input every dialog marks with autoFocus — the field the operator opened
    // the dialog to type into.
    const preferred =
      panel?.querySelector<HTMLElement>("[autofocus], [data-autofocus]") ??
      panel?.querySelector<HTMLElement>(FOCUSABLE);
    if (preferred) preferred.focus();
    else panel?.focus();

    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        if (dismissible) onClose();
        return;
      }
      if (e.key !== "Tab" || !panel) return;

      const items = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
        (el) => el.offsetParent !== null || el === document.activeElement,
      );
      if (items.length === 0) {
        e.preventDefault();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      restoreRef.current?.focus?.();
    };
  }, [onClose, dismissible]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* The backdrop is a sibling of the panel, not its wrapper. Wrapping
          meant every click inside the dialog bubbled out to the dismiss
          handler, which the panel then had to cancel with a stopPropagation
          click handler — a click listener on a non-interactive element that
          existed only to undo the structure around it. As a sibling it needs
          no such counterweight.

          aria-hidden because it is exactly that: decoration. It carries no
          content, and the dismissal it offers is also on Escape and on the
          labelled close button, so a screen-reader or keyboard user loses
          nothing by not being told it exists. */}
      <div
        aria-hidden="true"
        className="absolute inset-0 bg-black/60"
        onClick={dismissible ? onClose : undefined}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={`relative flex max-h-[85vh] w-full flex-col overflow-y-auto ${size === "lg" ? "max-w-lg" : "max-w-sm"} rounded-lg border bg-surface p-4 shadow-xl ${
          tone === "danger" ? "border-danger/45" : "border-border"
        }`}
      >
        <div className="mb-3 flex items-center justify-between gap-3">
          <h3 id={titleId} className="text-sm font-semibold text-text-primary">
            {title}
          </h3>
          <button
            type="button"
            onClick={onClose}
            disabled={!dismissible}
            aria-label="Close dialog"
            className="shrink-0 rounded p-1 text-text-tertiary hover:bg-surface-raised hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-30"
          >
            <X className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
