"use client";

import { useState, useTransition, useRef, useEffect } from "react";
import {
  approveActionAction,
  executeActionAction,
  rejectActionAction,
  releaseActionAction,
} from "@/app/(app)/cases/[caseId]/actions";
import { ApproveDialog } from "@/components/ApproveDialog";
import type { PendingAction } from "@/lib/api-client/types";

interface ApprovalRowActionsProps {
  approvalId: string;
  action: PendingAction;
  expiresAt?: string;
  size?: "sm" | "md";
}

export function ApprovalRowActions({ approvalId, action, expiresAt, size = "sm" }: ApprovalRowActionsProps) {
  const [isPending, startTransition] = useTransition();
  const [confirming, setConfirming] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [failedStage, setFailedStage] = useState<"approve" | "execute" | undefined>();
  const [executed, setExecuted] = useState<PendingAction | undefined>();
  const [rejected, setRejected] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);

  // "sm" fits a dense table cell; "md" gives the mobile card a real touch
  // target instead of the same compressed row-scale button.
  const isSm = size === "sm";
  const btnBase = isSm ? "h-7 px-2.5 text-xs min-w-[64px]" : "h-9 px-3.5 text-sm min-w-[76px]";
  const inputHeight = isSm ? "h-7 text-xs" : "h-9 text-sm";

  useEffect(() => {
    if (rejecting) inputRef.current?.focus();
  }, [rejecting]);

  function closeDialog() {
    setConfirming(false);
    setError(null);
    setFailedStage(undefined);
    setExecuted(undefined);
  }

  function handleCancelReject() {
    setRejecting(false);
    setReason("");
    setError(null);
  }

  function submitApprove() {
    setError(null);
    setFailedStage(undefined);
    startTransition(async () => {
      try {
        const result = await approveActionAction(approvalId, action.id, action.case_id);
        if (result.error) {
          setError(result.error);
          setFailedStage(result.failedStage);
          return;
        }
        // Undefined when the confirmation read failed: the dialog then closes
        // rather than asserting a state nobody confirmed.
        if (result.executed) setExecuted(result.executed);
        else closeDialog();
      } catch {
        setError("Could not approve. Try again.");
      }
    });
  }

  function retryExecute() {
    setError(null);
    startTransition(async () => {
      try {
        const result = await executeActionAction(action.id, action.case_id);
        if (result.error) {
          setError(result.error);
          setFailedStage(result.failedStage);
          return;
        }
        setFailedStage(undefined);
        if (result.executed) setExecuted(result.executed);
        else closeDialog();
      } catch {
        setError("Could not execute. Try again.");
      }
    });
  }

  function releaseNow() {
    setError(null);
    startTransition(async () => {
      try {
        const result = await releaseActionAction(action.id, action.case_id);
        if (result.error) setError(result.error);
        else closeDialog();
      } catch {
        setError("Could not release. Try again.");
      }
    });
  }

  function submitReject() {
    if (!reason.trim() || isPending) return;
    setError(null);
    startTransition(async () => {
      try {
        const result = await rejectActionAction(approvalId, reason.trim(), action.case_id);
        if (result.error) {
          setError(result.error);
          return;
        }
        // Approving ends in a receipt; rejecting used to end in the row
        // silently vanishing. Both halves of the same decision deserve a
        // confirmation that it was recorded.
        setRejected(true);
        setRejecting(false);
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
      {confirming && (
        <ApproveDialog
          action={action}
          expiresAt={expiresAt}
          onConfirm={submitApprove}
          onCancel={closeDialog}
          isPending={isPending}
          error={error}
          failedStage={failedStage}
          executed={executed}
          onRetryExecute={retryExecute}
          onRelease={releaseNow}
        />
      )}

      {rejected ? (
        <span className="inline-flex items-center gap-1.5 text-xs font-medium text-text-secondary" role="status">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth={2}
            strokeLinecap="round"
            strokeLinejoin="round"
            className="h-3.5 w-3.5 text-safe"
            aria-hidden="true"
          >
            <polyline points="5 12.5 9.5 17 19 6.5" />
          </svg>
          Rejection recorded
        </span>
      ) : rejecting ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            submitReject();
          }}
          className="flex items-center gap-1.5"
        >
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
            className={`w-44 rounded-sm border border-border-strong bg-surface-raised px-2.5 sm:w-56 ${inputHeight} text-text-primary transition-colors duration-150 placeholder:text-text-tertiary focus:border-danger focus:ring-1 focus:ring-danger focus:outline-none disabled:cursor-not-allowed disabled:opacity-50`}
          />

          <button
            type="submit"
            disabled={!reason.trim() || isPending}
            aria-label="Confirm rejection"
            className={`inline-flex items-center justify-center rounded-sm border border-danger bg-danger/10 ${btnBase} font-medium whitespace-nowrap text-danger transition-all duration-150 hover:bg-danger hover:text-bg disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-danger/10 disabled:hover:text-danger`}
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
            className={`inline-flex items-center justify-center rounded-sm border border-border-strong bg-surface-raised px-2 ${
              isSm ? "h-7 text-xs" : "h-9 text-sm"
            } text-text-tertiary transition-colors duration-150 hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-40`}
          >
            Cancel
          </button>
        </form>
      ) : (
        <div className="flex items-center gap-1.5">
          {/* Opens the consequence sheet rather than approving: the mode,
              target, predicted effect and rollback path have to be on screen
              before this decision is reachable (PRODUCT.md principle 4). */}
          <button
            type="button"
            onClick={() => setConfirming(true)}
            disabled={isPending}
            aria-haspopup="dialog"
            aria-label={`Review and approve ${action.intent} on ${action.target.value}`}
            className={`inline-flex items-center justify-center gap-1.5 rounded-sm ${btnBase} font-medium whitespace-nowrap transition-all duration-150 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-40 disabled:active:scale-100 ${
              action.mode === "execute"
                ? "border border-danger bg-danger/10 text-danger hover:bg-danger hover:text-bg"
                : "bg-accent text-bg hover:opacity-90"
            }`}
          >
            Review…
          </button>

          <button
            type="button"
            onClick={() => setRejecting(true)}
            disabled={isPending}
            aria-label={`Reject ${action.intent} on ${action.target.value}`}
            className={`inline-flex items-center justify-center rounded-sm border border-danger/80 ${btnBase} font-medium whitespace-nowrap text-danger transition-all duration-150 hover:bg-danger/10 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-40 disabled:active:scale-100`}
          >
            Reject
          </button>
        </div>
      )}

      {/* Reserves its own line so an error never shifts the row's layout.
          While the dialog is open it owns the error instead. */}
      {error && !confirming && (
        <span role="alert" className="text-2xs font-medium text-danger">
          {error}
        </span>
      )}
    </div>
  );
}
