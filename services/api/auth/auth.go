// Package auth verifies Keycloak-issued bearer JWTs and enforces
// services/api/actions.EvaluatePolicy's approvalRoles against the caller's
// real realm roles (adr/0024). Gated behind AUTH_ENABLE_REAL, default off —
// with auth off, Middleware is never mounted and FromContext always
// returns its zero value, so every call site that already has a
// literal-actor fallback keeps today's exact behavior.
package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"soar-platform/services/api/httpx"
)

// Enabled reports whether the caller asked for real Keycloak verification —
// same master-switch convention as opnsense.Enabled()/wazuhpoll.Enabled().
func Enabled() bool {
	return os.Getenv("AUTH_ENABLE_REAL") == "true"
}

// Claims is the subset of a Keycloak access token this service cares about.
type Claims struct {
	Subject string
	Email   string
	Roles   []string
}

// HasRole reports whether c.Roles contains any of the given roles.
func (c Claims) HasRole(roles ...string) bool {
	for _, want := range roles {
		for _, have := range c.Roles {
			if have == want {
				return true
			}
		}
	}
	return false
}

// keycloakClaims is the on-the-wire shape of a Keycloak access token's
// claims this service reads, layered on jwt.RegisteredClaims for
// signature/expiry verification.
type keycloakClaims struct {
	jwt.RegisteredClaims
	Email       string `json:"email"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// Verifier holds a JWKS keyset kept fresh in the background (keyfunc's own
// "refresh goroutine", stopped when ctx is canceled) and verifies bearer
// tokens against it.
type Verifier struct {
	jwks keyfunc.Keyfunc
}

// NewVerifier builds a Verifier against issuerURL's realm JWKS endpoint
// (Keycloak's standard "{issuer}/protocol/openid-connect/certs" path).
func NewVerifier(ctx context.Context, issuerURL string) (*Verifier, error) {
	jwksURL := strings.TrimRight(issuerURL, "/") + "/protocol/openid-connect/certs"
	k, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("building JWKS keyfunc for %s: %w", jwksURL, err)
	}
	return &Verifier{jwks: k}, nil
}

// NewVerifierWithKeyfunc is the same construction path as NewVerifier, but
// takes an already-built Keyfunc — used by integration/auth_test.go to
// verify against a locally minted keypair instead of a live Keycloak.
func NewVerifierWithKeyfunc(k keyfunc.Keyfunc) *Verifier {
	return &Verifier{jwks: k}
}

const claimsKey = "auth_claims"

// Middleware parses Authorization: Bearer <token>, verifies its signature
// and expiry against the Verifier's JWKS, and sets Claims in the Gin
// context for FromContext to read. A missing or invalid token is a 401 —
// this middleware is only ever mounted when Enabled() is true, so there is
// no "auth off" branch here; that lives in FromContext instead.
func (v *Verifier) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenString, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || tokenString == "" {
			httpx.WriteProblem(c, httpx.ErrUnauthorized("missing_token", "Authorization: Bearer <token> is required"))
			c.Abort()
			return
		}

		var claims keycloakClaims
		token, err := jwt.ParseWithClaims(tokenString, &claims, v.jwks.Keyfunc)
		if err != nil || !token.Valid {
			httpx.WriteProblem(c, httpx.ErrUnauthorized("invalid_token", "token is invalid or expired"))
			c.Abort()
			return
		}

		c.Set(claimsKey, Claims{
			Subject: claims.Subject,
			Email:   claims.Email,
			Roles:   claims.RealmAccess.Roles,
		})
		c.Next()
	}
}

// FromContext returns the caller's Claims and true if Middleware verified a
// token on this request; the zero value and false otherwise — including
// every request when AUTH_ENABLE_REAL is off, since Middleware is never
// mounted in that case.
func FromContext(c *gin.Context) (Claims, bool) {
	v, ok := c.Get(claimsKey)
	if !ok {
		return Claims{}, false
	}
	claims, ok := v.(Claims)
	return claims, ok
}

// ActorFrom resolves the audit actor/actor-type pair for c: the caller's
// real email (or subject, if no email claim) and roles when a token was
// verified, or fallbackActor/fallbackType unchanged when it wasn't —
// preserving every existing call site's exact literal-string behavior when
// AUTH_ENABLE_REAL is off.
func ActorFrom(c *gin.Context, fallbackActor, fallbackType string) (actor, actorType string) {
	claims, ok := FromContext(c)
	if !ok {
		return fallbackActor, fallbackType
	}
	actor = claims.Email
	if actor == "" {
		actor = claims.Subject
	}
	actorType = fallbackType
	if len(claims.Roles) > 0 {
		actorType = strings.Join(claims.Roles, ",")
	}
	return actor, actorType
}
