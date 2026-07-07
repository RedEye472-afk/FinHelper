package http

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	"github.com/RedEye472-afk/FinHelper/internal/service/budget"
	"github.com/RedEye472-afk/FinHelper/internal/service/credit"
	"github.com/RedEye472-afk/FinHelper/internal/service/categorization"
	"github.com/RedEye472-afk/FinHelper/internal/service/dashboard"
	"github.com/RedEye472-afk/FinHelper/internal/service/goals"
	"github.com/RedEye472-afk/FinHelper/internal/service/operations"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// Deps bundles everything the v1 API router needs. main() assembles it once
// at boot and passes it to NewRouter. Keeping the bag explicit (vs many
// params) means adding a dependency doesn't break call sites.
type Deps struct {
	Pool   *storage.Pool
	Issuer *auth.JWTIssuer
	Salt   string
	Logger *slog.Logger

	// Operations is the business service for ф.1 (manual entry). nil = skip
	// mounting /operations routes (used by smoke/CI boots without a full
	// service graph; the pool itself is still required).
	Operations *operations.Service
	// Categorization is the auto-categorizer for ф.2. nil = skip mounting
	// /categories + /categorization routes. The operations service is wired
	// with it separately via SetCategorizer in main().
	Categorization *categorization.Service
	// Dashboard is the summary service for ф.3. nil = skip mounting /dashboard.
	Dashboard *dashboard.Service
	// Budget is the per-category limit service for ф.4. nil = skip mounting
	// /budgets.
	Budget *budget.Service
	// Goals is the savings-goal tracker service for ф.5. nil = skip mounting
	// /goals + /calc/goal.
	Goals *goals.Service
	// Credit is the loan calculator service for ф.7. nil = skip mounting
	// /calc/credit.
	Credit *credit.Service
}

// NewRouter mounts the public and authenticated route groups under /api/v1.
//
// Layout:
//
//	/api/v1/auth/{register,login,refresh}   public
//	/api/v1/operations/...                   behind AuthMiddleware (ф.1)
//	/api/v1/...                              other authenticated routes
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
		// handlers get mounted here as their services come online.
		r.Group(func(r chi.Router) {
			r.Use(mw.Wrap)

			if deps.Operations != nil {
				NewOperationsHandler(deps.Operations, deps.Logger).Register(r)
			}
			if deps.Categorization != nil {
				NewCategoriesHandler(deps.Pool, deps.Categorization, deps.Logger).Register(r)
			}
			// Accounts is always available (pool-only handler, no service dep).
			NewAccountsHandler(deps.Pool, deps.Logger).Register(r)
			if deps.Dashboard != nil {
				NewDashboardHandler(deps.Dashboard, deps.Logger).Register(r)
			}
			if deps.Budget != nil {
				NewBudgetHandler(deps.Budget, deps.Logger).Register(r)
			}
			if deps.Goals != nil {
				NewGoalsHandler(deps.Goals, deps.Logger).Register(r)
			}
			if deps.Credit != nil {
				NewCreditHandler(deps.Credit, deps.Logger).Register(r)
			}

			// GET /me returns the authenticated user's profile.
			r.Get("/me", func(w http.ResponseWriter, req *http.Request) {
				ctx := req.Context()
				uid, ok := MustUserID(ctx)
				if !ok {
					writeError(w, http.StatusUnauthorized, "auth.unauthorized", "no user in context")
					return
				}
				user, err := deps.Pool.GetUserByID(ctx, uid)
				if err != nil {
					if errors.Is(err, storage.ErrUserNotFound) {
						writeError(w, http.StatusNotFound, "me.not_found", "user not found")
						return
					}
					deps.Logger.Error("me: lookup", "error", err.Error())
					writeError(w, http.StatusInternalServerError, "internal", "")
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"id":         user.ID,
					"email":      user.Email,
					"created_at": user.CreatedAt.Format(time.RFC3339),
				})
			})
		})
	})

	return r
}
