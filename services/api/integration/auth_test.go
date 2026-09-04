//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/auth"
	"soar-platform/services/api/cases"
	"soar-platform/services/api/events"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/tenants"
)

// This suite exercises adr/0024's real code paths (JWT verification, role
// enforcement) without a live Keycloak — a locally generated RSA keypair
// stands in for Keycloak's own signing key, and keyfunc.NewJWKSetJSON
// builds a Keyfunc directly from a JWK Set blob rather than serving one
// over HTTP, since what's under test here is this service's verification
// and RBAC logic, not the JWKS-fetch transport (a third-party library
// concern already covered by keyfunc's own tests).

const testKID = "test-key-1"

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// jwksJSON returns a JWK Set (RFC 7517) exposing pub's public key under
// testKID, in the shape keyfunc.NewJWKSetJSON expects.
func jwksJSON(t *testing.T, pub *rsa.PublicKey) []byte {
	t.Helper()
	set := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA", "kid": testKID, "use": "sig", "alg": "RS256",
				"n": b64url(pub.N.Bytes()),
				"e": b64url(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
	raw, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshaling JWK set: %v", err)
	}
	return raw
}

// mintToken signs a Keycloak-shaped access token with priv, under testKID.
func mintToken(t *testing.T, priv *rsa.PrivateKey, subject, email string, roles []string, expiresAt time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   subject,
		"email": email,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Add(-time.Minute).Unix(),
		"realm_access": map[string]any{
			"roles": roles,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKID
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("signing test token: %v", err)
	}
	return signed
}

func newTestVerifier(t *testing.T, pub *rsa.PublicKey) *auth.Verifier {
	t.Helper()
	k, err := keyfunc.NewJWKSetJSON(jwksJSON(t, pub))
	if err != nil {
		t.Fatalf("building test Keyfunc from JWK set: %v", err)
	}
	return auth.NewVerifierWithKeyfunc(k)
}

// setupAuthServer is a narrower setupServer: only the handlers adr/0024's
// tests need (events, cases, actions), with the real auth.Verifier
// middleware mounted — unlike every other test in this suite, which stays
// on setupServer's unauthenticated router (AUTH_ENABLE_REAL defaults off).
func setupAuthServer(t *testing.T, verifier *auth.Verifier) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()
	pool := setupPool(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	resolveTenant := func(ctx context.Context, key string) (string, error) { return tenants.Resolve(ctx, pool, key) }
	r.Use(httpx.RequestIDMiddleware(), httpx.TenantKeyMiddleware("lab", resolveTenant), verifier.Middleware())

	eventsH := events.NewHandler(pool, nil)
	casesH := cases.NewHandler(pool, nil)
	netConn := mockNetworkConnector()
	actionsH := actions.NewHandler(pool, netConn, nil)

	v1 := r.Group("/v1")
	v1.POST("/events", eventsH.Ingest)
	v1.GET("/cases", casesH.List)
	v1.GET("/cases/:caseId", casesH.Get)
	v1.PATCH("/cases/:caseId", casesH.Update)
	v1.GET("/approvals", actionsH.ListApprovals)
	v1.POST("/approvals/:approvalId/approve", actionsH.Approve)
	v1.POST("/approvals/:approvalId/reject", actionsH.Reject)

	return httptest.NewServer(r), pool
}

func bearerRequest(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Idempotency-Key", "auth-test-key-"+method+"-"+url)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func TestAuth_MissingOrInvalidTokenRejected(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating test RSA key: %v", err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating second test RSA key: %v", err)
	}
	verifier := newTestVerifier(t, &priv.PublicKey)
	srv, _ := setupAuthServer(t, verifier)
	defer srv.Close()

	// No token at all.
	resp := bearerRequest(t, http.MethodGet, srv.URL+"/v1/cases", "", "")
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with no token, got %d", resp.StatusCode)
	}

	// Expired token, correctly signed.
	expired := mintToken(t, priv, "user-1", "analyst@lab.local", []string{"analyst"}, time.Now().Add(-time.Hour))
	resp = bearerRequest(t, http.MethodGet, srv.URL+"/v1/cases", "", expired)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with an expired token, got %d", resp.StatusCode)
	}

	// Valid claims, wrong signing key (not in the JWKS the verifier trusts).
	wrongSig := mintToken(t, other, "user-1", "analyst@lab.local", []string{"analyst"}, time.Now().Add(time.Hour))
	resp = bearerRequest(t, http.MethodGet, srv.URL+"/v1/cases", "", wrongSig)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with a bad-signature token, got %d", resp.StatusCode)
	}

	// A valid token succeeds, proving the middleware isn't just rejecting everything.
	valid := mintToken(t, priv, "user-1", "analyst@lab.local", []string{"analyst"}, time.Now().Add(time.Hour))
	resp = bearerRequest(t, http.MethodGet, srv.URL+"/v1/cases", "", valid)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with a valid token, got %d", resp.StatusCode)
	}
}

func TestAuth_ApproveRequiresTheRolePolicyNames(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating test RSA key: %v", err)
	}
	verifier := newTestVerifier(t, &priv.PublicKey)
	srv, _ := setupAuthServer(t, verifier)
	defer srv.Close()

	analystToken := mintToken(t, priv, "analyst-1", "analyst@lab.local", []string{"analyst"}, time.Now().Add(time.Hour))
	commanderToken := mintToken(t, priv, "ic-1", "ic@lab.local", []string{"incident-commander"}, time.Now().Add(time.Hour))

	// Ingest an event with an ipv4 observable, same fixture shape the rest
	// of this suite's approval tests use, to get a case with a
	// block_network_indicator action pending approval (requires
	// "incident-commander" per actions/policy.go).
	ingestResp := bearerRequest(t, http.MethodPost, srv.URL+"/v1/events", string(fixtureWithObservablesAndClass(t, "auth-rbac-0001", "authentication")), analystToken)
	ingestResp.Body.Close()
	if ingestResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 ingesting the fixture event, got %d", ingestResp.StatusCode)
	}

	casesResp := bearerRequest(t, http.MethodGet, srv.URL+"/v1/cases", "", analystToken)
	var page struct {
		Items []struct {
			Id string `json:"id"`
		} `json:"items"`
	}
	_ = json.NewDecoder(casesResp.Body).Decode(&page)
	casesResp.Body.Close()
	if len(page.Items) == 0 {
		t.Fatal("expected at least one case after ingestion")
	}
	caseID := page.Items[0].Id

	blockResp := bearerRequest(t, http.MethodPatch, srv.URL+"/v1/cases/"+caseID,
		`{"operation":"request_block","target_value":"198.51.100.10"}`, analystToken)
	blockResp.Body.Close()
	if blockResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 requesting a block, got %d", blockResp.StatusCode)
	}

	approvalsResp := bearerRequest(t, http.MethodGet, srv.URL+"/v1/approvals", "", analystToken)
	var approvalsPage struct {
		Items []struct {
			Id string `json:"id"`
		} `json:"items"`
	}
	_ = json.NewDecoder(approvalsResp.Body).Decode(&approvalsPage)
	approvalsResp.Body.Close()
	if len(approvalsPage.Items) == 0 {
		t.Fatal("expected a pending approval after request_block")
	}
	approvalID := approvalsPage.Items[0].Id

	// An analyst-only token cannot approve a block_network_indicator (needs
	// incident-commander) — the enforcement adr/0024 adds on top of
	// EvaluatePolicy's previously-unenforced approvalRoles.
	denied := bearerRequest(t, http.MethodPost, srv.URL+"/v1/approvals/"+approvalID+"/approve", "", analystToken)
	denied.Body.Close()
	if denied.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 approving with only the analyst role, got %d", denied.StatusCode)
	}

	// An incident-commander token succeeds.
	allowed := bearerRequest(t, http.MethodPost, srv.URL+"/v1/approvals/"+approvalID+"/approve", "", commanderToken)
	defer allowed.Body.Close()
	if allowed.StatusCode != http.StatusOK {
		t.Errorf("expected 200 approving with the incident-commander role, got %d", allowed.StatusCode)
	}
}
