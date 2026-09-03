"use client";

import { useState, useTransition, useRef, useEffect } from "react";
import { approveActionAction, rejectActionAction } from "@/app/cases/[caseId]/actions";

interface ApprovalRowActionsProps {
  approvalId: string;
  actionId: string;
  caseId: string;
  size?: "sm" | "md";
}

export function ApprovalRowActions({
  approvalId,
  actionId,
  caseId,
  size = "sm",
}: ApprovalRowActionsProps) {
  const [isPending, startTransition] = useTransition();
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);

  const inputRef = useRef<HTMLInputElement>(null);

  // "sm" fits a dense table cell; "md" gives the mobile card a real touch
  // target instead of the same compressed row-scale button.
  const isSm = size === "sm";
  const btnBase = isSm
    ? "h-7 px-2.5 text-[12px] min-w-[64px]"
    : "h-9 px-3.5 text-[13px] min-w-[76px]";
  const inputHeight = isSm ? "h-7 text-[12px]" : "h-9 text-[13px]";

  // Auto-focus the reason input once rejecting mode opens.
  useEffect(() => {
    if (rejecting) {
      inputRef.current?.focus();
    }
  }, [rejecting]);

  function handleCancelReject() {
    setRejecting(false);
    setReason("");
    setError(null);
  }

  function submitApprove() {
    setError(null);
    startTransition(async () => {
      try {
        await approveActionAction(approvalId, actionId, caseId);
      } catch {
        setError("Could not approve. Try again.");
      }
    });
  }

  function submitReject() {
    if (!reason.trim() || isPending) return;
    setError(null);
    startTransition(async () => {
      try {
        await rejectActionAction(approvalId, reason.trim(), caseId);
      } catch {
        setError("Could not reject. Try again.");
      }
    });
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault();
      submitReject();
    } else if (e.key === "Escape") {
      e.preventDefault();
      handleCancelReject();
    }
  }

  return (
    <div className="relative inline-flex flex-col items-start gap-1">
      {rejecting ? (
        /* Rejection Input Mode */
        <form
          onSubmit={(e) => {
            e.preventDefault();
            submitReject();
          }}
          className="flex items-center gap-1.5 transition-all duration-150 ease-out"
        >
          <div className="relative">
            <input
              ref={inputRef}
              type="text"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Reason (required)"
              aria-label="Rejection reason"
              aria-invalid={!!error}
              disabled={isPending}
              className={`w-44 sm:w-56 rounded-[4px] border border-border-strong bg-surface-raised px-2.5 ${inputHeight} text-text-primary placeholder:text-text-tertiary transition-colors duration-150 focus:border-danger focus:outline-none focus:ring-1 focus:ring-danger disabled:cursor-not-allowed disabled:opacity-50`}
            />
          </div>

          <button
            type="submit"
            disabled={!reason.trim() || isPending}
            aria-label="Confirm rejection"
            className={`inline-flex items-center justify-center rounded-[4px] border border-danger bg-danger/10 ${btnBase} font-medium text-danger whitespace-nowrap transition-all duration-150 hover:bg-danger hover:text-bg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger focus-visible:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-danger/10 disabled:hover:text-danger`}
          >
            {isPending ? (
              <span className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
            ) : (
              "Confirm"
            )}
          </button>

          <button
            type="button"
            onClick={handleCancelReject}
            disabled={isPending}
            aria-label="Cancel rejection"
            className={`inline-flex items-center justify-center rounded-[4px] border border-border-strong bg-surface-raised px-2 ${isSm ? "h-7 text-[12px]" : "h-9 text-[13px]"} text-text-tertiary transition-colors duration-150 hover:bg-surface-raised/80 hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-border-strong disabled:cursor-not-allowed disabled:opacity-40`}
          >
            Cancel
          </button>
        </form>
      ) : (
        /* Default Action Buttons */
        <div className="flex items-center gap-1.5 transition-all duration-150 ease-out">
          <button
            type="button"
            onClick={submitApprove}
            disabled={isPending}
            aria-label="Approve case"
            className={`inline-flex items-center justify-center rounded-[4px] bg-accent ${btnBase} font-medium text-bg whitespace-nowrap transition-all duration-150 hover:opacity-90 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-40 disabled:active:scale-100`}
          >
            {isPending ? (
              <span className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
            ) : (
              "Approve"
            )}
          </button>

          <button
            type="button"
            onClick={() => setRejecting(true)}
            disabled={isPending}
            aria-label="Reject case"
            className={`inline-flex items-center justify-center rounded-[4px] border border-danger/80 ${btnBase} font-medium text-danger whitespace-nowrap transition-all duration-150 hover:bg-danger/10 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger focus-visible:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-40 disabled:active:scale-100`}
          >
            Reject
          </button>
        </div>
      )}

      {/* Reserves its own line so an error never shifts the row's layout */}
      {error && (
        <span
          role="alert"
          aria-live="polite"
          className="text-[11px] font-medium text-danger animate-in fade-in slide-in-from-top-0.5 duration-150"
        >
          {error}
        </span>
      )}
    </div>
  );
}