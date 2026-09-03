# ADR-0022: Lab tenant keys and OpenSearch derived-search projection

## Status

Accepted — implemented and proven end-to-end against the real running lab
Compose stack, including a real cross-tenant isolation check.

## Decision

Two pieces that compose: `tenant_id` was already a column on every table,
but always resolved to the same hardcoded `"lab"` value process-wide
(`services/api/httpx.TenantStubMiddleware`) — there was no way for more
than one tenant to exist. Real auth/RBAC (Keycloak/OIDC, `adr/0005`) stays
a separate, larger, not-yet-started item; this ADR adds only the smallest
thing that makes `tenant_id` vary for real, so `docs/02-architecture/data-architecture.md`'s
already-decided OpenSearch design (versioned indices, `tenant_id` on every
document) has more than one tenant to actually prove isolation against.

### 1. Lab tenant keys (not authentication)

- New `tenants` table (`id, name, api_key_hash, created_at`), seeded with
  the existing `"lab"` tenant (`api_key_hash` null — it was never issued a
  key, it's the implicit default).
- New `services/api/tenants` package: `Create` generates a random key,
  returns it exactly once (only its SHA-256 hash is stored — not a
  slow/salted password hash, since this selects a tenant, it doesn't
  protect a credential worth defending against offline brute force), `List`,
  `Resolve`. `GET`/`POST /v1/tenants`, same no-auth-gate posture
  `/v1/allowlist` already has.
- `httpx.TenantStubMiddleware` is replaced by `httpx.TenantKeyMiddleware`:
  takes a `resolve func(ctx, key) (string, error)` parameter (bound to
  `tenants.Resolve` by `main.go`) so `httpx` stays free of any domain-package
  dependency. No `X-Tenant-Key` header → `defaultTenantID` (every existing
  curl/test/UI call keeps working, unchanged). A header present but
  unrecognized → 401 (new `httpx.ErrUnauthorized`), never a silent fallback
  to the wrong tenant.
- Explicitly **not** authentication: no identity, no session, no RBAC. A
  key only ever selects which `tenant_id` a request's data belongs to.

### 2. OpenSearch derived-search projection

- New `opensearch` service (`docker-compose.core.yml`, `profiles:
  ["search"]`), single-node, `plugins.security.disabled=true` — a lab-only
  posture, published on `127.0.0.1:9201` (not 9200, which a real lab's own
  Wazuh indexer often already occupies).
- New `outbox_events` table + `services/api/outbox.Write`: called inside
  the same transaction as `events.Handler.Ingest`'s case/event writes, so
  the projection can never observe a value that was never actually
  committed. Only `events/handler.go` writes outbox rows in this pass —
  `cases/handler.go`'s `Update`/`Close` operations do not re-project (see
  Consequences).
- New `services/api/searchindex` package: `Run` polls unindexed
  `outbox_events` rows and `PUT`s each to OpenSearch at
  `soar-<entity_type>s-v1/_doc/<entity_id>` (the exact index names
  `data-architecture.md` already specifies), marking `indexed_at`. Plain
  HTTP client, no OpenSearch SDK dependency — the same posture
  `wazuhpoll`/`opnsense` already use.
- New `GET /v1/search?q=...` (`searchindex.Handler.Search`): a
  `multi_match` query across `soar-cases-v1,soar-events-v1`, filtered by
  `term: {"tenant_id.keyword": <caller's tenant>}`. Only mounted when
  `OPENSEARCH_ENABLE_REAL=true` (no fallback search over Postgres).
- Minimal frontend: `web/app/search/page.tsx` — a plain GET `<form>`, no
  client JS, results linking back to `/cases/{id}` when the hit is (or
  names) a case.

## Verification

Full unit + integration suite passed (`go test ./...`,
`go test -tags integration ./integration/...` including three new tests:
`TestTenantIsolation_SecondTenantCannotSeeFirstTenantsCases`,
`TestTenantKey_UnrecognizedKeyIsRejected`,
`TestTenantKey_NoHeaderKeepsDefaultTenantBehavior`, plus
`TestIngestion_WritesOutboxRowsForSearchIndex`). Against the real running
Compose stack: created two real tenants via `POST /v1/tenants`, posted one
event under each tenant's key, confirmed `GET /v1/search` (and a direct
query against `localhost:9201`) returned only each tenant's own document —
caught and fixed a real bug in the process (the `term` filter against bare
`tenant_id` silently matched nothing, since OpenSearch's dynamic mapping
made it an analyzed `text` field tokenized on `-`; fixed to
`tenant_id.keyword`). Confirmed the default `lab` tenant (no header) still
works unchanged. Confirmed the frontend `/search` page end-to-end in a real headless
Chromium (Playwright's Python bindings, found already installed on this
host): typed a query into the input, clicked Search, screenshotted the
results, clicked a result's link, and confirmed it actually navigated to
the correct case detail page — not just that the right `href` was present
in the markup.

## Consequences

- **Not real auth/RBAC.** `httpx.TenantKeyMiddleware` selects a tenant, it
  does not authenticate a person — Keycloak/OIDC (`adr/0005`) remains
  unstarted, a separate and larger item.
- **OpenSearch security is disabled** (`plugins.security.disabled=true`) —
  a lab-only posture; the `GET /v1/search` tenant filter is defense in
  depth, not a real security boundary, matching
  `data-architecture.md`'s own "must not be the sole source of truth"
  caveat for this whole projection.
- **Only `events.Handler.Ingest` writes outbox rows.** A case mutated later
  through `cases/handler.go`'s `Update`/`Close`/`request_block` operations
  does not re-project — its OpenSearch document can go stale relative to
  status/severity changes made through those paths. Disclosed, not fixed
  here — extending outbox writes to those call sites is a small, separate
  follow-up if search accuracy for closed/updated cases ever matters.
- **No backfill.** Only new writes flow through the outbox from this point
  forward; every case/event that existed before this ADR has no OpenSearch
  document. A one-off backfill script (read existing rows, call
  `outbox.Write` for each) is a disclosed follow-up if ever needed.
- `httpx.TenantStubMiddleware` no longer exists — every call site
  (`main.go`, `integration/ingestion_test.go`) was updated to
  `TenantKeyMiddleware`.
