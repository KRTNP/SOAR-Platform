// Package secrets abstracts where the four genuinely secret values this
// service reads (DATABASE_URL, OPNSENSE_API_KEY, OPNSENSE_API_SECRET,
// WAZUH_INDEXER_PASSWORD — see adr/0023) come from. Everything else
// (URLs, ports, *_ENABLE_REAL flags, intervals) stays plain os.Getenv,
// unchanged, in each package's own LoadConfig.
package secrets

import (
	"context"
	"fmt"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
)

// Provider resolves a secret by its existing environment variable name
// (e.g. "DATABASE_URL"), so callers don't need to know which backend is
// active.
type Provider interface {
	Get(ctx context.Context, key string) (string, error)
}

// New selects a Provider by the SECRETS_BACKEND env var: "env" (default,
// today's os.Getenv behavior) or "vault" (opt-in), mirroring the
// *_ENABLE_REAL toggle convention already used by opnsense/wazuhpoll/
// searchindex — nothing changes for anyone not opting in.
func New() (Provider, error) {
	switch os.Getenv("SECRETS_BACKEND") {
	case "", "env":
		return envProvider{}, nil
	case "vault":
		return newVaultProvider()
	default:
		return nil, fmt.Errorf("SECRETS_BACKEND must be %q or %q, got %q", "env", "vault", os.Getenv("SECRETS_BACKEND"))
	}
}

// envProvider wraps os.Getenv — today's behavior, the default.
type envProvider struct{}

func (envProvider) Get(_ context.Context, key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

// vaultProvider reads secrets from a single lab-scoped KV-v2 path
// (secret/data/soar) — dev-mode Vault only, no unsealing/ACL/AppRole (see
// adr/0023). A connectivity error surfaces at the first Get, not here at
// construction, matching LoadConfig's existing "only fail when a value is
// actually missing" posture.
type vaultProvider struct {
	client *vaultapi.Client
}

const vaultMount = "secret"
const vaultPath = "soar"

func newVaultProvider() (Provider, error) {
	cfg := vaultapi.DefaultConfig()
	if err := cfg.ReadEnvironment(); err != nil {
		return nil, fmt.Errorf("reading Vault environment (VAULT_ADDR/VAULT_TOKEN): %w", err)
	}
	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("building Vault client: %w", err)
	}
	return &vaultProvider{client: client}, nil
}

func (v *vaultProvider) Get(ctx context.Context, key string) (string, error) {
	secret, err := v.client.KVv2(vaultMount).Get(ctx, vaultPath)
	if err != nil {
		return "", fmt.Errorf("reading %s/%s from Vault: %w", vaultMount, vaultPath, err)
	}
	value, ok := secret.Data[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("%s not found at %s/%s in Vault", key, vaultMount, vaultPath)
	}
	return value, nil
}
