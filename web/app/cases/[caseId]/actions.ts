"use server";

import { revalidatePath } from "next/cache";
import { transitionCase, closeCase, openTicket, requestBlock, approveAction, executeAction, rejectApproval } from "@/lib/api-client/client";
import type { CaseStatus } from "@/lib/api-client/types";

export async function transitionCaseAction(caseId: string, status: CaseStatus, reason: string) {
  await transitionCase(caseId, status, reason || undefined);
  revalidatePath(`/cases/${caseId}`);
  revalidatePath("/");
}

export async function closeCaseAction(caseId: string, closureReason: string) {
  await closeCase(caseId, closureReason);
  revalidatePath(`/cases/${caseId}`);
  revalidatePath("/");
}

export async function openTicketAction(caseId: string) {
  await openTicket(caseId);
  revalidatePath(`/cases/${caseId}`);
}

export async function requestBlockAction(caseId: string, targetValue: string) {
  await requestBlock(caseId, targetValue);
  revalidatePath(`/cases/${caseId}`);
}

// The UI's single "Approve" button performs both M3 steps (approve, then
// execute) — the server still enforces the gate on each call; this action
// just spares the analyst a second click for an outcome that is a single
// decision from their point of view.
export async function approveActionAction(approvalId: string, actionId: string, caseId: string) {
  await approveAction(approvalId);
  await executeAction(actionId);
  revalidatePath(`/cases/${caseId}`);
}

export async function rejectActionAction(approvalId: string, reason: string, caseId: string) {
  await rejectApproval(approvalId, reason);
  revalidatePath(`/cases/${caseId}`);
}
