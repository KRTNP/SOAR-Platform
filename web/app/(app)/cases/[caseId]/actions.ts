"use server";

import { revalidatePath } from "next/cache";
import {
  transitionCase,
  closeCase,
  openTicket,
  requestBlock,
  approveAction,
  executeAction,
  previewAction,
  getAction,
  getMetrics,
  rejectApproval,
  releaseAction,
  exportCase,
  ApiProblemError,
} from "@/lib/api-client/client";
import type { CaseStatus, CaseExportAccepted, PendingAction, RedactionProfile } from "@/lib/api-client/types";

export type ActionResult = { error?: string };

// Every mutating action here returns { error } instead of throwing on a
// Problem: Next.js strips the fields off any custom Error subclass thrown
// out of a Server Action before it reaches the client (the same stripping
// app/error.tsx's own comment describes for Server Components), so
// ApiProblemError's actual problem.title — e.g. adr/0024's "caller's role
// cannot approve this action", or a policy gate's reason for refusing a
// transition — would otherwise be replaced by a generic failure string the
// operator cannot act on. In a console whose purpose is an auditable record
// of why something was or wasn't allowed, that reason is the payload.
//
// A genuinely unexpected (non-Problem) error still throws, so it still hits
// the nearest error.tsx as before.
async function guard(fn: () => Promise<unknown>): Promise<ActionResult> {
  try {
    await fn();
  } catch (err) {
    if (err instanceof ApiProblemError) return { error: err.problem.title };
    throw err;
  }
  return {};
}

export async function transitionCaseAction(
  caseId: string,
  status: CaseStatus,
  reason: string,
): Promise<ActionResult> {
  const result = await guard(() => transitionCase(caseId, status, reason || undefined));
  if (result.error) return result;
  revalidatePath(`/cases/${caseId}`);
  revalidatePath("/");
  return {};
}

export async function closeCaseAction(caseId: string, closureReason: string): Promise<ActionResult> {
  const result = await guard(() => closeCase(caseId, closureReason));
  if (result.error) return result;
  revalidatePath(`/cases/${caseId}`);
  revalidatePath("/");
  return {};
}

export async function openTicketAction(caseId: string): Promise<ActionResult> {
  const result = await guard(() => openTicket(caseId));
  if (result.error) return result;
  revalidatePath(`/cases/${caseId}`);
  return {};
}

export async function requestBlockAction(caseId: string, targetValue: string): Promise<ActionResult> {
  const result = await guard(() => requestBlock(caseId, targetValue));
  if (result.error) return result;
  revalidatePath(`/cases/${caseId}`);
  return {};
}

// Which of the two calls failed, so the UI can say something true about the
// state the system is actually in.
export type ApproveOutcome = {
  error?: string;
  /** Present on failure. "execute" means the approval was already recorded. */
  failedStage?: "approve" | "execute";
  /** Present on success: the action re-read from the API after executing. */
  executed?: PendingAction;
};

// The UI's single approve performs both M3 steps (approve, then execute) —
// the server still enforces the gate on each call; this action just spares
// the analyst a second click for an outcome that is a single decision from
// their point of view.
//
// The two calls are not atomic, and pretending otherwise is the dangerous
// version: if approve succeeds and execute fails, the approval is recorded
// permanently while the firewall is untouched. A generic "could not approve"
// there would describe a state the system is not in, in a console whose
// entire value is that its record can be trusted — so the failing stage is
// reported, and the caller can retry execution alone.
//
// On success the action is re-read rather than assumed: what the UI then
// shows is the status the API reports, not the status we asked for.
export async function approveActionAction(
  approvalId: string,
  actionId: string,
  caseId: string,
): Promise<ApproveOutcome> {
  try {
    await approveAction(approvalId);
  } catch (err) {
    if (err instanceof ApiProblemError) return { error: err.problem.title, failedStage: "approve" };
    throw err;
  }

  try {
    await executeAction(actionId);
  } catch (err) {
    revalidatePath(`/cases/${caseId}`);
    revalidatePath("/approvals");
    if (err instanceof ApiProblemError) return { error: err.problem.title, failedStage: "execute" };
    throw err;
  }

  revalidatePath(`/cases/${caseId}`);
  revalidatePath("/approvals");
  revalidatePath("/");

  try {
    return { executed: await getAction(actionId) };
  } catch {
    // The write landed; only the confirmation read failed. Say nothing about
    // the resulting state rather than inventing one.
    return {};
  }
}

// Retry path for the case above: the approval already exists, only the
// execution needs to be attempted again.
export async function executeActionAction(actionId: string, caseId: string): Promise<ApproveOutcome> {
  try {
    await executeAction(actionId);
  } catch (err) {
    if (err instanceof ApiProblemError) return { error: err.problem.title, failedStage: "execute" };
    throw err;
  }
  revalidatePath(`/cases/${caseId}`);
  revalidatePath("/approvals");
  revalidatePath("/");
  try {
    return { executed: await getAction(actionId) };
  } catch {
    return {};
  }
}

// Consequence-before-commitment evidence for the approve dialog. The API
// answers in prose (StatusResponse.message), and with the real OPNsense
// connector wired it reports whether the target is already blocked — so the
// approver can see a no-op before authorizing it.
//
// `liveConnector` matters more than it looks. services/api/actions.Preview
// only consults the target system when a connector exposes a live state check
// — currently the real OPNsense one (adr/0013). With the mock wired,
// `h.net.Preview` is nil and the endpoint returns a fixed *planning* sentence
// about what would happen, with no observation behind it. Rendering that under
// "here is what the connector currently sees, as of 14:32" would be this
// console claiming a verification it never performed, which is the precise
// failure principle 3 exists to prevent. So the caller is told which kind of
// answer it got, and says so.
export async function previewActionAction(
  intent: string,
  target: string,
): Promise<{ message?: string; error?: string; liveConnector?: boolean }> {
  const liveConnector = await getMetrics()
    .then((snap) => !snap.connector_version.includes("mock"))
    .catch(() => undefined);

  try {
    const result = await previewAction(intent, target);
    return { message: result.message, liveConnector };
  } catch (err) {
    if (err instanceof ApiProblemError) return { error: err.problem.title, liveConnector };
    return { error: "Preview unavailable", liveConnector };
  }
}

export async function rejectActionAction(
  approvalId: string,
  reason: string,
  caseId: string,
): Promise<ActionResult> {
  const result = await guard(() => rejectApproval(approvalId, reason));
  if (result.error) return result;
  revalidatePath(`/cases/${caseId}`);
  revalidatePath("/approvals");
  revalidatePath("/");
  return {};
}

// Releasing is reachable from the approvals receipt and the landing panel,
// not just the case page, so those have to be revalidated too — otherwise a
// released block keeps rendering as executed on the surface it was released
// from.
export async function releaseActionAction(actionId: string, caseId: string): Promise<ActionResult> {
  const result = await guard(() => releaseAction(actionId));
  if (result.error) return result;
  revalidatePath(`/cases/${caseId}`);
  revalidatePath("/approvals");
  revalidatePath("/metrics");
  revalidatePath("/");
  return {};
}

// Returns { error } like every other mutating action here rather than
// throwing: an export refused by a redaction or RBAC policy has a reason, and
// throwing hands the caller a generic string instead of it.
export async function exportCaseAction(
  caseId: string,
  redactionProfile: RedactionProfile,
  purpose: string,
): Promise<{ result?: CaseExportAccepted; error?: string }> {
  try {
    const result = await exportCase(caseId, {
      format: "json",
      redaction_profile: redactionProfile,
      purpose: purpose || undefined,
    });
    revalidatePath(`/cases/${caseId}`);
    return { result };
  } catch (err) {
    if (err instanceof ApiProblemError) return { error: err.problem.title };
    throw err;
  }
}
