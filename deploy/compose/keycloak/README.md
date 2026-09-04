# Keycloak (security profile) — one-time local setup

Dev-mode only (adr/0024) — a single realm (`soar`), no per-tenant realm
separation, `start-dev --import-realm` auto-imports `realm-soar.json` on
first boot. Not committed anywhere: set the admin console password in the
gitignored `docker-compose.override.yml`, matching this repo's existing
lab-credential pattern.

```yaml
# docker-compose.override.yml
services:
  keycloak:
    environment:
      KEYCLOAK_ADMIN_PASSWORD: <pick any string, keep it local>
```

Admin console: `http://localhost:8081` (user `admin`). The `soar` realm's
three realm roles (`analyst`, `incident-commander`, `service-owner`) match
`services/api/actions/policy.go`'s existing vocabulary exactly — see
adr/0024 for why this doesn't attempt to model the broader tier list in
`docs/06-security/authentication-and-rbac.md`.

## Test users

`realm-soar.json` seeds one user per role — lab-only fixed passwords,
same low-stakes posture as this stack's other committed lab credentials
(e.g. Postgres's `soar_lab_only`):

| Email | Password | Role |
|---|---|---|
| `analyst@lab.local` | `lab-analyst-pw` | `analyst` |
| `ic@lab.local` | `lab-ic-pw` | `incident-commander` |
| `owner@lab.local` | `lab-owner-pw` | `service-owner` |

## Enabling real enforcement on the API

```yaml
# docker-compose.override.yml
services:
  api:
    environment:
      AUTH_ENABLE_REAL: "true"
      AUTH_KEYCLOAK_ISSUER_URL: "http://localhost:8081/realms/soar"
```

With this off (the default), every approval/audit call site keeps its
pre-adr/0024 fixed-literal-actor behavior — nothing else changes.
