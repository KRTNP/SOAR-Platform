"use client";

import Link from "next/link";
import type { Approval, PendingAction } from "@/lib/api-client/types";
import { ApprovalStatusBadge, SeverityBadge } from "@/components/badges";
import { ApprovalRowActions } from "./approval-actions";
import { formatExpiry } from "./format";

interface ApprovalCardProps {
  approval: Approval;
  action?: PendingAction;
  showActions: boolean;
}

export function ApprovalCard({ approval, action, showActions }: ApprovalCardProps) {
  const isExpiringSoon = Boolean(approval.expires_at);

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
        </div>

        {action?.case_id && (
          <Link
            href={`/cases/${action.case_id}`}
            aria-label={`View case ${action.case_id}`}
            className="inline-flex items-center gap-1 rounded-[4px] border border-border-strong/60 bg-surface-raised px-2 py-1 font-mono text-[11px] font-medium text-accent transition-all duration-150 hover:border-accent/60 hover:bg-surface-raised/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
          >
            <span>{action.case_id}</span>
            <span aria-hidden="true" className="text-[10px] opacity-70">↗</span>
          </Link>
        )}
      </div>

      {/* Main Content: Intent & Target */}
      <div className="flex flex-col gap-1">
        <h3
          id={`approval-intent-${approval.id}`}
          className="text-[13.5px] font-semibold tracking-tight text-text-primary"
        >
          {action?.intent ?? "No action intent specified"}
        </h3>

        {action?.target?.value && (
          <div className="flex items-center">
            <code className="max-w-full truncate rounded bg-surface-raised px-1.5 py-0.5 font-mono text-[11px] text-text-secondary border border-border-strong/40">
              {action.target.value}
            </code>
          </div>
        )}
      </div>

      {/* Metadata: Requester & Expiry Info */}
      <div className="flex flex-wrap items-center gap-x-2 text-[11.5px] text-text-tertiary">
        <span className="inline-flex items-center gap-1">
          <span className="text-text-secondary">by</span>
          <span className="font-medium text-text-primary">{action?.requested_by ?? "—"}</span>
        </span>

        <span aria-hidden="true" className="select-none text-text-tertiary/60">•</span>

        <span className="inline-flex items-center gap-1">
          <span className="text-text-secondary">expires</span>
          <time
            dateTime={approval.expires_at ?? undefined}
            className={`font-mono text-[11px] ${isExpiringSoon ? "text-text-secondary" : "text-text-tertiary"
              }`}
          >
            {approval.expires_at ? formatExpiry(approval.expires_at) : "—"}
          </time>
        </span>
      </div>

      {/* Action Footer: Law of Common Region */}
      {showActions && action && (
        <footer className="mt-0.5 border-t border-border/60 pt-2.5">
          <ApprovalRowActions
            approvalId={approval.id}
            actionId={action.id}
            caseId={action.case_id}
            size="md"
          />
        </footer>
      )}
    </article>
  );
}