// Package outbox implements the transactional-outbox half of
// docs/02-architecture/data-architecture.md's already-decided "PostgreSQL
// source of truth, OpenSearch derived projection" design (adr/0022):
// Write records one row in the same transaction as the domain write it
// documents, so services/api/searchindex's poller can never observe a
// projection value that was never actually committed.
package outbox

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oklog/ulid/v2"
)

// querier is satisfied by pgx.Tx (Write is always called from inside an
// already-open domain transaction — see events.Handler.Ingest).
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Write records payload for entityType/entityID so searchindex.Run projects
// it into OpenSearch's soar-<entityType>s-v1 index (data-architecture.md's
// naming). payload should already be the exact document body — this
// function does not shape it further.
func Write(ctx context.Context, q querier, tenantID, entityType, entityID string, payload map[string]any, now time.Time) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	id := "outbox-" + strings.ToLower(ulid.Make().String())
	_, err = q.Exec(ctx, `
		insert into outbox_events (id, tenant_id, entity_type, entity_id, payload, created_at)
		values ($1,$2,$3,$4,$5,$6)`,
		id, tenantID, entityType, entityID, body, now,
	)
	return err
}
