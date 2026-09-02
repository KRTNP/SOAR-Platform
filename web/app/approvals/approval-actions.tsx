"use client";

import { useState, useTransition } from "react";
import { approveActionAction, rejectActionAction } from "@/app/cases/[caseId]/actions";

// The whole point of a queue page is deciding without opening each case —
// a version of this that only linked out to status-control.tsx's controls
// would leave the same "nothing here actually does anything" gap it exists
// to fix. Mirrors that file's approve/reject behavior (approve performs
// both M3 steps; reject requires a non-empty reason) at row scale.
export function ApprovalRowActions({
  approvalId,
  actionId,
  caseId,
  size = "sm",
}: {
  approvalId: string;
  actionId: string;
  caseId: string;
  // "sm" fits a dense table cell; "md" gives the mobile card a real touch
  // target instead of the same compressed row-scale button.
  size?: "sm" | "md";
}) {
  const [isPending, startTransition] = useTransition();
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const pad = size === "md" ? "px-3 py-1.5" : "px-2.5 py-1";
  const text = size === "md" ? "text-[13px]" : "text-[12px]";

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
    if (!reason.trim()) return;
    setError(null);
    startTransition(async () => {
      try {
        await rejectActionAction(approvalId, reason, caseId);
      } catch {
        setError("Could not reject. Try again.");
      }
    });
  }

  if (rejecting) {
    return (
      <div className="flex items-center gap-1.5">
        <input
          autoFocus
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Reason (required)"
          aria-label="Rejection reason"
          className={`min-w-0 flex-1 rounded-[4px] border border-border-strong bg-surface-raised px-2 py-1 ${text} text-text-primary placeholder:text-text-tertiary`}
        />
        <button
          onClick={submitReject}
          disabled={!reason.trim() || isPending}
          className={`rounded-[4px] border border-danger ${pad} ${text} font-medium text-danger whitespace-nowrap disabled:cursor-not-allowed disabled:opacity-40`}
        >
          {isPending ? "…" : "Confirm"}
        </button>
        {error && <span className="text-[11px] text-danger">{error}</span>}
      </div>
    );
  }

  return (
    <div className="flex items-center gap-1.5">
      <button
        onClick={submitApprove}
        disabled={isPending}
        className={`rounded-[4px] bg-accent ${pad} ${text} font-medium whitespace-nowrap text-bg disabled:cursor-not-allowed disabled:opacity-40`}
      >
        {isPending ? "Approving…" : "Approve"}
      </button>
      <button
        onClick={() => setRejecting(true)}
        disabled={isPending}
        className={`rounded-[4px] border border-danger ${pad} ${text} font-medium whitespace-nowrap text-danger disabled:cursor-not-allowed disabled:opacity-40`}
      >
        Reject
      </button>
      {error && <span className="text-[11px] text-danger">{error}</span>}
    </div>
  );
}
