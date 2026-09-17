"use client";

import Link from "next/link";
import type { Approval, PendingAction } from "@/lib/api-client/types";
import { ApprovalStatusBadge, ModeBadge, SeverityBadge } from "@/components/badges";
import { ApprovalRowActions } from "./approval-actions";
import { formatExpiry, truncateCaseId } from "./format";

interface ApprovalCardProps {
  approval: Approval;
  action?: PendingAction;
  showActions: boolean;
}

export function ApprovalCard({ approval, action, showActions }: ApprovalCardProps) {
  return (
    <article
      aria-labelledby={`approval-intent-${approval.id}`}
      className="group relative flex flex-col gap-3 border-b border-border bg-surface px-4 py-3.5 transition-colors duration-150 last:border-b-0 hover:bg-surface-raised/40"
    >
      {/* Header: Status Badges & Case Link */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-1.5">
          <ApprovalStatusBadge status={approval.status} />
          {action?.risk && <SeverityBadge severity={action.risk} />}
          {action && <ModeBadge mode={action.mode} />}
        </div>

        {action?.case_id && (
          <Link
            href={`/cases/${action.case_id}`}
            aria-label={`View case ${action.case_id}`}
            className="inline-flex items-center gap-1 rounded-sm border border-border-strong/60 bg-surface-raised px-2 py-1 font-mono text-2xs font-medium text-accent transition-all duration-150 hover:border-accent/60 hover:bg-surface-raised/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
          >
            <span>{truncateCaseId(action.case_id)}</span>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" className="h-2.5 w-2.5 opacity-70" aria-hidden="true">
              <path d="M8 16L16 8" />
              <path d="M9 8h7v7" />
            </svg>
          </Link>
        )}
      </div>

      {/* Main Content: Intent & Target */}
      <div className="flex flex-col gap-1">
        <h3
          id={`approval-intent-${approval.id}`}
          className="text-sm font-semibold tracking-tight text-text-primary"
        >
          {action ? action.intent.replaceAll("_", " ") : "—"}
        </h3>

        {action?.target?.value && (
          <div className="flex items-center">
            <code className="max-w-full truncate rounded bg-surface-raised px-1.5 py-0.5 font-mono text-2xs text-text-secondary border border-border-strong/40">
              {action.target.value}
            </code>
          </div>
        )}
      </div>

      {/* Metadata: Requester & Expiry Info */}
      <div className="flex flex-wrap items-center gap-x-2 text-2xs text-text-tertiary">
        <span className="inline-flex items-center gap-1">
          <span className="text-text-secondary">by</span>
          <span className="font-medium text-text-primary">{action?.requested_by ?? "—"}</span>
        </span>

        <span aria-hidden="true" className="select-none text-text-tertiary/60">•</span>

        <span className="inline-flex items-center gap-1">
          <span className="text-text-secondary">{approval.decided_at ? "decided" : "expires"}</span>
          <time
            dateTime={approval.decided_at ?? approval.expires_at ?? undefined}
            className="font-mono text-2xs text-text-secondary"
          >
            {approval.decided_at
              ? formatExpiry(approval.decided_at)
              : approval.expires_at
                ? formatExpiry(approval.expires_at)
                : "—"}
          </time>
        </span>
      </div>

      {/* The decision record, on the surface where the decision is reviewed.
          The reason is the one field the operator was required to supply and
          it had nowhere to appear. */}
      {approval.reason && (
        <p className="rounded-sm border border-border bg-surface-raised/40 px-2 py-1.5 text-2xs leading-relaxed text-text-secondary">
          <span className="text-text-tertiary">{approval.status === "rejected" ? "Rejected" : "Noted"}:</span>{" "}
          {approval.reason}
        </p>
      )}

      {/* Action Footer: Law of Common Region */}
      {showActions && action && (
        <footer className="mt-0.5 border-t border-border/60 pt-2.5">
          <ApprovalRowActions
            approvalId={approval.id}
            action={action}
            expiresAt={approval.expires_at}
            size="md"
          />
        </footer>
      )}
    </article>
  );
}