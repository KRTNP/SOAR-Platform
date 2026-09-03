"use server";

import { revalidatePath } from "next/cache";
import { createTenant, addAllowlistEntry, addCriticalTarget } from "@/lib/api-client/client";
import type { Tenant } from "@/lib/api-client/types";

// Returns the created Tenant (including its one-time api_key) directly to
// the client component instead of relying on revalidatePath alone — the
// server-rendered list from GET /v1/tenants never includes api_key, so the
// only place it's ever visible is this action's own return value.
export async function createTenantAction(name: string): Promise<Tenant> {
  const tenant = await createTenant(name);
  revalidatePath("/settings");
  return tenant;
}

export async function addAllowlistEntryAction(value: string, reason: string, expiresAt: string) {
  await addAllowlistEntry(value, reason, expiresAt || undefined);
  revalidatePath("/settings");
}

export async function addCriticalTargetAction(value: string, reason: string) {
  await addCriticalTarget(value, reason);
  revalidatePath("/settings");
}
