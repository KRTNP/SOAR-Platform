# ADR-0023: Vault secrets for lab credentials

## Status

Accepted — implemented and verified against a real running dev-mode Vault
container.

## Decision

`adr/0005` picked HashiCorp Vault for secrets in one line, with no further
design anywhere in the repo. Today every credential — the DB password, the
OPNsense API key/secret, the Wazuh indexer password — lives as a plaintext
environment variable in the gitignored `docker-compose.override.yml` and is
read via a scattered `os.Getenv` call in whichever package needs it. This
ADR gives those four values (and only those four — everything else read
via `os.Getenv`, such as URLs, ports, and `*_ENABLE_REAL` flags, stays a
plain environment variable) an optional path through Vault instead,
without changing default behavior for anyone not opting in.

- New `services/api/secrets` package: a `Provider` interface
  (`Get(ctx, key string) (string, error)`) with two implementations —
  `envProvider` (wraps `os.Getenv`, today's behavior, the default) and
  `vaultProvider` (a KV-v2 read at a single fixed lab path,
  `secret/data/soar`, via `github.com/hashicorp/vault/api`). `secrets.New()`
  selects between them via a `SECRETS_BACKEND` env var (`env` default,
  `vault` opt-in), the same toggle convention `opnsense.Enabled()` /
  `wazuhpoll.Enabled()` / `searchindex.Enabled()` already use for real vs.
  mock integrations.
- `config.Load`, `opnsense.LoadConfig`, and `wazuhpoll.LoadConfig` each gain
  a `(ctx, secrets.Provider)` parameter, used only for their genuinely
  secret fields (`DATABASE_URL`; `OPNSENSE_API_KEY`/`OPNSENSE_API_SECRET`;
  `WAZUH_INDEXER_PASSWORD`) — every other field in those same functions is
  unchanged. `main.go` constructs one `secrets.Provider` near the top of
  `run()` and threads it through; `cmd/backfill-search` does the same for
  its own `DATABASE_URL` read.
- New `vault` service in `docker-compose.core.yml`'s `security` profile
  (`hashicorp/vault:1.18`, `server -dev`), filling in a placeholder that
  file already reserved. Dev-mode only: a single unsealed instance with a
  fixed root token supplied only via the gitignored override file — no
  unsealing, ACL, or AppRole design here, deliberately out of scope for a
  lab (see Consequences). `deploy/compose/vault/README.md` documents the
  one-time `vault kv put secret/soar ...` seeding step.

## Verification

`go build/vet/test ./...` with the default `env` backend — zero behavior
change, full existing unit and integration suite green
(`go test -tags integration ./integration/...`, including the tenant
isolation and case-projection tests from adr/0022). Against the real
stack: brought up the `security` profile's `vault` container, seeded the
four secrets via the README's `vault kv put` command, set
`SECRETS_BACKEND=vault` plus `VAULT_ADDR`/`VAULT_TOKEN` in the override,
restarted `api`, and confirmed a real event ingest still completed
end-to-end — proving `DATABASE_URL` resolved correctly through Vault
rather than the environment.

## Consequences

- **Dev-mode Vault only.** No unsealing, no ACL policies, no AppRole/
  Kubernetes auth — a single root-token-authenticated instance, matching
  this repo's existing lab-credential posture for e.g. the OpenSearch admin
  password and Wazuh basic auth already in this Compose stack. A dev-mode
  Vault's data does not survive a container restart, so the seeded secret
  must be re-`put` any time the `vault` container itself is recreated —
  documented in the README, not automated.
- **Only four values move.** Everything else this service reads from the
  environment (URLs, ports, `*_ENABLE_REAL` flags, poll intervals) stays a
  plain env var — Vault is not a general config store here.
- **Opt-in, not a cutover.** `SECRETS_BACKEND` defaults to `env`; nothing
  about today's deployment changes unless it's explicitly set to `vault`.
