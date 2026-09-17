import { getAudit, ApiProblemError } from "@/lib/api-client/client";
import { ProblemBanner } from "@/components/ProblemBanner";
import { EmptyState } from "@/components/EmptyState";
import { AuditTable } from "./audit-table";

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
    <div className="flex flex-col gap-5">
      <header className="flex flex-col gap-1 border-b border-border pb-4">
        <h1 className="text-2xl font-semibold tracking-tight">Audit log</h1>
        <p className="text-xs text-text-tertiary">
          Newest first &middot; each row&apos;s hash is recomputed on read, not just trusted from storage
        </p>
      </header>

      {page.items.length === 0 ? (
        <EmptyState
          title="No audit records yet"
          body="Every approval, execution, release, and export writes an entry here."
          action={{ label: "Back to case queue", href: "/" }}
        />
      ) : (
        <AuditTable records={page.items} />
      )}
    </div>
  );
}
