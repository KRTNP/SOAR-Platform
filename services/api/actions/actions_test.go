package actions

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeQuerier is a no-op querier for exercising expireIfStale's pure
// decision logic without a real database.
type fakeQuerier struct{}

func (fakeQuerier) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return nil }
func (fakeQuerier) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (fakeQuerier) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestExpireIfStale(t *testing.T) {
	tests := []struct {
		name       string
		apStatus   string
		expiresAt  time.Time
		wantStatus string
	}{
		{"pending and expired", "pending", time.Now().Add(-time.Minute), "expired"},
		{"pending and not expired", "pending", time.Now().Add(time.Hour), "pending"},
		{"approved and past expiry", "approved", time.Now().Add(-time.Minute), "approved"},
		{"rejected and past expiry", "rejected", time.Now().Add(-time.Minute), "rejected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Action{ID: "act-1", Status: "pending_approval"}
			if tt.apStatus != "pending" {
				a.Status = tt.apStatus
			}
			ap := &Approval{ID: "apr-1", Status: tt.apStatus, ExpiresAt: tt.expiresAt}
			if err := expireIfStale(t.Context(), fakeQuerier{}, a, ap); err != nil {
				t.Fatalf("expireIfStale: %v", err)
			}
			if ap.Status != tt.wantStatus {
				t.Errorf("approval status = %q, want %q", ap.Status, tt.wantStatus)
			}
		})
	}
}

func TestIdempotencyKey(t *testing.T) {
	got := idempotencyKey("case-001", "ipv4", "198.51.100.10")
	want := "case-001:block:ipv4:198.51.100.10"
	if got != want {
		t.Errorf("idempotencyKey() = %q, want %q", got, want)
	}
}
