// Package tenants implements adr/0022's lab-appropriate tenant *selector* —
// not authentication. It gives services/api/httpx.TenantKeyMiddleware
// something real to resolve an X-Tenant-Key header against, so more than
// one tenant can actually exist and be exercised (previously every request
// resolved to the same hardcoded "lab" tenant_id — see
// services/api/config.Config.DefaultTenantID's own doc comment). There is
// no identity, no session, no RBAC here; a key only ever selects which
// tenant_id a request's data belongs to.
package tenants

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
)

var ErrNotFound = errors.New("tenant not found")

// querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Tenant struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// hashKey is a plain SHA-256 digest, not a slow password hash (bcrypt/
// scrypt/argon2) — this is a lab tenant *selector*, not a credential
// protecting anything a real attacker would brute-force offline; matching
// this codebase's existing posture of disclosing exactly what security
// property is (and isn't) provided rather than adding cryptographic weight
// a lab doesn't need.
func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Create inserts a new tenant and returns it along with the plaintext API
// key — the only time the plaintext is ever available; only its hash is
// stored.
func Create(ctx context.Context, q querier, name string, now time.Time) (Tenant, string, error) {
	id := "tenant-" + strings.ToLower(ulid.Make().String())
	keyBytes := make([]byte, 24)
	if _, err := rand.Read(keyBytes); err != nil {
		return Tenant{}, "", fmt.Errorf("generating tenant api key: %w", err)
	}
	key := hex.EncodeToString(keyBytes)

	if _, err := q.Exec(ctx, `
		insert into tenants (id, name, api_key_hash, created_at) values ($1,$2,$3,$4)`,
		id, name, hashKey(key), now,
	); err != nil {
		return Tenant{}, "", fmt.Errorf("creating tenant: %w", err)
	}
	return Tenant{ID: id, Name: name, CreatedAt: now}, key, nil
}

// List returns every tenant, newest first. Tenant management has no
// tenant_id scope of its own (a tenant is not scoped to a tenant) — the
// same posture a future real auth layer would still need for whichever
// role manages tenants.
func List(ctx context.Context, q querier) ([]Tenant, error) {
	rows, err := q.Query(ctx, `select id, name, created_at from tenants order by created_at desc`)
	if err != nil {
		return nil, fmt.Errorf("listing tenants: %w", err)
	}
	defer rows.Close()

	var out []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Resolve returns the tenant id whose api_key_hash matches key, or
// ErrNotFound if no tenant has that key.
func Resolve(ctx context.Context, q querier, key string) (string, error) {
	var id string
	err := q.QueryRow(ctx, `select id from tenants where api_key_hash = $1`, hashKey(key)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}
