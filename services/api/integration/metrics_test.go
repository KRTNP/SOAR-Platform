//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestMetrics_ReturnsASnapshot: a light sanity check, not a full test matrix
// per metric — GET /v1/metrics returns 200 with the expected top-level
// shape over whatever this lab-scale run has already recorded.
func TestMetrics_ReturnsASnapshot(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	caseID := caseWithBlockTarget(t, srv, "metrics-snapshot-0001")
	action := requestBlock(t, srv, caseID, "198.51.100.10")
	approveApproval(t, srv, findApprovalForAction(t, srv, action.Id).Id).Body.Close()
	executeAction(t, srv, action.Id).Body.Close()

	resp, err := http.Get(srv.URL + "/v1/metrics")
	if err != nil {
		t.Fatalf("GET /v1/metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var snap struct {
		ConnectorReliability []struct {
			ConnectorVersion string `json:"connector_version"`
			Allowed          int    `json:"allowed"`
		} `json:"connector_reliability"`
		AnalystOverrideCount int `json:"analyst_override_count"`
		TotalCases           int `json:"total_cases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if snap.TotalCases < 1 {
		t.Errorf("expected at least 1 case counted, got %d", snap.TotalCases)
	}
	if len(snap.ConnectorReliability) != 1 || snap.ConnectorReliability[0].Allowed != 1 {
		t.Errorf("expected exactly one connector_version with 1 allowed execute, got %+v", snap.ConnectorReliability)
	}
}
