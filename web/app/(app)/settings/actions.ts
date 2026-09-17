"use server";

import { revalidatePath } from "next/cache";
import {
  createTenant,
  addAllowlistEntry,
  addCriticalTarget,
  deleteTenant,
  deleteAllowlistEntry,
  deleteCriticalTarget,
  ApiProblemError,
} from "@/lib/api-client/client";
import type { Tenant } from "@/lib/api-client/types";

export type SettingsResult = { error?: string };

// Same contract as the case actions: return the Problem's title rather than
// throwing. Next.js strips the fields off a custom Error thrown out of a
// Server Action, so every failure here used to reach the operator as "Could
// not delete. Try again." — which cannot distinguish an adr/0024 RBAC denial
// from a foreign-key constraint, in a console whose stated differentiator is
// showing *why* something was or wasn't permitted.
async function guard(fn: () => Promise<unknown>): Promise<SettingsResult> {
  try {
    await fn();
  } catch (err) {
    if (err instanceof ApiProblemError) return { error: err.problem.title };
    throw err;
  }
  return {};
}

// Returns the created Tenant (including its one-time api_key) directly to
// the client component instead of relying on revalidatePath alone — the
// server-rendered list from GET /v1/tenants never includes api_key, so the
// only place it's ever visible is this action's own return value.
export async function createTenantAction(name: string): Promise<{ tenant?: Tenant; error?: string }> {
  try {
    const tenant = await createTenant(name);
    revalidatePath("/settings");
    return { tenant };
  } catch (err) {
    if (err instanceof ApiProblemError) return { error: err.problem.title };
    throw err;
  }
}

export async function addAllowlistEntryAction(
  value: string,
  reason: string,
  expiresAt: string,
): Promise<SettingsResult> {
  const result = await guard(() => addAllowlistEntry(value, reason, expiresAt || undefined));
  if (!result.error) revalidatePath("/settings");
  return result;
}

export async function addCriticalTargetAction(value: string, reason: string): Promise<SettingsResult> {
  const result = await guard(() => addCriticalTarget(value, reason));
  if (!result.error) revalidatePath("/settings");
  return result;
}

export async function deleteTenantAction(id: string): Promise<SettingsResult> {
  const result = await guard(() => deleteTenant(id));
  if (!result.error) revalidatePath("/settings");
  return result;
}

export async function deleteAllowlistEntryAction(id: string): Promise<SettingsResult> {
  const result = await guard(() => deleteAllowlistEntry(id));
  if (!result.error) revalidatePath("/settings");
  return result;
}

export async function deleteCriticalTargetAction(id: string): Promise<SettingsResult> {
  const result = await guard(() => deleteCriticalTarget(id));
  if (!result.error) revalidatePath("/settings");
  return result;
}
