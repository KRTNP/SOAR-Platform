"use client";

import { useState, useTransition } from "react";
import type { Case, CaseStatus, RedactionProfile } from "@/lib/api-client/types";
import { MockMarker, ModeBadge, StatusBadge } from "@/components/badges";
import { Modal } from "@/components/Modal";
import { ApprovalRowActions } from "@/app/(app)/approvals/approval-actions";
import {
  transitionCaseAction,
  closeCaseAction,
  openTicketAction,
  requestBlockAction,
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
      <span className="text-2xs font-semibold tracking-wider text-text-tertiary uppercase">{children}</span>
      {badge}
    </div>
  );
}

function InlineError({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <p role="alert" className="text-2xs font-medium text-danger animate-in fade-in duration-150">
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
  const [staged, setStaged] = useState<CaseStatus | null>(null);
  const [statusReason, setStatusReason] = useState("");
  const [isStatusPending, startStatusTransition] = useTransition();
  const [statusError, setStatusError] = useState<string | null>(null);
  const [isTicketPending, startTicketTransition] = useTransition();
  const [ticketError, setTicketError] = useState<string | null>(null);
  const [closeOpen, setCloseOpen] = useState(false);
  const [exportOpen, setExportOpen] = useState(false);

  const closed = caseItem.status === "closed";

  // Staged, not committed on change. A native <select> fires change on an
  // arrow key or a scroll wheel passing over it, so committing there meant a
  // stray gesture could move a contained case back to investigating and write
  // that to the audit log with an empty reason. The transition now needs a
  // second, deliberate act.
  function applyStatus() {
    if (staged === null || staged === status) return;
    const next = staged;
    const previous = status;
    setStatus(next);
    setStatusError(null);
    startStatusTransition(async () => {
      try {
        const result = await transitionCaseAction(caseItem.id, next, statusReason.trim());
        if (result.error) {
          setStatus(previous);
          setStatusError(result.error);
          return;
        }
        setStaged(null);
        setStatusReason("");
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
        const result = await openTicketAction(caseItem.id);
        if (result.error) setTicketError(result.error);
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
          <div className="flex flex-wrap items-center justify-end gap-1.5">
            <select
              value={staged ?? status}
              onChange={(e) => setStaged(e.target.value as CaseStatus)}
              disabled={isStatusPending}
              aria-label="Case status"
              className="h-8 rounded-sm border border-border-strong bg-surface-raised px-2.5 text-xs font-medium text-text-primary capitalize transition-colors focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none disabled:opacity-60"
            >
              {STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>

            {staged !== null && staged !== status && (
              <>
                <input
                  value={statusReason}
                  onChange={(e) => setStatusReason(e.target.value)}
                  placeholder="Reason (optional)"
                  aria-label="Reason for status change"
                  disabled={isStatusPending}
                  className="h-8 w-40 rounded-sm border border-border-strong bg-surface px-2.5 text-xs text-text-primary placeholder:text-text-tertiary focus:border-accent focus:outline-none"
                />
                <button
                  type="button"
                  onClick={applyStatus}
                  disabled={isStatusPending}
                  className="inline-flex h-8 items-center rounded-sm bg-accent px-2.5 text-xs font-semibold text-bg hover:opacity-90 disabled:opacity-40"
                >
                  {isStatusPending ? <ActionSpinner className="h-3 w-3" /> : `Set ${staged}`}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setStaged(null);
                    setStatusReason("");
                  }}
                  disabled={isStatusPending}
                  className="inline-flex h-8 items-center rounded-sm border border-border-strong px-2 text-xs text-text-tertiary hover:text-text-primary disabled:opacity-40"
                >
                  Cancel
                </button>
              </>
            )}
          </div>
        )}

        {caseItem.ticket_ref ? (
          <span className="inline-flex h-8 items-center gap-1.5 rounded-sm border border-dashed border-border-strong bg-surface-raised px-2.5 font-mono text-xs text-text-secondary">
            {caseItem.ticket_ref}
            <span className="text-text-tertiary capitalize">({caseItem.ticket_status})</span>
            <MockMarker title="Local mock ticket (adr/0010) — never reaches an external tracker" />
          </span>
        ) : (
          <button
            type="button"
            onClick={handleCreateTicket}
            disabled={isTicketPending}
            className="inline-flex h-8 items-center rounded-sm border border-border-strong bg-surface-raised px-3 text-xs font-medium text-text-primary transition-colors hover:bg-surface-hover disabled:cursor-not-allowed disabled:opacity-40"
          >
            {isTicketPending ? "Creating…" : "Create ticket"}
          </button>
        )}

        <button
          type="button"
          onClick={() => setExportOpen(true)}
          className="inline-flex h-8 items-center rounded-sm border border-border-strong bg-surface-raised px-3 text-xs font-medium text-text-primary transition-colors hover:bg-surface-hover"
        >
          Export
        </button>

        {!closed && (
          <button
            type="button"
            onClick={() => setCloseOpen(true)}
            className="inline-flex h-8 items-center rounded-sm border border-danger/40 bg-surface-raised px-3 text-xs font-medium text-danger transition-colors hover:bg-danger hover:text-bg"
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
        const result = await closeCaseAction(caseItem.id, reason.trim());
        if (result.error) setError(result.error);
        else onClose();
      } catch {
        setError("Could not close case. Try again.");
      }
    });
  }

  return (
    <Modal title="Close case" onClose={onClose} dismissible={!isPending}>
      <p className="mb-3 text-xs leading-relaxed text-text-secondary">
        This is a terminal state and cannot be undone. Enter a reason to continue.
      </p>
      <input
        autoFocus
        data-autofocus
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && submit()}
        placeholder="Closure reason (required)"
        aria-label="Closure reason"
        disabled={isPending}
        className="h-8 w-full rounded-sm border border-border-strong bg-surface-raised px-2.5 text-xs text-text-primary placeholder:text-text-tertiary focus:border-danger focus:outline-none focus:ring-1 focus:ring-danger"
      />
      <InlineError message={error} />
      <div className="mt-3 flex justify-end gap-2">
        <button
          type="button"
          onClick={onClose}
          disabled={isPending}
          className="h-8 rounded-sm px-3 text-xs text-text-tertiary hover:text-text-primary"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={submit}
          disabled={!reason.trim() || isPending}
          className="inline-flex h-8 items-center gap-1.5 rounded-sm bg-danger px-3 text-xs font-medium text-bg transition-all hover:bg-danger/90 disabled:cursor-not-allowed disabled:opacity-40"
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
        const { result: accepted, error: problem } = await exportCaseAction(caseItem.id, profile, purpose);
        if (problem || !accepted) {
          setError(problem ?? "Could not export manifest. Try again.");
          return;
        }
        setResult({ exportId: accepted.export_id, sha256: accepted.manifest_sha256, manifest: accepted.manifest });
      } catch {
        setError("Could not export manifest. Try again.");
      }
    });
  }

  return (
    <Modal title="Export case manifest" onClose={onClose} dismissible={!isPending}>
      {!result ? (
        <div className="flex flex-col gap-2">
          <select
            value={profile}
            onChange={(e) => setProfile(e.target.value as RedactionProfile)}
            aria-label="Redaction profile"
            disabled={isPending}
            className="h-8 rounded-sm border border-border-strong bg-surface-raised px-2 text-xs text-text-primary capitalize focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
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
            className="h-8 rounded-sm border border-border-strong bg-surface-raised px-2.5 text-xs text-text-primary placeholder:text-text-tertiary focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
          <InlineError message={error} />
          <div className="mt-1 flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="h-8 rounded-sm px-3 text-xs text-text-tertiary hover:text-text-primary"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={submit}
              disabled={isPending}
              className="inline-flex h-8 items-center gap-1.5 rounded-sm bg-accent px-3 text-xs font-medium text-bg transition-all hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {isPending ? <ActionSpinner className="h-3 w-3" /> : "Generate manifest"}
            </button>
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-2 text-xs">
          <div className="flex items-center justify-between">
            <span className="font-mono text-2xs text-text-secondary">{result.exportId}</span>
            <span className="rounded bg-accent/15 px-1.5 py-0.5 text-2xs font-medium text-accent">Ready</span>
          </div>
          {result.sha256 && (
            <p className="truncate font-mono text-2xs text-text-tertiary" title={result.sha256}>
              sha256:{result.sha256}
            </p>
          )}
          {result.manifest && (
            <div className="flex flex-col gap-1">
              <button
                type="button"
                onClick={() => setShowManifest((v) => !v)}
                className="w-fit text-2xs font-medium text-accent hover:underline"
              >
                {showManifest ? "▲ Hide manifest" : "▼ View manifest JSON"}
              </button>
              {showManifest && (
                <pre className="max-h-52 overflow-auto rounded border border-border-strong/40 bg-surface-raised p-2 font-mono text-2xs text-text-secondary">
                  {JSON.stringify(result.manifest, null, 2)}
                </pre>
              )}
            </div>
          )}
          <button
            type="button"
            onClick={onClose}
            className="mt-2 h-8 self-end rounded-sm border border-border-strong px-3 text-xs text-text-primary hover:bg-surface-hover"
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
  const [confirmingRelease, setConfirmingRelease] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const action = caseItem.pending_action;

  function submitRequest(targetValue: string) {
    setError(null);
    startTransition(async () => {
      try {
        const result = await requestBlockAction(caseItem.id, targetValue);
        if (result.error) setError(result.error);
      } catch {
        setError("Could not request block. Try again.");
      }
    });
  }

  function submitRelease() {
    if (!action || isPending) return;
    setError(null);
    startTransition(async () => {
      try {
        const result = await releaseActionAction(action.id, caseItem.id);
        if (result.error) setError(result.error);
        else setConfirmingRelease(false);
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
        <SectionLabel>Network containment</SectionLabel>
        {/* No confirmation here on purpose: this only opens an approval
            request, it does not touch the firewall. The gate that matters
            lives on the approval itself. Saying so keeps the ceremony
            where the consequence actually is. */}
        <p className="pb-2 text-2xs text-text-tertiary">
          Opens an approval request — nothing reaches the firewall until it is approved.
        </p>
        <div className="flex flex-col gap-1.5">
          {targets.map((o) => (
            <button
              key={o.value}
              type="button"
              onClick={() => submitRequest(o.value)}
              disabled={isPending}
              className="inline-flex h-8 items-center justify-between rounded-sm border border-border-strong bg-surface-raised px-2.5 text-xs font-medium text-text-primary transition-colors hover:border-warn hover:text-warn disabled:cursor-not-allowed disabled:opacity-40"
            >
              <code className="font-mono">{o.value}</code>
              <span className="text-2xs font-normal">{isPending ? "Requesting…" : "Request block"}</span>
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
          <span className="rounded-full bg-surface-raised px-2 py-0.5 text-2xs font-medium text-text-secondary">
            {ACTION_STATUS_LABEL[action.status] ?? action.status}
          </span>
        }
      >
        Network block
      </SectionLabel>

      <div className="rounded border border-border-strong/40 bg-surface-raised p-2 text-xs">
        <div className="flex items-center justify-between gap-2">
          <code className="truncate font-mono font-medium text-text-primary">{action.target.value}</code>
          <ModeBadge mode={action.mode} />
        </div>
        {action.reason && <p className="mt-1 text-2xs text-text-tertiary">{action.reason}</p>}
      </div>

      {/* The same control the Approvals queue uses, rather than a second
          approve/reject built to different proportions — one decision,
          one interface, wherever it is reached from. */}
      {action.status === "pending_approval" && action.approval_id && (
        <div className="border-t border-border/60 pt-2.5">
          <ApprovalRowActions
            approvalId={action.approval_id}
            action={action}
            expiresAt={action.expires_at}
            size="md"
          />
        </div>
      )}

      {action.status === "executed" && (
        <button
          type="button"
          onClick={() => setConfirmingRelease(true)}
          disabled={isPending}
          aria-haspopup="dialog"
          className="inline-flex h-8 items-center justify-center gap-1.5 rounded-sm border border-warn text-xs font-medium text-warn transition-all hover:bg-warn/10 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-40"
        >
          {isPending ? <ActionSpinner /> : "Release block now"}
        </button>
      )}

      {confirmingRelease && (
        <Modal title="Release this block?" onClose={() => setConfirmingRelease(false)} tone="danger" dismissible={!isPending}>
          <div className="flex flex-col gap-3">
            <p className="text-xs leading-relaxed text-text-secondary">
              Removes the block on{" "}
              <code className="rounded bg-surface-raised px-1 py-0.5 font-mono text-2xs text-text-primary">
                {action.target.value}
              </code>{" "}
              from the live firewall immediately. Traffic from this address is no longer contained.
            </p>
            <p className="text-2xs text-text-tertiary">
              This block would otherwise release on its own when its TTL elapses.
            </p>
            {error && (
              <p role="alert" className="rounded-md border border-danger/40 bg-danger/[0.08] px-2.5 py-2 text-xs font-medium text-danger">
                {error}
              </p>
            )}
            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setConfirmingRelease(false)}
                disabled={isPending}
                className="inline-flex h-9 items-center rounded-sm border border-border-strong bg-surface-raised px-3.5 text-xs font-medium text-text-secondary hover:text-text-primary disabled:opacity-40"
              >
                Keep blocked
              </button>
              <button
                type="button"
                onClick={submitRelease}
                disabled={isPending}
                className="inline-flex h-9 min-w-[110px] items-center justify-center rounded-sm bg-warn px-3.5 text-xs font-semibold text-bg hover:opacity-90 disabled:opacity-40"
              >
                {isPending ? <ActionSpinner /> : "Release now"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {!confirmingRelease && <InlineError message={error} />}
    </div>
  );
}
