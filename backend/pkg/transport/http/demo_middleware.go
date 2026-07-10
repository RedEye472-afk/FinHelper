package http

import (
	"context"
	"net/http"
	"log/slog"

	applog "github.com/RedEye472-afk/FinHelper/backend/pkg/log"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/auth"
)

// DemoAuthMiddleware is an MVP middleware that bypasses JWT verification
// and always injects the demo user (user_id = 10) into the context.
// This allows the frontend to work without auth flow for MVP.
type DemoAuthMiddleware struct {
	verifier   auth.JWTVerifier
	logger     *slog.Logger
	demoUserID int64
}

// NewDemoAuthMiddleware creates a demo auth middleware that injects
// the given demo user ID into every request context.
func NewDemoAuthMiddleware(demoUserID int64, verifier auth.JWTVerifier, logger *slog.Logger) *DemoAuthMiddleware {
	if verifier == nil || logger == nil {
		panic("http: NewDemoAuthMiddleware requires non-nil deps")
	}
	return &DemoAuthMiddleware{
		verifier:   verifier,
		logger:     logger,
		demoUserID: demoUserID,
	}
}

// Wrap returns the middleware that injects demo user into context.
func (m *DemoAuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First try real JWT verification (for real auth flow)
		token := extractBearer(r)
		if token != "" {
			claims, err := m.verifier.Verify(token)
			if err == nil {
				// Real token valid - use it
				ctx := r.Context()
				ctx = context.WithValue(ctx, keyUserID, claims.UserID)
				ctx = context.WithValue(ctx, keyUserHash, claims.UserHash)
				ctx = applog.WithUserHash(ctx, claims.UserHash)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// No valid token - use demo user for MVP
		ctx := r.Context()
		ctx = context.WithValue(ctx, keyUserID, m.demoUserID)
		ctx = context.WithValue(ctx, keyUserHash, "demo-user-hash")
		ctx = applog.WithUserHash(ctx, "demo-user-hash")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}