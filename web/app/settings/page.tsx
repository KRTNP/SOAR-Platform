import { listTenants, listAllowlist, listCriticalTargets, ApiProblemError } from "@/lib/api-client/client";
import type { Tenant, TrustedSource, CriticalTarget, Problem } from "@/lib/api-client/types";
import { ProblemBanner } from "@/components/ProblemBanner";
import { EmptyState } from "@/components/EmptyState";
import { TenantForm, AllowlistForm, CriticalTargetForm } from "./forms";
import { DeleteButton } from "./delete-button";
import { deleteTenantAction, deleteAllowlistEntryAction, deleteCriticalTargetAction } from "./actions";

export const dynamic = "force-dynamic";

interface SectionResult<T> {
  items?: T[];
  problem?: Problem;
}

async function loadSection<T>(fn: () => Promise<{ items: T[] }>): Promise<SectionResult<T>> {
  try {
    const page = await fn();
    return { items: page.items };
  } catch (err) {
    if (err instanceof ApiProblemError) return { problem: err.problem };
    throw err;
  }
}

function fmtDate(iso: string) {
  return new Date(iso).toLocaleDateString([], { year: "numeric", month: "short", day: "numeric" });
}

export default async function SettingsPage() {
  const [tenants, allowlist, criticalTargets] = await Promise.all([
    loadSection<Tenant>(listTenants),
    loadSection<TrustedSource>(listAllowlist),
    loadSection<CriticalTarget>(listCriticalTargets),
  ]);

  return (
    <div className="flex flex-col gap-[28px]">
      <header>
        <h1 className="text-xl font-semibold tracking-tight">Settings</h1>
        <p className="text-xs text-text-tertiary">
          Tenants (adr/0022), the SSH brute-force allowlist (adr/0018), and critical targets (adr/0019) &mdash; no
          auth/RBAC gate yet
        </p>
      </header>

      <section className="flex flex-col gap-3">
        <h2 className="text-[13px] font-semibold tracking-wide text-text-secondary uppercase">Tenants</h2>
        <TenantForm />
        {tenants.problem && <ProblemBanner problem={tenants.problem} fallbackTitle="Could not load tenants" />}
        {tenants.items && tenants.items.length === 0 && (
          <EmptyState title="No tenants yet" body="Every request without an X-Tenant-Key header belongs to the default lab tenant." />
        )}
        {tenants.items && tenants.items.length > 0 && (
          <div className="overflow-x-auto rounded-md border border-border">
            <table className="w-full min-w-[480px] border-collapse text-[12.5px]">
              <thead>
                <tr className="border-b border-border bg-surface text-left text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">
                  <th className="px-3 py-2.5 font-semibold">ID</th>
                  <th className="px-3 py-2.5 font-semibold">Name</th>
                  <th className="px-3 py-2.5 font-semibold">Created</th>
                  <th className="px-3 py-2.5 font-semibold" />
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {tenants.items.map((t) => (
                  <tr key={t.id} className="bg-surface hover:bg-surface-hover">
                    <td className="px-3 py-2.5 font-mono text-text-tertiary" title={t.id}>
                      {t.id.slice(0, 18)}…
                    </td>
                    <td className="px-3 py-2.5 text-text-primary">{t.name}</td>
                    <td className="px-3 py-2.5 font-mono text-text-tertiary whitespace-nowrap">{fmtDate(t.created_at)}</td>
                    <td className="px-3 py-2.5 text-right">
                      <DeleteButton label={t.name} onDelete={deleteTenantAction.bind(null, t.id)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-[13px] font-semibold tracking-wide text-text-secondary uppercase">
          Allowlist &middot; trusted sources
        </h2>
        <AllowlistForm />
        {allowlist.problem && <ProblemBanner problem={allowlist.problem} fallbackTitle="Could not load the allowlist" />}
        {allowlist.items && allowlist.items.length === 0 && (
          <EmptyState title="No trusted sources yet" body="The SSH brute-force auto-trigger will never request a block against a value listed here." />
        )}
        {allowlist.items && allowlist.items.length > 0 && (
          <div className="overflow-x-auto rounded-md border border-border">
            <table className="w-full min-w-[600px] border-collapse text-[12.5px]">
              <thead>
                <tr className="border-b border-border bg-surface text-left text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">
                  <th className="px-3 py-2.5 font-semibold">Value</th>
                  <th className="px-3 py-2.5 font-semibold">Reason</th>
                  <th className="px-3 py-2.5 font-semibold">Added by</th>
                  <th className="px-3 py-2.5 font-semibold">Created</th>
                  <th className="px-3 py-2.5 font-semibold">Expires</th>
                  <th className="px-3 py-2.5 font-semibold" />
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {allowlist.items.map((a) => (
                  <tr key={a.id} className="bg-surface hover:bg-surface-hover">
                    <td className="px-3 py-2.5 font-mono text-text-primary">{a.value}</td>
                    <td className="px-3 py-2.5 text-text-secondary">{a.reason}</td>
                    <td className="px-3 py-2.5 text-text-tertiary">{a.created_by}</td>
                    <td className="px-3 py-2.5 font-mono text-text-tertiary whitespace-nowrap">{fmtDate(a.created_at)}</td>
                    <td className="px-3 py-2.5 font-mono text-text-tertiary whitespace-nowrap">
                      {a.expires_at ? fmtDate(a.expires_at) : "never"}
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      <DeleteButton label={a.value} onDelete={deleteAllowlistEntryAction.bind(null, a.id)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-[13px] font-semibold tracking-wide text-text-secondary uppercase">Critical targets</h2>
        <CriticalTargetForm />
        {criticalTargets.problem && (
          <ProblemBanner problem={criticalTargets.problem} fallbackTitle="Could not load critical targets" />
        )}
        {criticalTargets.items && criticalTargets.items.length === 0 && (
          <EmptyState title="No critical targets yet" body="A correlated case targeting a value listed here is escalated above Medium severity." />
        )}
        {criticalTargets.items && criticalTargets.items.length > 0 && (
          <div className="overflow-x-auto rounded-md border border-border">
            <table className="w-full min-w-[520px] border-collapse text-[12.5px]">
              <thead>
                <tr className="border-b border-border bg-surface text-left text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">
                  <th className="px-3 py-2.5 font-semibold">Value</th>
                  <th className="px-3 py-2.5 font-semibold">Reason</th>
                  <th className="px-3 py-2.5 font-semibold">Added by</th>
                  <th className="px-3 py-2.5 font-semibold">Created</th>
                  <th className="px-3 py-2.5 font-semibold" />
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {criticalTargets.items.map((ct) => (
                  <tr key={ct.id} className="bg-surface hover:bg-surface-hover">
                    <td className="px-3 py-2.5 font-mono text-text-primary">{ct.value}</td>
                    <td className="px-3 py-2.5 text-text-secondary">{ct.reason}</td>
                    <td className="px-3 py-2.5 text-text-tertiary">{ct.created_by}</td>
                    <td className="px-3 py-2.5 font-mono text-text-tertiary whitespace-nowrap">{fmtDate(ct.created_at)}</td>
                    <td className="px-3 py-2.5 text-right">
                      <DeleteButton label={ct.value} onDelete={deleteCriticalTargetAction.bind(null, ct.id)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
