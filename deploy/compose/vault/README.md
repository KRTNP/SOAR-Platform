# Vault (security profile) — one-time local setup

Dev-mode only (adr/0023) — a single unsealed instance with a fixed root
token, no ACL/AppRole. Not committed anywhere: set the token in the
gitignored `docker-compose.override.yml`, matching this repo's existing
lab-credential pattern (the same file already holds the real OPNsense/Wazuh
credentials).

```yaml
# docker-compose.override.yml
services:
  vault:
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: <pick any string, keep it local>
  api:
    environment:
      SECRETS_BACKEND: vault
      VAULT_ADDR: http://localhost:8200
      VAULT_TOKEN: <same value as VAULT_DEV_ROOT_TOKEN_ID>
```

Bring the container up, then seed the four secrets `services/api/secrets`
actually reads (everything else stays a plain env var — see adr/0023):

```sh
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=<same value as VAULT_DEV_ROOT_TOKEN_ID>
vault kv put secret/soar \
  DATABASE_URL="postgres://soar:soar_lab_only@localhost:5432/soar?sslmode=disable" \
  OPNSENSE_API_KEY="..." \
  OPNSENSE_API_SECRET="..." \
  WAZUH_INDEXER_PASSWORD="..."
```

Restart `api` after setting `SECRETS_BACKEND=vault` — a dev-mode Vault's
data (this KV entry included) does not survive a container restart, so
re-run the `vault kv put` above any time the `vault` container itself is
recreated.
