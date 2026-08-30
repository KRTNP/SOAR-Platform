package httpx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const MinIdempotencyKeyLength = 8

// Querier is the minimal pgx surface idempotency helpers need. Both
// *pgxpool.Pool and pgx.Tx satisfy it, so a handler can look up a key outside
// a transaction and save the result inside one, without httpx depending on
// the store package's concrete types.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type StoredResponse struct {
	RequestHash string
	StatusCode  int
	Body        json.RawMessage
}

func HashBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// LookupIdempotency returns the previously stored response for (tenant,
// route, key), or nil if this key has never been used.
func LookupIdempotency(ctx context.Context, q Querier, tenant, route, key string) (*StoredResponse, error) {
	var out StoredResponse
	err := q.QueryRow(ctx, `
		select request_hash, status_code, response_body
		from idempotency_keys
		where tenant_id = $1 and route = $2 and idempotency_key = $3`,
		tenant, route, key,
	).Scan(&out.RequestHash, &out.StatusCode, &out.Body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SaveIdempotency records a completed response so a retried request with the
// same key replays it instead of repeating the side effect. Call this inside
// the same transaction as the domain write it is guarding.
func SaveIdempotency(ctx context.Context, q Querier, tenant, method, route, key, requestHash string, statusCode int, body []byte) error {
	_, err := q.Exec(ctx, `
		insert into idempotency_keys (tenant_id, http_method, route, idempotency_key, request_hash, status_code, response_body)
		values ($1, $2, $3, $4, $5, $6, $7)`,
		tenant, method, route, key, requestHash, statusCode, body,
	)
	return err
}
