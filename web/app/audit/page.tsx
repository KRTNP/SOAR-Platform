import Link from "next/link";
import { getAudit, ApiProblemError } from "@/lib/api-client/client";
import { DecisionBadge, VerifiedBadge } from "@/components/badges";
import { ProblemBanner } from "@/components/ProblemBanner";
import { EmptyState } from "@/components/EmptyState";

export const dynamic = "force-dynamic";

export default async function AuditPage() {
  let page;
  try {
    page = await getAudit();
  } catch (err) {
    if (err instanceof ApiProblemError) {
      return <ProblemBanner problem={err.problem} fallbackTitle="Could not load the audit log" />;
    }
    throw err;
  }

  return (
    <div className="flex flex-col gap-[22px]">
      <header>
        <h1 className="text-xl font-semibold tracking-tight">Audit log</h1>
        <p className="text-xs text-text-tertiary">
          {page.items.length} record{page.items.length === 1 ? "" : "s"} &middot; newest first &middot; each row&apos;s hash is
          recomputed on read, not just trusted from storage
        </p>
      </header>

      {page.items.length === 0 ? (
        <EmptyState title="No audit records yet" body="Every approval, execution, release, and export writes an entry here." />
      ) : (
        <div>
          <p className="mb-1.5 text-[11px] text-text-tertiary md:hidden">Scroll sideways for decision, actor, case, and connector →</p>
          <div className="overflow-x-auto rounded-md border border-border">
            <table className="w-full min-w-[900px] border-collapse text-[12.5px]">
              <thead>
                <tr className="border-b border-border bg-surface text-left text-[10.5px] font-semibold tracking-wider text-text-tertiary uppercase">
                  <th className="px-3 py-2.5 font-semibold">Timestamp</th>
                  <th className="px-3 py-2.5 font-semibold">Action</th>
                  <th className="px-3 py-2.5 font-semibold">Decision</th>
                  <th className="px-3 py-2.5 font-semibold">Actor</th>
                  <th className="px-3 py-2.5 font-semibold">Case</th>
                  <th className="px-3 py-2.5 font-semibold">Connector</th>
                  <th className="px-3 py-2.5 font-semibold">Integrity</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {page.items.map((r) => (
                  <tr key={r.id} className="bg-surface hover:bg-surface-hover">
                    <td className="px-3 py-2.5 font-mono text-text-tertiary whitespace-nowrap">
                      {new Date(r.timestamp).toLocaleString()}
                    </td>
                    <td className="px-3 py-2.5 font-mono text-text-primary whitespace-nowrap">{r.action}</td>
                    <td className="px-3 py-2.5">
                      <DecisionBadge decision={r.decision} />
                    </td>
                    <td className="px-3 py-2.5 text-text-secondary whitespace-nowrap">
                      {r.actor} <span className="text-text-tertiary">({r.actor_type})</span>
                    </td>
                    <td className="px-3 py-2.5 whitespace-nowrap">
                      {r.case_id ? (
                        <Link href={`/cases/${r.case_id}`} className="font-mono text-accent hover:underline">
                          {r.case_id}
                        </Link>
                      ) : (
                        <span className="text-text-tertiary">—</span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 font-mono text-text-tertiary whitespace-nowrap">{r.connector_version ?? "—"}</td>
                    <td className="px-3 py-2.5">
                      <VerifiedBadge verified={r.verified} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
