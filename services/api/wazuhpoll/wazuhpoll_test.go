package wazuhpoll

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSeverityFromLevel(t *testing.T) {
	cases := map[int]string{0: "informational", 3: "informational", 4: "low", 6: "low", 7: "medium", 9: "medium", 10: "high", 12: "high", 13: "critical", 15: "critical"}
	for level, want := range cases {
		if got := severityFromLevel(level); got != want {
			t.Errorf("severityFromLevel(%d) = %q, want %q", level, got, want)
		}
	}
}

// event.schema.json's event_class is a closed enum
// (authentication/network/process/file/email/vulnerability/cloud/malware/
// other) — this asserts every mapped result actually lands in that set, the
// exact validation gap that surfaced against the real lab indexer (rule
// group "ossec" passed through raw was rejected with a 400).
func TestEventClassFromGroups_AlwaysInSchemaEnum(t *testing.T) {
	validClasses := map[string]bool{
		"authentication": true, "network": true, "process": true, "file": true,
		"email": true, "vulnerability": true, "cloud": true, "malware": true, "other": true,
	}
	cases := map[string]string{
		"ossec":                  "other",
		"authentication_failed":  "authentication",
		"sshd":                   "authentication",
		"syscheck":               "file",
		"rootcheck":              "malware",
		"vulnerability-detector": "vulnerability",
		"amazon":                 "cloud",
		"web,attack":             "network",
		"":                       "other",
	}
	for groups, want := range cases {
		var g []string
		if groups != "" {
			g = strings.Split(groups, ",")
		}
		got := eventClassFromGroups(g)
		if got != want {
			t.Errorf("eventClassFromGroups(%v) = %q, want %q", g, got, want)
		}
		if !validClasses[got] {
			t.Errorf("eventClassFromGroups(%v) = %q, not a valid schema enum value", g, got)
		}
	}
}

func TestConfidenceFromLevel_ClampedToUnitInterval(t *testing.T) {
	if c := confidenceFromLevel(15); c != 1 {
		t.Errorf("confidenceFromLevel(15) = %v, want 1", c)
	}
	if c := confidenceFromLevel(0); c != 0 {
		t.Errorf("confidenceFromLevel(0) = %v, want 0", c)
	}
	if c := confidenceFromLevel(30); c != 1 {
		t.Errorf("confidenceFromLevel(30) = %v, want clamped to 1", c)
	}
}

// searchStub serves one wazuh-alerts-*/_search response and captures the
// ingest requests forwarded to it, standing in for both the real Indexer and
// this service's own POST /v1/events in one process — enough to exercise
// Poll's real HTTP round trips without Testcontainers.
type stub struct {
	searchBody     string
	ingestRequests []map[string]any
	ingestStatus   int
}

func newStubServer(t *testing.T, s *stub) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/wazuh-alerts-*/_search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(s.searchBody))
	})
	mux.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding forwarded ingest body: %v", err)
		}
		s.ingestRequests = append(s.ingestRequests, body)
		status := s.ingestStatus
		if status == 0 {
			status = http.StatusAccepted
		}
		w.WriteHeader(status)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestPoll_ForwardsAlertAndAdvancesCheckpoint(t *testing.T) {
	s := &stub{searchBody: `{"hits":{"hits":[
		{"_id":"alert-1","_source":{"@timestamp":"2026-09-02T12:46:47.383Z","rule":{"id":"5710","level":10,"description":"test","groups":["authentication_failed"]},"agent":{"id":"001","name":"lab-host"}}}
	]}}`}
	srv := newStubServer(t, s)

	since := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	newSince, forwarded, err := Poll(t.Context(), srv.Client(), Config{IndexerBaseURL: srv.URL, IndexerUsername: "u", IndexerPassword: "p"}, srv.URL, since, testLogger())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if forwarded != 1 {
		t.Fatalf("forwarded = %d, want 1", forwarded)
	}
	wantSince := time.Date(2026, 9, 2, 12, 46, 47, 383000000, time.UTC)
	if !newSince.Equal(wantSince) {
		t.Errorf("newSince = %v, want %v", newSince, wantSince)
	}
	if len(s.ingestRequests) != 1 {
		t.Fatalf("ingest requests = %d, want 1", len(s.ingestRequests))
	}
	payload, _ := s.ingestRequests[0]["payload"].(map[string]any)
	if payload["source_event_id"] != "alert-1" {
		t.Errorf("source_event_id = %v, want alert-1", payload["source_event_id"])
	}
	if payload["severity"] != "high" {
		t.Errorf("severity = %v, want high (level 10)", payload["severity"])
	}
	if payload["event_class"] != "authentication" {
		t.Errorf("event_class = %v, want authentication (mapped from authentication_failed)", payload["event_class"])
	}
}

func TestPoll_NoObservablesWhenNoDataFields(t *testing.T) {
	s := &stub{searchBody: `{"hits":{"hits":[
		{"_id":"alert-2","_source":{"@timestamp":"2026-09-02T12:46:47Z","rule":{"id":"502","level":3,"description":"manager started"}}}
	]}}`}
	srv := newStubServer(t, s)

	_, _, err := Poll(t.Context(), srv.Client(), Config{IndexerBaseURL: srv.URL, IndexerUsername: "u", IndexerPassword: "p"}, srv.URL, time.Time{}, testLogger())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	payload, _ := s.ingestRequests[0]["payload"].(map[string]any)
	if _, present := payload["observables"]; present {
		t.Errorf("expected no observables field, got %v", payload["observables"])
	}
}

func TestPoll_ExtractsObservablesFromDataFields(t *testing.T) {
	s := &stub{searchBody: `{"hits":{"hits":[
		{"_id":"alert-3","_source":{"@timestamp":"2026-09-02T12:46:47Z","rule":{"id":"5710","level":10},"data":{"srcip":"198.51.100.10","srcuser":"root"}}}
	]}}`}
	srv := newStubServer(t, s)

	_, _, err := Poll(t.Context(), srv.Client(), Config{IndexerBaseURL: srv.URL, IndexerUsername: "u", IndexerPassword: "p"}, srv.URL, time.Time{}, testLogger())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	payload, _ := s.ingestRequests[0]["payload"].(map[string]any)
	observables, _ := payload["observables"].([]any)
	if len(observables) != 2 {
		t.Fatalf("observables = %d, want 2 (srcip + srcuser)", len(observables))
	}
}

func TestPoll_SkipsFailedForwardButStillAdvancesCheckpoint(t *testing.T) {
	s := &stub{
		ingestStatus: http.StatusInternalServerError,
		searchBody: `{"hits":{"hits":[
			{"_id":"alert-4","_source":{"@timestamp":"2026-09-02T13:00:00Z","rule":{"id":"1","level":1}}}
		]}}`,
	}
	srv := newStubServer(t, s)

	since := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	newSince, forwarded, err := Poll(t.Context(), srv.Client(), Config{IndexerBaseURL: srv.URL, IndexerUsername: "u", IndexerPassword: "p"}, srv.URL, since, testLogger())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if forwarded != 0 {
		t.Errorf("forwarded = %d, want 0 (ingest failed)", forwarded)
	}
	if !newSince.After(since) {
		t.Errorf("checkpoint did not advance past the failed alert's timestamp: %v", newSince)
	}
}

func TestPoll_NoHitsIsANoop(t *testing.T) {
	s := &stub{searchBody: `{"hits":{"hits":[]}}`}
	srv := newStubServer(t, s)

	since := time.Now().UTC()
	newSince, forwarded, err := Poll(t.Context(), srv.Client(), Config{IndexerBaseURL: srv.URL, IndexerUsername: "u", IndexerPassword: "p"}, srv.URL, since, testLogger())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if forwarded != 0 || !newSince.Equal(since) {
		t.Errorf("expected a no-op, got forwarded=%d newSince=%v", forwarded, newSince)
	}
}
