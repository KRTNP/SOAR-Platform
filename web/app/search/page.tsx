import Link from "next/link";
import { search, ApiProblemError } from "@/lib/api-client/client";
import type { SearchResult } from "@/lib/api-client/types";
import { ProblemBanner } from "@/components/ProblemBanner";
import { EmptyState } from "@/components/EmptyState";

export const dynamic = "force-dynamic";

// GET /v1/search (adr/0022) is only mounted server-side when
// OPENSEARCH_ENABLE_REAL=true — a 503 here means the search indexer isn't
// running in this environment, not that the query itself was invalid.
export default async function SearchPage({ searchParams }: { searchParams: Promise<{ q?: string }> }) {
  const { q } = await searchParams;
  const query = q?.trim() ?? "";

  let results: SearchResult[] | undefined;
  let loadError: ApiProblemError | undefined;
  if (query) {
    try {
      results = (await search(query)).items;
    } catch (err) {
      if (err instanceof ApiProblemError) {
        loadError = err;
      } else {
        throw err;
      }
    }
  }

  return (
    <div className="flex flex-col gap-[22px]">
      <header>
        <h1 className="text-xl font-semibold tracking-tight">Search</h1>
        <p className="text-xs text-text-tertiary">
          Cross-entity search over the OpenSearch derived projection (adr/0022) &mdash; cases and events, never the sole
          source of truth for anything shown here
        </p>
      </header>

      <form method="get" className="flex gap-2">
        <input
          type="text"
          name="q"
          defaultValue={query}
          placeholder="Search cases and events…"
          aria-label="Search query"
          className="flex-1 rounded-[4px] border border-border-strong bg-surface-raised px-2.5 py-1.5 text-[13px] text-text-primary placeholder:text-text-tertiary"
        />
        <button
          type="submit"
          className="rounded-[4px] bg-accent px-3 py-1.5 text-[13px] font-medium text-bg"
        >
          Search
        </button>
      </form>

      {loadError && <ProblemBanner problem={loadError.problem} fallbackTitle="Search is unavailable" />}

      {!loadError && query && results && results.length === 0 && (
        <EmptyState title="No matches" body={`Nothing in the search index matched "${query}".`} />
      )}

      {!loadError && !query && (
        <EmptyState title="Search cases and events" body="Type a term above and press Search." />
      )}

      {!loadError && results && results.length > 0 && (
        <ul className="flex flex-col gap-2">
          {results.map((r) => (
            <SearchResultRow key={`${r.index}:${r.id}`} result={r} />
          ))}
        </ul>
      )}
    </div>
  );
}

function SearchResultRow({ result }: { result: SearchResult }) {
  const isCase = result.index === "soar-cases-v1";
  const title = typeof result.source.title === "string" ? result.source.title : result.id;
  const severity = typeof result.source.severity === "string" ? result.source.severity : undefined;
  const status = typeof result.source.status === "string" ? result.source.status : undefined;
  const caseId = isCase ? result.id : typeof result.source.case_id === "string" ? result.source.case_id : undefined;

  return (
    <li className="rounded-md border border-border bg-surface p-3.5">
      <div className="flex items-center gap-2">
        <span className="rounded-[4px] border border-border-strong px-1.5 py-0.5 font-mono text-[10px] tracking-wider text-text-tertiary uppercase">
          {isCase ? "case" : "event"}
        </span>
        {caseId ? (
          <Link href={`/cases/${encodeURIComponent(caseId)}`} className="text-[13px] font-medium text-accent hover:underline">
            {title}
          </Link>
        ) : (
          <span className="text-[13px] font-medium text-text-primary">{title}</span>
        )}
      </div>
      <p className="mt-1 font-mono text-[11px] text-text-tertiary">
        {result.id}
        {severity && <> &middot; {severity}</>}
        {status && <> &middot; {status}</>}
      </p>
    </li>
  );
}
