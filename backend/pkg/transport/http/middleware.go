package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/RedEye472-afk/FinHelper/pkg/auth"
	applog "github.com/RedEye472-afk/FinHelper/pkg/log"
)

type ctxKey int

const (
	keyUserID ctxKey = iota
	keyUserHash
)

// userIDFrom exposes the identity AuthMiddleware injected. Downstream
// handlers read it via MustUserID instead of importing the auth package.
func userIDFrom(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(keyUserID).(int64)
	return v, ok
}

// UserHashFrom returns the user_hash attached by AuthMiddleware, or "".
func UserHashFrom(ctx context.Context) string {
	if v, ok := ctx.Value(keyUserHash).(string); ok {
		return v
	}
	return ""
}

// MustUserID is the convenience handlers reach for after AuthMiddleware.
func MustUserID(ctx context.Context) (int64, bool) { return userIDFrom(ctx) }

// MustUserHash returns the user_hash from context, or empty string if missing.
func MustUserHash(ctx context.Context) string { return UserHashFrom(ctx) }

// AuthMiddleware verifies the Bearer access token and stashes user_id +
// user_hash in the request context. On failure it responds 401 and stops
// the chain.
//
// It also attaches user_hash to the log context so every log line
// downstream carries the anonymous id, not PII.
type AuthMiddleware struct {
	verifier auth.JWTVerifier
	logger   *slog.Logger
}

// NewAuthMiddleware constructs the middleware. verifier is the same issuer
// used by AuthHandler (access-secret side); accepting an interface lets
// tests substitute a fake.
func NewAuthMiddleware(verifier auth.JWTVerifier, logger *slog.Logger) *AuthMiddleware {
	if verifier == nil || logger == nil {
		panic("http: NewAuthMiddleware requires non-nil deps")
	}
	return &AuthMiddleware{verifier: verifier, logger: logger}
}

// Wrap returns the http.Handler-compatible middleware function.
func (m *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := m.verifier.Verify(extractBearer(r))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "auth.unauthorized", "missing or invalid token")
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, keyUserID, claims.UserID)
		ctx = context.WithValue(ctx, keyUserHash, claims.UserHash)
		ctx = applog.WithUserHash(ctx, claims.UserHash)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractBearer pulls the access token out of "Authorization: Bearer <jwt>".
// Empty string on missing/malformed header; the verifier turns that into a
// 401 uniformly.
func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
