//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"soar-platform/services/api/genapi"
)

func createTenant(t *testing.T, srv *httptest.Server, name string) genapi.Tenant {
	t.Helper()
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		t.Fatalf("marshaling tenant request: %v", err)
	}
	resp, err := http.Post(srv.URL+"/v1/tenants", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("creating tenant: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating tenant, got %d", resp.StatusCode)
	}
	var tenant genapi.Tenant
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		t.Fatalf("decoding tenant response: %v", err)
	}
	if tenant.ApiKey == nil || *tenant.ApiKey == "" {
		t.Fatalf("expected an api_key in the create-tenant response")
	}
	return tenant
}

func postEventAsTenant(t *testing.T, srv *httptest.Server, tenantKey, idempotencyKey string, body []byte) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/events", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	if tenantKey != "" {
		req.Header.Set("X-Tenant-Key", tenantKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("posting event: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func getCasesAsTenant(t *testing.T, srv *httptest.Server, tenantKey string) (int, []genapi.Case) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/v1/cases", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	if tenantKey != "" {
		req.Header.Set("X-Tenant-Key", tenantKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("listing cases: %v", err)
	}
	defer resp.Body.Close()
	var page struct {
		Items []genapi.Case `json:"items"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&page)
	return resp.StatusCode, page.Items
}

// TestTenantIsolation_SecondTenantCannotSeeFirstTenantsCases proves
// adr/0022's tenant-key selector actually isolates data between two real
// tenants, not just the schema's tenant_id column existing.
func TestTenantIsolation_SecondTenantCannotSeeFirstTenantsCases(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	tenantA := createTenant(t, srv, "Tenant A")
	tenantB := createTenant(t, srv, "Tenant B")

	if status := postEventAsTenant(t, srv, *tenantA.ApiKey, "tenant-a-evt-0001", fixtureWithObservablesAndClass(t, "tenant-a-evt-0001", "network")); status != http.StatusAccepted {
		t.Fatalf("expected 202 posting event as tenant A, got %d", status)
	}
	if status := postEventAsTenant(t, srv, *tenantB.ApiKey, "tenant-b-evt-0001", fixtureWithObservablesAndClass(t, "tenant-b-evt-0001", "network")); status != http.StatusAccepted {
		t.Fatalf("expected 202 posting event as tenant B, got %d", status)
	}

	if status, casesA := getCasesAsTenant(t, srv, *tenantA.ApiKey); status != http.StatusOK || len(casesA) != 1 {
		t.Fatalf("expected tenant A to see exactly its own 1 case, got status=%d count=%d", status, len(casesA))
	}
	if status, casesB := getCasesAsTenant(t, srv, *tenantB.ApiKey); status != http.StatusOK || len(casesB) != 1 {
		t.Fatalf("expected tenant B to see exactly its own 1 case, got status=%d count=%d", status, len(casesB))
	}
}

// TestTenantKey_UnrecognizedKeyIsRejected proves an unrecognized
// X-Tenant-Key never silently falls back to the default tenant.
func TestTenantKey_UnrecognizedKeyIsRejected(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	status, _ := getCasesAsTenant(t, srv, "not-a-real-key")
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an unrecognized tenant key, got %d", status)
	}
}

// TestTenantKey_NoHeaderKeepsDefaultTenantBehavior proves every existing
// curl/test/UI call that never sends X-Tenant-Key keeps working exactly as
// before this ADR (the "lab" default tenant).
func TestTenantKey_NoHeaderKeepsDefaultTenantBehavior(t *testing.T) {
	srv, _ := setupServer(t)
	defer srv.Close()

	if status := postEventAsTenant(t, srv, "", "no-tenant-header-evt-0001", fixtureWithObservablesAndClass(t, "no-tenant-header-evt-0001", "network")); status != http.StatusAccepted {
		t.Fatalf("expected 202 posting event with no X-Tenant-Key header, got %d", status)
	}
	if status, cases := getCasesAsTenant(t, srv, ""); status != http.StatusOK || len(cases) != 1 {
		t.Fatalf("expected the default 'lab' tenant to see its 1 case, got status=%d count=%d", status, len(cases))
	}
}

// TestIngestion_WritesOutboxRowsForSearchIndex proves adr/0022's outbox
// writes actually happen inside the same transaction as event ingestion —
// services/api/searchindex has something real to project.
func TestIngestion_WritesOutboxRowsForSearchIndex(t *testing.T) {
	srv, pool := setupServer(t)
	defer srv.Close()

	if status := postEventAsTenant(t, srv, "", "outbox-evt-0001", fixtureWithObservablesAndClass(t, "outbox-evt-0001", "network")); status != http.StatusAccepted {
		t.Fatalf("expected 202 posting event, got %d", status)
	}

	rows, err := pool.Query(context.Background(), `select entity_type from outbox_events order by entity_type`)
	if err != nil {
		t.Fatalf("querying outbox_events: %v", err)
	}
	defer rows.Close()
	var entityTypes []string
	for rows.Next() {
		var et string
		if err := rows.Scan(&et); err != nil {
			t.Fatalf("scanning outbox row: %v", err)
		}
		entityTypes = append(entityTypes, et)
	}
	if len(entityTypes) != 2 || entityTypes[0] != "case" || entityTypes[1] != "event" {
		t.Fatalf("expected exactly one 'case' and one 'event' outbox row, got %v", entityTypes)
	}
}
