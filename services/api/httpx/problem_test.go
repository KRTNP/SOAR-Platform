package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"soar-platform/services/api/genapi"
)

func TestHashBody_Deterministic(t *testing.T) {
	a := HashBody([]byte(`{"a":1}`))
	b := HashBody([]byte(`{"a":1}`))
	if a != b {
		t.Errorf("HashBody is not deterministic: %q != %q", a, b)
	}
	if c := HashBody([]byte(`{"a":2}`)); c == a {
		t.Error("HashBody produced the same hash for different bodies")
	}
}

func writeAndDecode(t *testing.T, err error) (int, genapi.Problem) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(string(requestIDKey), "req-test-0001")

	WriteProblem(c, err)

	var problem genapi.Problem
	if decErr := json.NewDecoder(w.Body).Decode(&problem); decErr != nil {
		t.Fatalf("decoding Problem body: %v", decErr)
	}
	return w.Code, problem
}

func TestWriteProblem_AppError(t *testing.T) {
	appErr := ErrNotFound("case_not_found", "case does not exist").WithDetail("no case with that id")
	status, problem := writeAndDecode(t, appErr)

	if status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", status, http.StatusNotFound)
	}
	if problem.Code != "case_not_found" {
		t.Errorf("code = %q, want %q", problem.Code, "case_not_found")
	}
	if problem.RequestId != "req-test-0001" {
		t.Errorf("request_id = %q, want it threaded from context", problem.RequestId)
	}
	if problem.Detail == nil || *problem.Detail != "no case with that id" {
		t.Errorf("detail = %v, want the WithDetail value", problem.Detail)
	}
	if problem.Retryable {
		t.Error("a not-found error should not be marked retryable")
	}
}

func TestWriteProblem_UnclassifiedError(t *testing.T) {
	// A plain error (e.g. a raw pgx failure) must never leak its message to
	// the client — only a stable internal_error code.
	status, problem := writeAndDecode(t, errors.New("pq: connection refused, password=hunter2"))

	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", status)
	}
	if problem.Code != "internal_error" {
		t.Errorf("code = %q, want %q", problem.Code, "internal_error")
	}
	if problem.Title == "connection refused, password=hunter2" {
		t.Error("raw error text leaked into the client-visible Problem title")
	}
	if !problem.Retryable {
		t.Error("an unclassified 5xx should default to retryable")
	}
}
