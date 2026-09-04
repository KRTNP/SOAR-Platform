"use client";

import { useState, useTransition } from "react";
import type { Case, CaseStatus, RedactionProfile } from "@/lib/api-client/types";
import { StatusBadge } from "@/components/badges";
import { Modal } from "@/components/Modal";
import {
  transitionCaseAction,
  closeCaseAction,
  openTicketAction,
  requestBlockAction,
  approveActionAction,
  rejectActionAction,
  releaseActionAction,
  exportCaseAction,
} from "./actions";

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

const BLOCKABLE_TYPES = new Set(["ipv4", "ipv6"]);

const ACTION_STATUS_LABEL: Record<string, string> = {
  pending_approval: "Awaiting approval",
  approved: "Approved",
  executed: "Blocked",
  rejected: "Rejected",
  expired: "Expired",
  released: "Released",
};

const REDACTION_PROFILES: RedactionProfile[] = ["standard", "full"];

// --- Shared UI ---

function ActionSpinner({ className = "h-3.5 w-3.5" }: { className?: string }) {
  return (
    <span
      className={`inline-block animate-spin rounded-full border-2 border-current border-t-transparent ${className}`}
      aria-hidden="true"
    />
  );
}

function SectionLabel({ children, badge }: { children: React.ReactNode; badge?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between pb-1">
      <span className="text-[11px] font-semibold tracking-wider text-text-tertiary uppercase">{children}</span>
      {badge}
    </div>
  );
}

function InlineError({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <p role="alert" aria-live="polite" className="text-[11px] font-medium text-danger animate-in fade-in duration-150">
      {message}
    </p>
  );
}

// --- Header action bar: status, ticket, export, close ---
// Consolidates what used to be four separately-bordered stacked panels
// (StatusControl/CloseCaseControl/TicketControl/ExportControl) into one
// compact row next to the case title — status and ticket are single-click,
// close and export open a Modal since both need more than one field.

export function CaseHeaderActions({ caseItem }: { caseItem: Case }) {
  const [status, setStatus] = useState<CaseStatus>(caseItem.status);
  const [isStatusPending, startStatusTransition] = useTransition();
  const [statusError, setStatusError] = useState<string | null>(null);
  const [isTicketPending, startTicketTransition] = useTransition();
  const [ticketError, setTicketError] = useState<string | null>(null);
  const [closeOpen, setCloseOpen] = useState(false);
  const [exportOpen, setExportOpen] = useState(false);

  const closed = caseItem.status === "closed";

  function handleStatusChange(next: CaseStatus) {
    const previous = status;
    setStatus(next);
    setStatusError(null);
    startStatusTransition(async () => {
      try {
        await transitionCaseAction(caseItem.id, next, "");
      } catch {
        setStatus(previous);
        setStatusError("Could not update status.");
      }
    });
  }

  function handleCreateTicket() {
    setTicketError(null);
    startTicketTransition(async () => {
      try {
        await openTicketAction(caseItem.id);
      } catch {
        setTicketError("Could not create ticket.");
      }
    });
  }

  return (
    <div className="flex flex-col items-end gap-1.5">
      <div className="flex flex-wrap items-center justify-end gap-2">
        {closed ? (
          <StatusBadge status="closed" />
        ) : (
          <select
            value={status}
            onChange={(e) => handleStatusChange(e.target.value as CaseStatus)}
            disabled={isStatusPending}
            aria-label="Case status"
            className="h-8 rounded-[4px] border border-border-strong bg-surface-raised px-2.5 text-[12.5px] font-medium text-text-primary capitalize transition-colors focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:opacity-60"
          >
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        )}

        {caseItem.ticket_ref ? (
          <span className="inline-flex h-8 items-center gap-1.5 rounded-[4px] border border-border-strong bg-surface-raised px-2.5 font-mono text-[12px] text-text-secondary">
            {caseItem.ticket_ref}
            <span className="text-text-tertiary capitalize">({caseItem.ticket_status})</span>
          </span>
        ) : (
          <button
            type="button"
            onClick={handleCreateTicket}
            disabled={isTicketPending}
            className="inline-flex h-8 items-center rounded-[4px] border border-border-strong bg-surface-raised px-3 text-[12.5px] font-medium text-text-primary transition-colors hover:bg-surface-hover disabled:cursor-not-allowed disabled:opacity-40"
          >
            {isTicketPending ? "Creating…" : "Create ticket"}
          </button>
        )}

        <button
          type="button"
          onClick={() => setExportOpen(true)}
          className="inline-flex h-8 items-center rounded-[4px] border border-border-strong bg-surface-raised px-3 text-[12.5px] font-medium text-text-primary transition-colors hover:bg-surface-hover"
        >
          Export
        </button>

        {!closed && (
          <button
            type="button"
            onClick={() => setCloseOpen(true)}
            className="inline-flex h-8 items-center rounded-[4px] border border-danger/40 bg-surface-raised px-3 text-[12.5px] font-medium text-danger transition-colors hover:bg-danger hover:text-bg"
          >
            Close case
          </button>
        )}
      </div>
      <InlineError message={statusError ?? ticketError} />

      {closeOpen && <CloseCaseModal caseItem={caseItem} onClose={() => setCloseOpen(false)} />}
      {exportOpen && <ExportModal caseItem={caseItem} onClose={() => setExportOpen(false)} />}
    </div>
  );
}

function CloseCaseModal({ caseItem, onClose }: { caseItem: Case; onClose: () => void }) {
  const [reason, setReason] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  function submit() {
    if (!reason.trim() || isPending) return;
    setError(null);
    startTransition(async () => {
      try {
        await closeCaseAction(caseItem.id, reason.trim());
        onClose();
      } catch {
        setError("Could not close case. Try again.");
      }
    });
  }

  return (
    <Modal title="Close case" onClose={onClose}>
      <p className="mb-3 text-[12px] leading-relaxed text-text-secondary">
        This is a terminal state and cannot be undone. Enter a reason to continue.
      </p>
      <input
        autoFocus
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && submit()}
        placeholder="Closure reason (required)"
        aria-label="Closure reason"
        disabled={isPending}
        className="h-8 w-full rounded-[4px] border border-border-strong bg-surface-raised px-2.5 text-[12.5px] text-text-primary placeholder:text-text-tertiary focus:border-danger focus:outline-none focus:ring-1 focus:ring-danger"
      />
      <InlineError message={error} />
      <div className="mt-3 flex justify-end gap-2">
        <button
          type="button"
          onClick={onClose}
          disabled={isPending}
          className="h-8 rounded-[4px] px-3 text-[12.5px] text-text-tertiary hover:text-text-primary"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={submit}
          disabled={!reason.trim() || isPending}
          className="inline-flex h-8 items-center gap-1.5 rounded-[4px] bg-danger px-3 text-[12.5px] font-medium text-bg transition-all hover:bg-danger/90 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {isPending ? <ActionSpinner className="h-3 w-3" /> : "Confirm close"}
        </button>
      </div>
    </Modal>
  );
}

function ExportModal({ caseItem, onClose }: { caseItem: Case; onClose: () => void }) {
  const [profile, setProfile] = useState<RedactionProfile>("standard");
  const [purpose, setPurpose] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<{ exportId: string; sha256?: string; manifest?: Record<string, unknown> } | null>(
    null,
  );
  const [showManifest, setShowManifest] = useState(false);

  function submit() {
    setError(null);
    startTransition(async () => {
      try {
        const accepted = await exportCaseAction(caseItem.id, profile, purpose);
        setResult({ exportId: accepted.export_id, sha256: accepted.manifest_sha256, manifest: accepted.manifest });
      } catch {
        setError("Could not export manifest. Try again.");
      }
    });
  }

  return (
    <Modal title="Export case manifest" onClose={onClose}>
      {!result ? (
        <div className="flex flex-col gap-2">
          <select
            value={profile}
            onChange={(e) => setProfile(e.target.value as RedactionProfile)}
            aria-label="Redaction profile"
            disabled={isPending}
            className="h-8 rounded-[4px] border border-border-strong bg-surface-raised px-2 text-[12.5px] text-text-primary capitalize focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          >
            {REDACTION_PROFILES.map((p) => (
              <option key={p} value={p}>
                {p} redaction
              </option>
            ))}
          </select>
          <input
            value={purpose}
            onChange={(e) => setPurpose(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && submit()}
            placeholder="Export purpose (optional)"
            aria-label="Export purpose"
            disabled={isPending}
            className="h-8 rounded-[4px] border border-border-strong bg-surface-raised px-2.5 text-[12px] text-text-primary placeholder:text-text-tertiary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
          <InlineError message={error} />
          <div className="mt-1 flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="h-8 rounded-[4px] px-3 text-[12.5px] text-text-tertiary hover:text-text-primary"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={submit}
              disabled={isPending}
              className="inline-flex h-8 items-center gap-1.5 rounded-[4px] bg-accent px-3 text-[12.5px] font-medium text-bg transition-all hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {isPending ? <ActionSpinner className="h-3 w-3" /> : "Generate manifest"}
            </button>
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-2 text-[12px]">
          <div className="flex items-center justify-between">
            <span className="font-mono text-[11.5px] text-text-secondary">{result.exportId}</span>
            <span className="rounded bg-accent/15 px-1.5 py-0.5 text-[10px] font-medium text-accent">Ready</span>
          </div>
          {result.sha256 && (
            <p className="truncate font-mono text-[10.5px] text-text-tertiary" title={result.sha256}>
              sha256:{result.sha256}
            </p>
          )}
          {result.manifest && (
            <div className="flex flex-col gap-1">
              <button
                type="button"
                onClick={() => setShowManifest((v) => !v)}
                className="w-fit text-[11px] font-medium text-accent hover:underline"
              >
                {showManifest ? "▲ Hide manifest" : "▼ View manifest JSON"}
              </button>
              {showManifest && (
                <pre className="max-h-52 overflow-auto rounded border border-border-strong/40 bg-surface-raised p-2 font-mono text-[10px] text-text-secondary">
                  {JSON.stringify(result.manifest, null, 2)}
                </pre>
              )}
            </div>
          )}
          <button
            type="button"
            onClick={onClose}
            className="mt-2 h-8 self-end rounded-[4px] border border-border-strong px-3 text-[12.5px] text-text-primary hover:bg-surface-hover"
          >
            Done
          </button>
        </div>
      )}
    </Modal>
  );
}

// --- Sidebar: network containment (unchanged behavior, still the one
// operationally complex control — request/approve/reject/release moves
// through real states, unlike the quick header actions above, so it stays
// a dedicated card rather than folding into the header). ---

export function ActionControl({ caseItem }: { caseItem: Case }) {
  const [isPending, startTransition] = useTransition();
  const [reason, setReason] = useState("");
  const [isRejecting, setIsRejecting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const action = caseItem.pending_action;

  function submitRequest(targetValue: string) {
    setError(null);
    startTransition(async () => {
      try {
        await requestBlockAction(caseItem.id, targetValue);
      } catch {
        setError("Could not request block. Try again.");
      }
    });
  }

  function submitApprove() {
    if (!action?.approval_id || isPending) return;
    setError(null);
    startTransition(async () => {
      try {
        await approveActionAction(action.approval_id!, action.id, caseItem.id);
      } catch {
        setError("Could not approve action. Try again.");
      }
    });
  }

  function submitReject() {
    if (!action?.approval_id || !reason.trim() || isPending) return;
    setError(null);
    startTransition(async () => {
      try {
        await rejectActionAction(action.approval_id!, reason.trim(), caseItem.id);
        setReason("");
        setIsRejecting(false);
      } catch {
        setError("Could not reject action. Try again.");
      }
    });
  }

  function submitRelease() {
    if (!action || isPending) return;
    setError(null);
    startTransition(async () => {
      try {
        await releaseActionAction(action.id, caseItem.id);
      } catch {
        setError("Could not release block. Try again.");
      }
    });
  }

  if (!action) {
    const targets = (caseItem.observables ?? []).filter((o) => BLOCKABLE_TYPES.has(o.type));
    if (targets.length === 0) return null;

    return (
      <div className="rounded-lg border border-border bg-surface p-4 shadow-xs">
        <SectionLabel>Network Containment</SectionLabel>
        <div className="flex flex-col gap-1.5">
          {targets.map((o) => (
            <button
              key={o.value}
              type="button"
              onClick={() => submitRequest(o.value)}
              disabled={isPending}
              className="inline-flex h-8 items-center justify-between rounded-[4px] border border-border-strong bg-surface-raised px-2.5 text-[12px] font-medium text-text-primary transition-colors hover:border-warn hover:text-warn focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warn disabled:cursor-not-allowed disabled:opacity-40"
            >
              <code className="font-mono">{o.value}</code>
              <span className="text-[11px] font-normal">{isPending ? "Requesting…" : "Request Block →"}</span>
            </button>
          ))}
        </div>
        <InlineError message={error} />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2.5 rounded-lg border border-border bg-surface p-4 shadow-xs">
      <SectionLabel
        badge={
          <span className="rounded-full bg-surface-raised px-2 py-0.5 text-[10.5px] font-medium text-text-secondary">
            {ACTION_STATUS_LABEL[action.status] ?? action.status}
          </span>
        }
      >
        Network Block
      </SectionLabel>

      <div className="rounded bg-surface-raised p-2 border border-border-strong/40 text-[12px]">
        <code className="font-mono font-medium text-text-primary">{action.target.value}</code>
        {action.reason && <p className="mt-1 text-[11px] text-text-tertiary">{action.reason}</p>}
      </div>

      {action.status === "pending_approval" && (
        <div className="flex flex-col gap-2 border-t border-border/60 pt-2">
          {!isRejecting ? (
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={submitApprove}
                disabled={isPending}
                className="inline-flex h-8 flex-1 items-center justify-center gap-1.5 rounded-[4px] bg-accent text-[12.5px] font-medium text-bg transition-all hover:opacity-90 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent disabled:cursor-not-allowed disabled:opacity-40"
              >
                {isPending ? <ActionSpinner /> : "Approve"}
              </button>
              <button
                type="button"
                onClick={() => setIsRejecting(true)}
                disabled={isPending}
                className="inline-flex h-8 flex-1 items-center justify-center rounded-[4px] border border-danger text-[12.5px] font-medium text-danger transition-colors hover:bg-danger/10 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger disabled:cursor-not-allowed disabled:opacity-40"
              >
                Reject…
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-1.5 animate-in fade-in duration-150">
              <input
                autoFocus
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") submitReject();
                  if (e.key === "Escape") setIsRejecting(false);
                }}
                placeholder="Rejection reason (required)"
                aria-label="Rejection reason"
                disabled={isPending}
                className="h-8 rounded-[4px] border border-border-strong bg-surface px-2.5 text-[12px] text-text-primary placeholder:text-text-tertiary focus:border-danger focus:outline-none focus:ring-1 focus:ring-danger"
              />
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={submitReject}
                  disabled={!reason.trim() || isPending}
                  className="inline-flex h-7 flex-1 items-center justify-center rounded-[4px] bg-danger text-[12px] font-medium text-bg hover:bg-danger/90 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  {isPending ? <ActionSpinner className="h-3 w-3" /> : "Confirm Reject"}
                </button>
                <button
                  type="button"
                  onClick={() => setIsRejecting(false)}
                  disabled={isPending}
                  className="inline-flex h-7 px-2.5 items-center justify-center rounded-[4px] border border-border-strong bg-surface text-[12px] text-text-tertiary hover:text-text-primary"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {action.status === "executed" && (
        <button
          type="button"
          onClick={submitRelease}
          disabled={isPending}
          className="inline-flex h-8 items-center justify-center gap-1.5 rounded-[4px] border border-warn text-[12.5px] font-medium text-warn transition-all hover:bg-warn/10 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warn disabled:cursor-not-allowed disabled:opacity-40"
        >
          {isPending ? <ActionSpinner /> : "Release Block Now"}
        </button>
      )}

      <InlineError message={error} />
    </div>
  );
}
