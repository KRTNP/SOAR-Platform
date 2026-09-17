"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Modal } from "@/components/Modal";
import { ModeBadge, SeverityBadge } from "@/components/badges";
import { previewActionAction } from "@/app/(app)/cases/[caseId]/actions";
import type { PendingAction } from "@/lib/api-client/types";
import { Check } from "lucide-react";

// Rendered on the client only, after mount: computing "in 12m" during a
// server render is what makes a timestamp wrong the moment it arrives and
// mismatched between server and client markup.
function useCountdown(iso?: string): string | null {
  const [label, setLabel] = useState<string | null>(null);

  useEffect(() => {
    if (!iso) return;
    const target = new Date(iso).getTime();

    function tick() {
      const ms = target - Date.now();
      if (ms <= 0) return setLabel("expired");
      const mins = Math.floor(ms / 60000);
      if (mins < 60) return setLabel(`${Math.max(1, mins)}m left`);
      const hours = Math.floor(mins / 60);
      setLabel(hours < 24 ? `${hours}h ${mins % 60}m left` : `${Math.floor(hours / 24)}d left`);
    }

    tick();
    const id = setInterval(tick, 30_000);
    return () => clearInterval(id);
  }, [iso]);

  return label;
}

// The API answers "would this change anything?" in prose
// (StatusResponse.message), and only for block_network_indicator — anything
// else returns unsupported_intent, so this does not ask.
function usePreview(intent: string, target: string) {
  const supported = intent === "block_network_indicator";
  const [state, setState] = useState<{ message?: string; error?: string; loading: boolean }>({
    loading: supported,
  });

  useEffect(() => {
    if (!supported) return;
    let cancelled = false;
    previewActionAction(intent, target).then((result) => {
      if (!cancelled) setState({ ...result, loading: false });
    });
    return () => {
      cancelled = true;
    };
  }, [supported, intent, target]);

  return { ...state, supported };
}

// An on-demand observed-state read, not a background poller: this is an
// episodic tool, so it checks when the operator asks rather than hammering a
// firewall nobody is watching.
function TargetCheck({ intent, target }: { intent: string; target: string }) {
  const [state, setState] = useState<{
    message?: string;
    error?: string;
    liveConnector?: boolean;
    at?: Date;
  } | null>(null);
  const [checking, setChecking] = useState(false);

  function check() {
    setChecking(true);
    previewActionAction(intent, target).then((result) => {
      setState({ ...result, at: new Date() });
      setChecking(false);
    });
  }

  // The endpoint only consults the target system when a connector exposes a
  // live state check. With the mock wired it returns a fixed sentence about
  // what *would* happen — true, but a plan, not an observation. Presenting
  // that as "what the connector currently sees, as of 14:32" would be a
  // verification this console never performed.
  const observed = state?.liveConnector === true;

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border bg-surface-raised/40 p-2.5">
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs text-text-tertiary">Target state</span>
        <button
          type="button"
          onClick={check}
          disabled={checking}
          className="inline-flex h-7 items-center rounded-sm border border-border-strong px-2.5 text-2xs font-medium text-text-secondary hover:text-text-primary disabled:opacity-40"
        >
          {checking ? "Checking…" : state ? "Re-check" : "Check target"}
        </button>
      </div>

      {state === null ? (
        <p className="text-2xs leading-relaxed text-text-tertiary">
          Not checked. This asks the wired connector what it reports, which is a separate question from what the API
          recorded above.
        </p>
      ) : (
        <>
          <p className="text-xs leading-relaxed text-text-secondary">
            {state.error ? `Could not check — ${state.error}` : (state.message ?? "No answer returned.")}
          </p>
          {!state.error && (
            <p className={`text-2xs leading-relaxed ${observed ? "text-text-tertiary" : "text-warn"}`}>
              {observed ? (
                <>
                  Observed on the live connector as of{" "}
                  {state.at?.toLocaleTimeString([], { hour: "numeric", minute: "2-digit", second: "2-digit" })} &middot;
                  connector&apos;s own words, not interpreted
                </>
              ) : state.liveConnector === false ? (
                <>
                  Not an observation. This lab runs the mock network connector, which has no live state check — the
                  sentence above describes what would happen, not what is. Verify on the firewall itself.
                </>
              ) : (
                <>
                  Could not tell which connector answered (metrics unreachable), so this may be a planning message
                  rather than an observation. Verify on the firewall itself.
                </>
              )}
            </p>
          )}
        </>
      )}
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-4 py-1.5">
      <span className="shrink-0 text-xs text-text-tertiary">{label}</span>
      <span className="min-w-0 text-right text-xs text-text-secondary">{children}</span>
    </div>
  );
}

export function ApproveDialog({
  action,
  expiresAt,
  onConfirm,
  onCancel,
  isPending,
  error,
  failedStage,
  executed,
  onRetryExecute,
  onRelease,
}: {
  action: PendingAction;
  expiresAt?: string;
  onConfirm: () => void;
  onCancel: () => void;
  isPending: boolean;
  error: string | null;
  failedStage?: "approve" | "execute";
  executed?: PendingAction;
  onRetryExecute: () => void;
  onRelease: () => void;
}) {
  const isExecute = action.mode === "execute";
  const countdown = useCountdown(expiresAt);
  const preview = usePreview(action.intent, action.target.value);
  const isExpired = countdown === "expired";

  // Only block-class intents have a documented rollback path (adr/0014's TTL
  // scheduler plus the manual release endpoint). The exact release time is
  // computed from block_ttl_seconds, which api/openapi.yaml does not expose —
  // so this names the mechanism and refuses to invent a timestamp for it.
  const isReversible = action.intent === "block_network_indicator";

  // Receipt — the moment after. The row used to just vanish, leaving "did the
  // firewall take it?" answerable only by leaving for the OPNsense UI. This
  // states what the API reported back, is explicit that it is a recorded
  // state rather than an independent check, and puts the rollback control in
  // the same place as the act that needs rolling back.
  if (executed) {
    return (
      <Modal title="Executed" onClose={onCancel} size="lg" dismissible={!isPending}>
        <div className="flex flex-col gap-3.5">
          <div className="flex flex-col gap-2 rounded-md border border-safe/40 bg-safe/[0.07] p-3">
            <div className="flex items-center gap-2">
              <Check className="h-4 w-4 shrink-0 text-safe" aria-hidden="true" />
              <span className="min-w-0 truncate text-sm font-semibold text-safe">
                {action.intent.replaceAll("_", " ")} &middot; {action.target.value}
              </span>
            </div>
            <p className="text-xs leading-relaxed text-text-secondary">
              The API reports this action as <span className="font-mono text-text-primary">{executed.status}</span>.
              That is the recorded state, not an independent re-check of the target system.
            </p>
          </div>

          <div className="divide-y divide-border border-y border-border">
            <Row label="Mode">
              <ModeBadge mode={executed.mode} />
            </Row>
            <Row label="Idempotency key">
              <code className="font-mono text-2xs break-all text-text-tertiary">{executed.idempotency_key}</code>
            </Row>
          </div>

          {/* The alt-tab closer. POST /v1/actions/preview asks the wired
              connector whether the target is currently in effect, so re-running
              it after an execute is an observed-state read rather than a replay
              of what we asked for.

              The connector answers in prose and this renders that prose
              verbatim with the time it was obtained. It deliberately does not
              compile the sentence down to a "Verified" badge: deriving a
              verdict by string-matching the message would be the UI inventing
              a fact the contract never returned, which is the exact failure
              this console exists to avoid. The operator reads the connector's
              own words and judges. */}
          {isReversible && <TargetCheck intent={action.intent} target={action.target.value} />}

          <div className="flex items-center justify-end gap-2">
            {isReversible && executed.status === "executed" && (
              <button
                type="button"
                onClick={onRelease}
                disabled={isPending}
                className="inline-flex h-9 items-center rounded-sm border border-warn px-3.5 text-xs font-medium text-warn hover:bg-warn/10 disabled:opacity-40"
              >
                Release now
              </button>
            )}
            <button
              type="button"
              onClick={onCancel}
              className="inline-flex h-9 items-center rounded-sm bg-surface-raised px-3.5 text-xs font-medium text-text-primary hover:bg-surface-hover"
            >
              Done
            </button>
          </div>
        </div>
      </Modal>
    );
  }

  // Partial failure: the approval is recorded, the target was not touched.
  if (failedStage === "execute") {
    return (
      <Modal title="Approved, but not executed" onClose={onCancel} tone="danger" size="lg" dismissible={!isPending}>
        <div className="flex flex-col gap-3.5">
          <p className="rounded-md border border-warn/40 bg-warn/[0.07] px-3 py-2.5 text-xs leading-relaxed text-warn">
            The approval was recorded. Execution against{" "}
            <code className="font-mono break-all">{action.target.value}</code> failed, so the target system was not
            changed.
          </p>
          {error && (
            <p
              role="alert"
              className="rounded-md border border-danger/40 bg-danger/[0.08] px-2.5 py-2 text-xs font-medium text-danger"
            >
              {error}
            </p>
          )}
          <div className="flex items-center justify-end gap-2">
            <button
              type="button"
              onClick={onCancel}
              disabled={isPending}
              className="inline-flex h-9 items-center rounded-sm border border-border-strong bg-surface-raised px-3.5 text-xs font-medium text-text-secondary hover:text-text-primary disabled:opacity-40"
            >
              Leave it
            </button>
            <button
              type="button"
              onClick={onRetryExecute}
              disabled={isPending}
              className="inline-flex h-9 min-w-[132px] items-center justify-center rounded-sm bg-danger px-3.5 text-xs font-semibold text-bg hover:opacity-90 disabled:opacity-40"
            >
              {isPending ? (
                <span className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
              ) : (
                "Retry execution"
              )}
            </button>
          </div>
        </div>
      </Modal>
    );
  }

  return (
    <Modal
      title={isExecute ? "Execute on the live system" : "Run simulated action"}
      onClose={onCancel}
      tone={isExecute ? "danger" : "neutral"}
      size="lg"
      dismissible={!isPending}
    >
      <div className="flex flex-col gap-3.5">
        <div
          className={`flex flex-col gap-2 rounded-md border p-3 ${
            isExecute ? "border-danger/40 bg-danger/[0.07]" : "border-border-strong bg-surface-raised/50"
          }`}
        >
          <div className="flex items-center justify-between gap-3">
            <ModeBadge mode={action.mode} size="lg" />
            <SeverityBadge severity={action.risk} />
          </div>
          <p className={`text-xs leading-relaxed ${isExecute ? "text-danger" : "text-text-secondary"}`}>
            {isExecute
              ? "Approving runs this against the real target system. The change takes effect immediately."
              : "Approving runs this in simulation. Nothing changes on the target system."}
          </p>
        </div>

        <div>
          <span className="text-xs text-text-tertiary">Action</span>
          <div className="mt-1.5 flex flex-col gap-1 rounded-md border border-border bg-bg/40 p-2.5 font-mono text-xs">
            <span className="text-text-primary">{action.intent}</span>
            <span className="break-all text-text-secondary">
              <span className="text-text-tertiary">{action.target.type}</span> {action.target.value}
            </span>
          </div>
        </div>

        {/* Predicted effect, straight from POST /v1/actions/preview. With the
            real connector wired this is where "already blocked — no additional
            effect" appears, before the confirm button is reachable. */}
        {/* Always present, even when the API has no preview for this intent.
            Hiding the row for unsupported intents said "there is nothing to
            know here", which is a different claim from "this system cannot
            tell you" — and the Rollback row three lines down already makes the
            second one out loud. One convention, both rows. */}
        <div>
          <span className="text-xs text-text-tertiary">Predicted effect</span>
          <p className="mt-1.5 rounded-md border border-border bg-surface-raised/40 p-2.5 text-xs leading-relaxed text-text-secondary">
            {!preview.supported
              ? `No preview exists for ${action.intent} — the API only previews block_network_indicator.`
              : preview.loading
                ? "Checking connector state…"
                : preview.error
                  ? `Preview unavailable — ${preview.error}`
                  : (preview.message ?? "No preview returned.")}
          </p>
        </div>

        {/* Consequence was on screen; cause was not. The stated reason is
            recorded on the action and shown on the case page, but the one
            moment a human is deciding on it, it was the screen they had just
            left. */}
        {action.reason && (
          <div>
            <span className="text-xs text-text-tertiary">Requested because</span>
            <p className="mt-1.5 rounded-md border border-border bg-surface-raised/40 p-2.5 text-xs leading-relaxed text-text-secondary">
              {action.reason}
            </p>
          </div>
        )}

        <div className="divide-y divide-border border-y border-border">
          <Row label="Case">
            <Link href={`/cases/${action.case_id}`} className="font-mono text-2xs text-accent hover:underline">
              {action.case_id}
            </Link>
          </Row>
          <Row label="Requested by">{action.requested_by}</Row>
          {expiresAt && (
            <Row label="Expires">
              <span className={isExpired ? "text-danger" : undefined}>
                {new Date(expiresAt).toLocaleString([], {
                  month: "numeric",
                  day: "numeric",
                  hour: "numeric",
                  minute: "2-digit",
                })}
                {countdown && <span className="ml-1.5 text-text-tertiary">({countdown})</span>}
              </span>
            </Row>
          )}
          <Row label="Rollback">
            {isReversible ? (
              <span>
                Releases automatically at its TTL, or manually from the case
                <span className="text-text-tertiary"> — exact release time is not returned by the API</span>
              </span>
            ) : (
              <span className="text-text-tertiary">Not documented for this intent</span>
            )}
          </Row>
        </div>

        <p className="text-2xs leading-relaxed text-text-tertiary">
          An expired request can no longer be approved or executed — the server rejects it.
        </p>

        {error && (
          <p
            role="alert"
            className="rounded-md border border-danger/40 bg-danger/[0.08] px-2.5 py-2 text-xs font-medium text-danger"
          >
            {error}
          </p>
        )}

        <div className="flex items-center justify-end gap-2 pt-0.5">
          <button
            type="button"
            onClick={onCancel}
            disabled={isPending}
            className="inline-flex h-9 items-center justify-center rounded-sm border border-border-strong bg-surface-raised px-3.5 text-xs font-medium text-text-secondary hover:text-text-primary disabled:opacity-40"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={isPending || isExpired}
            className={`inline-flex h-9 min-w-[132px] items-center justify-center rounded-sm px-3.5 text-xs font-semibold whitespace-nowrap disabled:cursor-not-allowed disabled:opacity-40 ${
              isExecute ? "bg-danger text-bg hover:opacity-90" : "bg-accent text-bg hover:opacity-90"
            }`}
          >
            {isPending ? (
              <span className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
            ) : isExpired ? (
              "Request expired"
            ) : isExecute ? (
              "Execute now"
            ) : (
              "Run simulation"
            )}
          </button>
        </div>
      </div>
    </Modal>
  );
}
