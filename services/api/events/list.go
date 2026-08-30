package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Summary is the shape returned by GET /v1/events — the full Observable graph
// isn't needed for a list view, only what identifies and classifies the event.
type Summary struct {
	ID            string    `json:"id"`
	Source        string    `json:"source"`
	SourceEventID *string   `json:"source_event_id,omitempty"`
	EventClass    string    `json:"event_class"`
	Severity      string    `json:"severity"`
	Confidence    float64   `json:"confidence"`
	ObservedAt    time.Time `json:"observed_at"`
	ReceivedAt    time.Time `json:"received_at"`
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// List returns up to limit events for tenantID older than cursor (an event
// id), newest first. IDs are ULIDs, so lexicographic order is chronological —
// no separate timestamp cursor encoding is needed.
func List(ctx context.Context, q querier, tenantID, cursor string, limit int) ([]Summary, string, error) {
	var rows pgx.Rows
	var err error
	if cursor == "" {
		rows, err = q.Query(ctx, `
			select id, source, source_event_id, event_class, severity, confidence, observed_at, received_at
			from events where tenant_id = $1 order by id desc limit $2`, tenantID, limit+1)
	} else {
		rows, err = q.Query(ctx, `
			select id, source, source_event_id, event_class, severity, confidence, observed_at, received_at
			from events where tenant_id = $1 and id < $2 order by id desc limit $3`, tenantID, cursor, limit+1)
	}
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var out []Summary
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.ID, &s.Source, &s.SourceEventID, &s.EventClass, &s.Severity, &s.Confidence, &s.ObservedAt, &s.ReceivedAt); err != nil {
			return nil, "", err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(out) > limit {
		nextCursor = out[limit-1].ID
		out = out[:limit]
	}
	return out, nextCursor, nil
}
