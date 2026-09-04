# ADR-0024: Keycloak auth and RBAC enforcement

## Status

Accepted — implemented and verified end-to-end against a real Keycloak
instance, including a real browser login/logout round trip.

## Decision

`services/api/actions/policy.go` already computed `approvalRoles` per
intent (`analyst`/`incident-commander`/`service-owner`), but nothing ever
checked it against a real caller — every audit `Actor` and every
`Approve(...)` call was a hardcoded Go string literal, and
`httpx.TenantKeyMiddleware` (adr/0022) only ever selected a tenant, never
authenticated a person. This ADR wires a real identity into that existing,
previously-inert authorization logic, rather than designing a new
permission system. It deliberately enforces `policy.go`'s 3-role
vocabulary, not the broader 7-tier model in
`docs/06-security/authentication-and-rbac.md` — reconciling those two is a
separate, later exercise; the goal here is "make the check that already
exists check something real."

### API-side verification

- New `services/api/auth` package: `Verifier.Middleware()` parses
  `Authorization: Bearer <token>`, verifies signature and expiry against a
  Keycloak realm's JWKS (via `github.com/golang-jwt/jwt/v5` +
  `github.com/MicahParks/keyfunc/v3` — the minimal pair for "verify a
  bearer token against a remote JWKS," no OIDC discovery/userinfo client
  needed since no login flow happens server-side), and sets `Claims`
  (subject, email, realm roles) in the Gin context. Gated behind
  `AUTH_ENABLE_REAL` (default off, same convention as
  `opnsense.Enabled()`), mounted globally on `/v1` when on.
- `actions.Handler.Approve`/`Reject`: when a token was verified, the
  caller's roles are checked against `EvaluatePolicy`'s `approvalRoles`
  for the approval's underlying action *before* deciding anything —
  `httpx.ErrForbidden("insufficient_role", ...)` on a miss. This activates
  `approvalRoles`'s previously-computed-but-unused value. Skipped entirely
  when auth is off, preserving the pre-ADR "anyone can approve" behavior.
- Every audit/approval call site that used to hardcode
  `"incident-commander"`/`"analyst"` now calls `auth.ActorFrom(c, fallback,
  fallbackType)`, which returns the caller's real email (or subject) and
  roles when a token was verified, or the fallback literal unchanged when
  it wasn't.
- Test strategy: existing `integration/*_test.go` stay unauthenticated and
  untouched. New `integration/auth_test.go` mints its own RS256 JWTs
  against a locally generated RSA keypair and builds a `keyfunc.Keyfunc`
  directly from a hand-built JWK Set JSON (`keyfunc.NewJWKSetJSON`) rather
  than serving one over HTTP — what's under test is this service's
  verification/RBAC logic, not the JWKS-fetch transport (a keyfunc
  concern). Covers: no/expired/wrong-signature token → 401; valid token
  missing the required role → 403; the right role → 200.

### Compose / Keycloak

- New `keycloak` service (`profiles: ["security"]`, `quay.io/keycloak/
  keycloak:26.0`, `start-dev --import-realm`) filling in the reserved
  placeholder alongside adr/0023's `vault`. `deploy/compose/keycloak/
  realm-soar.json` seeds a single `soar` realm: the three
  `policy.go`-matching realm roles, a public `soar-web` client
  (Authorization Code + PKCE, no client secret), and one test user per
  role. Passwords are lab-only fixed values, same low-stakes posture as
  this stack's other committed lab credentials (e.g. Postgres's
  `soar_lab_only`) — documented in `deploy/compose/keycloak/README.md`.
- Hit the same Fedora-Enforcing SELinux bind-mount permission issue
  documented in adr/0020 (`AccessDeniedException` reading the realm-import
  file) — fixed the same way: `chmod o+rx` the directory, `chmod 644` the
  file, on top of the `:Z` mount flag already in place.
- Single realm, no per-tenant Keycloak realms — explicitly out of scope.

### Web-side login

- Auth.js (`next-auth@5.0.0-beta.32`) with its Keycloak provider —
  owns the PKCE exchange and session cookie rather than hand-rolling OIDC.
  **The spike this ADR planned for worked on the first attempt**: renaming
  the file to `proxy.ts` and exporting `{ auth as proxy }` (per Next 16's
  rename of `middleware`→`proxy`) is a drop-in match for Auth.js's
  documented `{ auth as middleware }` pattern — `next build`'s route table
  confirms `ƒ Proxy (Middleware)` is registered and running. No
  `openid-client` fallback was needed.
- `web/lib/auth.ts`: JWT session strategy (no DB adapter exists),
  `callbacks.jwt` copies `account.access_token` and the profile's
  `realm_access.roles` onto the session token once at sign-in;
  `callbacks.authorized` gates every route `proxy.ts`'s matcher covers.
  `soar-web` is a public client, so the provider is configured with
  `client: { token_endpoint_auth_method: "none" }` and `checks: ["pkce"]`.
- `web/lib/api-client/client.ts`'s `request()` now calls `auth()` and
  forwards the session's access token as `Authorization: Bearer` — stays
  `server-only` as before; omitted entirely when there's no session (auth
  off, or an unauthenticated request during dev).
- `web/app/login/page.tsx` + `web/app/actions.ts`'s `signOutAction` +
  a small sign-out control added to `Sidebar` complete the loop.

## Verification

`go build/vet/test ./...` including the new `auth_test.go`, full existing
suite green and unauthenticated/unchanged
(`go test -tags integration ./integration/...`). `cd web && npx tsc
--noEmit && npx eslint . && npm run build` (the build output's route table
is what first confirmed `proxy.ts` was recognized).

Real stack, `security` profile up, real Keycloak: minted real tokens for
all three test users via the admin API, confirmed `curl` against the live
`api` container with `AUTH_ENABLE_REAL=true` — an `analyst` token gets 403
approving a `block_network_indicator` approval (needs
`incident-commander`), an `incident-commander` token gets 200, and
`GET /v1/audit` afterward shows the real `ic@lab.local`/`analyst@lab.local`
emails as `Actor` instead of the old fixed literals. Real browser
(Playwright): full sign-in round trip (redirect to `/login` when
unauthenticated → Keycloak's real login form → back on the app with a
`authjs.session-token` cookie), subsequent page navigation works without
re-prompting, and sign-out clears the cookie and bounces back to `/login`.

## Consequences

- **3-role scope, not the docs' 7-tier model.** Reconciling
  `docs/06-security/authentication-and-rbac.md`'s broader tiering with
  `policy.go`'s narrower enforced set is explicitly deferred.
- **Single realm.** No per-tenant Keycloak realm separation — every tenant
  (adr/0022) shares one `soar` realm and one set of roles.
- **Dev-mode Keycloak only**, same posture as adr/0023's Vault — no
  production hardening (no real TLS, no HA, no external user federation).
- **`AUTH_ENABLE_REAL` default off.** Nothing changes for anyone not
  opting in; every literal-actor fallback this ADR touches is preserved
  exactly.
- **Minor cosmetic rough edge**: immediately after the OIDC callback
  completes, the browser can briefly re-render `/login` once before a
  subsequent navigation confirms the session and proceeds normally — the
  session itself is valid the entire time (confirmed by re-navigating
  immediately succeeding); this reads as a one-frame flash, not a
  functional bug, and wasn't worth chasing further for a lab build.
