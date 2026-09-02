"use client";

import { useState, useTransition } from "react";
import type { Case, CaseStatus, RedactionProfile } from "@/lib/api-client/types";
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

// blockableTargets mirrors the mock enrichment connector's enrichable types
// (services/api/enrichment/enrichment.go) — only indicator-shaped
// observables are something the mock network-block connector understands.
const BLOCKABLE_TYPES = new Set(["ipv4", "ipv6"]);

const ACTION_STATUS_LABEL: Record<string, string> = {
  pending_approval: "Awaiting approval",
  approved: "Approved",
  executed: "Blocked",
  rejected: "Rejected",
  expired: "Expired",
  released: "Released",
};

// ActionControl is the M3 approval/deny/expiry gate surfaced in the UI: an
// analyst requests a block on one of the case's observables, a (possibly
// different, once real auth exists) analyst approves or rejects it, and the
// connector only runs once approved — see adr/0011.
export function ActionControl({ caseItem }: { caseItem: Case }) {
  const [isPending, startTransition] = useTransition();
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);

  const action = caseItem.pending_action;

  function submitRequest(targetValue: string) {
    setError(null);
    startTransition(async () => {
      try {
        await requestBlockAction(caseItem.id, targetValue);
      } catch {
        setError("Could not request a block. Try again.");
      }
    });
  }

  function submitApprove() {
    if (!action?.approval_id) return;
    setError(null);
    startTransition(async () => {
      try {
        await approveActionAction(action.approval_id!, action.id, caseItem.id);
      } catch {
        setError("Could not approve the action. Try again.");
      }
    });
  }

  function submitReject() {
    if (!action?.approval_id || !reason.trim()) return;
    setError(null);
    startTransition(async () => {
      try {
        await rejectActionAction(action.approval_id!, reason, caseItem.id);
        setReason("");
      } catch {
        setError("Could not reject the action. Try again.");
      }
    });
  }

  function submitRelease() {
    if (!action) return;
    setError(null);
    startTransition(async () => {
      try {
        await releaseActionAction(action.id, caseItem.id);
      } catch {
        setError("Could not release the block. Try again.");
      }
    });
  }

  if (!action) {
    const targets = (caseItem.observables ?? []).filter((o) => BLOCKABLE_TYPES.has(o.type));
    if (targets.length === 0) return null;
    return (
      <div className="flex flex-col gap-2 rounded-md border border-border bg-surface p-3.5">
        <label className="text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">Network block</label>
        {targets.map((o) => (
          <button
            key={o.value}
            onClick={() => submitRequest(o.value)}
            disabled={isPending}
            className="rounded-[4px] border border-border-strong px-3 py-1.5 text-[13px] font-medium text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
          >
            {isPending ? "Requesting…" : `Request block: ${o.value}`}
          </button>
        ))}
        {error && <p className="text-[12px] text-danger">{error}</p>}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border bg-surface p-3.5">
      <label className="text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">Network block</label>
      <p className="text-[13px] text-text-primary">
        {action.target.value} <span className="text-text-tertiary">· {ACTION_STATUS_LABEL[action.status] ?? action.status}</span>
      </p>
      {action.reason && <p className="text-[12px] text-text-tertiary">{action.reason}</p>}
      {action.status === "pending_approval" && (
        <>
          <button
            onClick={submitApprove}
            disabled={isPending}
            className="rounded-[4px] bg-accent px-3 py-1.5 text-[13px] font-medium text-bg disabled:cursor-not-allowed disabled:opacity-40"
          >
            {isPending ? "Approving…" : "Approve"}
          </button>
          <input
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Rejection reason (required to reject)"
            className="rounded-[4px] border border-border-strong bg-surface-raised px-2.5 py-1.5 text-[13px] text-text-primary placeholder:text-text-tertiary"
          />
          <button
            onClick={submitReject}
            disabled={!reason.trim() || isPending}
            className="rounded-[4px] border border-danger px-3 py-1.5 text-[13px] font-medium text-danger disabled:cursor-not-allowed disabled:opacity-40"
          >
            Reject
          </button>
        </>
      )}
      {action.status === "executed" && (
        // Manual early rollback (adr/0014) — the in-process scheduler also
        // releases this automatically once block_ttl_seconds elapses; this
        // button just lets an analyst do it sooner without waiting on a tick.
        <button
          onClick={submitRelease}
          disabled={isPending}
          className="rounded-[4px] border border-warn px-3 py-1.5 text-[13px] font-medium text-warn disabled:cursor-not-allowed disabled:opacity-40"
        >
          {isPending ? "Releasing…" : "Release now"}
        </button>
      )}
      {error && <p className="text-[12px] text-danger">{error}</p>}
    </div>
  );
}

const REDACTION_PROFILES: RedactionProfile[] = ["standard", "full"];

// ExportControl surfaces M4's redacted case manifest export (adr/0012).
// Only format=json is offered — csv is spec'd in api/openapi.yaml but the
// server rejects it (services/api/cases/export.go), so there is no working
// csv option to show.
export function ExportControl({ caseItem }: { caseItem: Case }) {
  const [profile, setProfile] = useState<RedactionProfile>("standard");
  const [purpose, setPurpose] = useState("");
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<{ exportId: string; sha256?: string; manifest?: Record<string, unknown> } | null>(null);
  const [showManifest, setShowManifest] = useState(false);

  function submit() {
    setError(null);
    startTransition(async () => {
      try {
        const accepted = await exportCaseAction(caseItem.id, profile, purpose);
        setResult({ exportId: accepted.export_id, sha256: accepted.manifest_sha256, manifest: accepted.manifest });
        setShowManifest(false);
      } catch {
        setError("Could not export this case. Try again.");
      }
    });
  }

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border bg-surface p-3.5">
      <label className="text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">Export case</label>
      <select
        value={profile}
        onChange={(e) => setProfile(e.target.value as RedactionProfile)}
        className="rounded-[4px] border border-border-strong bg-surface-raised px-2.5 py-1.5 text-[13px] text-text-primary capitalize"
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
        placeholder="Purpose (optional)"
        className="rounded-[4px] border border-border-strong bg-surface-raised px-2.5 py-1.5 text-[13px] text-text-primary placeholder:text-text-tertiary"
      />
      <button
        onClick={submit}
        disabled={isPending}
        className="rounded-[4px] border border-border-strong px-3 py-1.5 text-[13px] font-medium text-text-primary disabled:cursor-not-allowed disabled:opacity-40"
      >
        {isPending ? "Exporting…" : "Export manifest"}
      </button>
      {error && <p className="text-[12px] text-danger">{error}</p>}
      {result && (
        <div className="flex flex-col gap-1.5 border-t border-border pt-2">
          <p className="text-[12px] text-text-secondary">
            {result.exportId} <span className="text-text-tertiary">· completed</span>
          </p>
          {result.sha256 && <p className="truncate font-mono text-[10.5px] text-text-tertiary">sha256:{result.sha256}</p>}
          {result.manifest && (
            <>
              <button
                onClick={() => setShowManifest((v) => !v)}
                className="w-fit text-[11.5px] font-medium text-accent"
              >
                {showManifest ? "Hide manifest" : "View manifest"}
              </button>
              {showManifest && (
                <pre className="max-h-64 overflow-auto rounded-[4px] border border-border bg-surface-raised p-2.5 font-mono text-[10.5px] text-text-secondary">
                  {JSON.stringify(result.manifest, null, 2)}
                </pre>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
