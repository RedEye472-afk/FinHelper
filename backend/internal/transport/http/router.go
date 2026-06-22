package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/user/finhelper/internal/auth"
	"github.com/user/finhelper/internal/storage"
)

// Deps bundles everything the v1 API router needs. main() assembles it once
// at boot and passes it to NewRouter. Keeping the bag explicit (vs many
// params) means adding a dependency doesn't break call sites.
type Deps struct {
	Pool   *storage.Pool
	Issuer *auth.JWTIssuer
	Salt   string
	Logger *slog.Logger
}

// NewRouter mounts the public and authenticated route groups under /api/v1.
//
// Layout:
//
//	/api/v1/auth/{register,login,refresh}   public
//	/api/v1/...                             behind AuthMiddleware (placeholder for Задача 3+)
//
// Returns nil if deps are incomplete — main treats that as a fatal boot error.
func NewRouter(deps Deps, mw *AuthMiddleware) http.Handler {
	if deps.Pool == nil || deps.Issuer == nil || deps.Salt == "" || deps.Logger == nil {
		panic("http: NewRouter requires all deps non-nil/non-empty")
	}
	if mw == nil {
		panic("http: NewRouter requires non-nil AuthMiddleware")
	}

	r := chi.NewRouter()

	authH := NewAuthHandler(AuthDeps{
		Pool:   deps.Pool,
		Issuer: deps.Issuer,
		Salt:   deps.Salt,
		Logger: deps.Logger,
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authH.Register)
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.Refresh)
		})

		// Everything else under /api/v1 is authenticated. Concrete feature
		// handlers (operations, dashboard, …) get mounted here in Задача 3+.
		r.Group(func(r chi.Router) {
			r.Use(mw.Wrap)
			// Example placeholder so the group is non-empty and verified by
			// integration tests. Returns the authenticated user_id.
			r.Get("/me", func(w http.ResponseWriter, req *http.Request) {
				uid, ok := MustUserID(req.Context())
				if !ok {
					writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"user_id":   uid,
					"user_hash": UserHashFrom(req.Context()),
				})
			})
		})
	})

	return r
}
