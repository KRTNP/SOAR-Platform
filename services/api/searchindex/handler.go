package searchindex

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"soar-platform/services/api/httpx"
)

// Handler implements GET /v1/search — a thin proxy over OpenSearch's
// multi-index _search API. Only mounted (services/api/main.go) when
// Enabled() is true; there is no fallback search over Postgres.
type Handler struct {
	baseURL string
	client  *http.Client
}

func NewHandler(cfg Config) *Handler {
	return &Handler{
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}, //nolint:gosec // lab-only, explicit opt-in
			},
		},
	}
}

type openSearchHit struct {
	Index  string          `json:"_index"`
	ID     string          `json:"_id"`
	Source json.RawMessage `json:"_source"`
}

type openSearchResponse struct {
	Hits struct {
		Hits []openSearchHit `json:"hits"`
	} `json:"hits"`
}

// Search implements GET /v1/search?q=... — a multi_match query across
// soar-cases-v1 and soar-events-v1, filtered to the caller's resolved
// tenant_id. This tenant filter is defense in depth, not a real security
// boundary: the lab OpenSearch this targets runs with
// plugins.security.disabled=true (adr/0022), matching
// docs/02-architecture/data-architecture.md's own "must not be the sole
// source of truth" caveat for this whole projection.
func (h *Handler) Search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		httpx.WriteProblem(c, httpx.ErrValidation("missing_query", "q is required"))
		return
	}
	tenant := httpx.TenantID(c)

	reqBody, err := json.Marshal(map[string]any{
		"size": 25,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{{"multi_match": map[string]any{"query": q}}},
				// tenant_id.keyword, not tenant_id: OpenSearch's dynamic
				// mapping makes tenant_id an analyzed text field (tokenized
				// on "-"), so a term filter against the bare field name
				// never matches the literal id — only its unanalyzed
				// .keyword sub-field does (confirmed against the real
				// index; a bare-field filter silently matched nothing).
				"filter": []map[string]any{{"term": map[string]any{"tenant_id.keyword": tenant}}},
			},
		},
	})
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("search_query_failed", "could not build search query").WithCause(err))
		return
	}

	url := h.baseURL + "/soar-cases-v1,soar-events-v1/_search"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("search_query_failed", "could not build search request").WithCause(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("search_unavailable", "could not reach the search index").WithCause(err))
		return
	}
	defer resp.Body.Close()

	var out openSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		httpx.WriteProblem(c, httpx.ErrInternal("search_decode_failed", "could not decode search results").WithCause(err))
		return
	}

	items := make([]gin.H, 0, len(out.Hits.Hits))
	for _, hit := range out.Hits.Hits {
		items = append(items, gin.H{"index": hit.Index, "id": hit.ID, "source": hit.Source})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
