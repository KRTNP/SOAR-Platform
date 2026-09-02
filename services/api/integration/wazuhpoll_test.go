//go:build integration

package integration

import (
	"testing"
	"time"

	"soar-platform/services/api/wazuhpoll"
)

// TestWazuhCheckpoint_PersistsAcrossRuns: adr/0018 replaced wazuhpoll's
// in-memory-only checkpoint (reset to time.Now() on every process restart,
// silently dropping any alert that arrived during the restart window) with
// a single-row Postgres table. This drives LoadCheckpoint/SaveCheckpoint
// directly against a real database, the same way scheduler_test.go-style
// tests exercise their own package against setupPool.
func TestWazuhCheckpoint_PersistsAcrossRuns(t *testing.T) {
	pool := setupPool(t)
	ctx := t.Context()

	// No row yet: LoadCheckpoint falls back to "now" rather than erroring or
	// zero-valuing (a zero time would re-poll the indexer's entire history).
	before := time.Now().UTC()
	got, err := wazuhpoll.LoadCheckpoint(ctx, pool)
	if err != nil {
		t.Fatalf("LoadCheckpoint with no row: %v", err)
	}
	if got.Before(before) {
		t.Errorf("LoadCheckpoint with no row = %v, want a fallback at/after %v", got, before)
	}

	want := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	if err := wazuhpoll.SaveCheckpoint(ctx, pool, want); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}
	got, err = wazuhpoll.LoadCheckpoint(ctx, pool)
	if err != nil {
		t.Fatalf("LoadCheckpoint after save: %v", err)
	}
	if !got.Equal(want) {
		t.Errorf("LoadCheckpoint after save = %v, want %v", got, want)
	}

	// A second save overwrites the single row rather than erroring or
	// inserting a second one — this is what makes a restart resume from the
	// last real checkpoint instead of the first one ever recorded.
	later := want.Add(time.Hour)
	if err := wazuhpoll.SaveCheckpoint(ctx, pool, later); err != nil {
		t.Fatalf("SaveCheckpoint (overwrite): %v", err)
	}
	got, err = wazuhpoll.LoadCheckpoint(ctx, pool)
	if err != nil {
		t.Fatalf("LoadCheckpoint after overwrite: %v", err)
	}
	if !got.Equal(later) {
		t.Errorf("LoadCheckpoint after overwrite = %v, want %v", got, later)
	}
}
