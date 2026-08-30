"use client";

import { useState, useTransition } from "react";
import type { Case, CaseStatus } from "@/lib/api-client/types";
import { transitionCaseAction, closeCaseAction, openTicketAction } from "./actions";

// "closed" is deliberately excluded: reaching it always goes through the
// dedicated close action below, which enforces a mandatory closure_reason —
// the API rejects a transition straight to closed for the same reason.
const STATUSES: CaseStatus[] = [
  "new",
  "triage",
  "investigating",
  "contained",
  "eradication",
  "recovery",
  "escalated",
  "dismissed",
];

export function StatusControl({ caseItem }: { caseItem: Case }) {
  const [status, setStatus] = useState<CaseStatus>(caseItem.status);
  const [reason, setReason] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  const dirty = status !== caseItem.status;

  function submit() {
    setError(null);
    startTransition(async () => {
      try {
        await transitionCaseAction(caseItem.id, status, reason);
        setReason("");
      } catch {
        setError("Could not update the case status. Try again.");
        setStatus(caseItem.status);
      }
    });
  }

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border bg-surface p-3.5">
      <label className="text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">
        Transition status
      </label>
      <select
        value={status}
        onChange={(e) => setStatus(e.target.value as CaseStatus)}
        className="rounded-[4px] border border-border-strong bg-surface-raised px-2.5 py-1.5 text-[13px] text-text-primary capitalize"
      >
        {STATUSES.map((s) => (
          <option key={s} value={s}>
            {s}
          </option>
        ))}
      </select>
      {dirty && (
        <input
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Reason (optional)"
          className="rounded-[4px] border border-border-strong bg-surface-raised px-2.5 py-1.5 text-[13px] text-text-primary placeholder:text-text-tertiary"
        />
      )}
      <button
        onClick={submit}
        disabled={!dirty || isPending}
        className="rounded-[4px] bg-accent px-3 py-1.5 text-[13px] font-medium text-bg disabled:cursor-not-allowed disabled:opacity-40"
      >
        {isPending ? "Updating…" : "Update"}
      </button>
      {error && <p className="text-[12px] text-danger">{error}</p>}
    </div>
  );
}

export function CloseCaseControl({ caseItem }: { caseItem: Case }) {
  const [closureReason, setClosureReason] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  if (caseItem.status === "closed") {
    return null;
  }

  function submit() {
    setError(null);
    startTransition(async () => {
      try {
        await closeCaseAction(caseItem.id, closureReason);
      } catch {
        setError("Could not close the case. Try again.");
      }
    });
  }

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border bg-surface p-3.5">
      <label className="text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">
        Close case
      </label>
      <input
        value={closureReason}
        onChange={(e) => setClosureReason(e.target.value)}
        placeholder="Closure reason (required)"
        className="rounded-[4px] border border-border-strong bg-surface-raised px-2.5 py-1.5 text-[13px] text-text-primary placeholder:text-text-tertiary"
      />
      <button
        onClick={submit}
        disabled={!closureReason.trim() || isPending}
        className="rounded-[4px] border border-danger px-3 py-1.5 text-[13px] font-medium text-danger disabled:cursor-not-allowed disabled:opacity-40"
      >
        {isPending ? "Closing…" : "Close case"}
      </button>
      {error && <p className="text-[12px] text-danger">{error}</p>}
    </div>
  );
}

export function TicketControl({ caseItem }: { caseItem: Case }) {
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  if (caseItem.ticket_ref) {
    return (
      <div className="flex flex-col gap-1 rounded-md border border-border bg-surface p-3.5">
        <label className="text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">Ticket</label>
        <p className="text-[13px] text-text-primary">
          {caseItem.ticket_ref} <span className="text-text-tertiary capitalize">· {caseItem.ticket_status}</span>
        </p>
      </div>
    );
  }

  function submit() {
    setError(null);
    startTransition(async () => {
      try {
        await openTicketAction(caseItem.id);
      } catch {
        setError("Could not open a ticket. Try again.");
      }
    });
  }

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border bg-surface p-3.5">
      <label className="text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">Ticket</label>
      <button
        onClick={submit}
        disabled={isPending}
        className="rounded-[4px] border border-border-strong px-3 py-1.5 text-[13px] font-medium text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
      >
        {isPending ? "Opening…" : "Open ticket"}
      </button>
      {error && <p className="text-[12px] text-danger">{error}</p>}
    </div>
  );
}
