"use server";

import { revalidatePath } from "next/cache";
import { transitionCase, closeCase, openTicket } from "@/lib/api-client/client";
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
