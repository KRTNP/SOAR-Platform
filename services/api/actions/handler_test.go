package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"soar-platform/services/api/genapi"
)

// newPreviewRequest builds a gin.Context carrying the given preview body,
// without a database — Preview only ever reads h.net, never h.pool.
func newPreviewRequest(t *testing.T, body genapi.PreviewActionJSONBody) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/actions/preview", bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, rec
}

func TestHandlerPreview_NoLiveCheck(t *testing.T) {
	h := &Handler{net: NetworkConnector{}} // Preview left nil, as the mock leaves it
	c, rec := newPreviewRequest(t, genapi.PreviewActionJSONBody{Intent: "block_network_indicator", Target: "203.0.113.9"})
	h.Preview(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp genapi.StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Message == nil || *resp.Message == "" {
		t.Fatalf("expected a generic dry-run message, got %v", resp.Message)
	}
}

func TestHandlerPreview_LiveCheckAlreadyInEffect(t *testing.T) {
	var gotTarget string
	h := &Handler{net: NetworkConnector{
		Preview: func(ctx context.Context, target string) (bool, error) {
			gotTarget = target
			return true, nil
		},
	}}
	c, rec := newPreviewRequest(t, genapi.PreviewActionJSONBody{Intent: "block_network_indicator", Target: "203.0.113.9"})
	h.Preview(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if gotTarget != "203.0.113.9" {
		t.Fatalf("Preview called with target %q, want 203.0.113.9", gotTarget)
	}
	var resp genapi.StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Message == nil || !bytes.Contains([]byte(*resp.Message), []byte("already blocked")) {
		t.Fatalf("expected message to mention already blocked, got %v", resp.Message)
	}
}

func TestHandlerPreview_LiveCheckError(t *testing.T) {
	h := &Handler{net: NetworkConnector{
		Preview: func(ctx context.Context, target string) (bool, error) {
			return false, errors.New("connector unreachable")
		},
	}}
	c, rec := newPreviewRequest(t, genapi.PreviewActionJSONBody{Intent: "block_network_indicator", Target: "203.0.113.9"})
	h.Preview(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
}
