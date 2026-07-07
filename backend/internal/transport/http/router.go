package http

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	"github.com/RedEye472-afk/FinHelper/internal/email"
	"github.com/RedEye472-afk/FinHelper/internal/ratelimit"
	"github.com/RedEye472-afk/FinHelper/internal/service/budget"
	"github.com/RedEye472-afk/FinHelper/internal/service/categorization"
	"github.com/RedEye472-afk/FinHelper/internal/service/credit"
	"github.com/RedEye472-afk/FinHelper/internal/service/dashboard"
	"github.com/RedEye472-afk/FinHelper/internal/service/goals"
	"github.com/RedEye472-afk/FinHelper/internal/service/operations"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// Deps bundles everything the v1 API router needs.
type Deps struct {
	Pool          *storage.Pool
	Issuer        *auth.JWTIssuer
	Salt          string
	Logger        *slog.Logger
	Mailer        *email.Sender // email sender (nil = skip email features)
	RateLimiter   *ratelimit.Limiter
	FrontendURL   string // URL for password reset links

	Operations     *operations.Service
	Categorization *categorization.Service
	Dashboard      *dashboard.Service
	Budget         *budget.Service
	Goals          *goals.Service
	Credit         *credit.Service
}

// NewRouter mounts the public and authenticated route groups under /api/v1.
func NewRouter(deps Deps, mw *AuthMiddleware) http.Handler {
	if deps.Pool == nil || deps.Issuer == nil || deps.Salt == "" || deps.Logger == nil {
		panic("http: NewRouter requires all deps non-nil/non-empty")
	}
	if mw == nil {
		panic("http: NewRouter requires non-nil AuthMiddleware")
	}

	r := chi.NewRouter()

	authH := NewAuthHandler(AuthDeps{
		Pool:        deps.Pool,
		Issuer:      deps.Issuer,
		Salt:        deps.Salt,
		Logger:      deps.Logger,
		Mailer:      deps.Mailer,
		FrontendURL: deps.FrontendURL,
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes — rate limited.
		if deps.RateLimiter != nil {
			r.Group(func(r chi.Router) {
				r.Use(deps.RateLimiter.Middleware)
				r.Post("/auth/register", authH.Register)
				r.Post("/auth/login", authH.Login)
				r.Post("/auth/verify-email", authH.VerifyEmail)
				r.Post("/auth/send-code", authH.SendCode)
				r.Post("/auth/forgot-password", authH.ForgotPassword)
				r.Post("/auth/reset-password", authH.ResetPassword)
			})
		} else {
			r.Post("/auth/register", authH.Register)
			r.Post("/auth/login", authH.Login)
			r.Post("/auth/verify-email", authH.VerifyEmail)
			r.Post("/auth/send-code", authH.SendCode)
			r.Post("/auth/forgot-password", authH.ForgotPassword)
			r.Post("/auth/reset-password", authH.ResetPassword)
		}
		r.Post("/auth/refresh", authH.Refresh)

		// Everything else under /api/v1 is authenticated.
		r.Group(func(r chi.Router) {
			r.Use(mw.Wrap)

			if deps.Operations != nil {
				NewOperationsHandler(deps.Operations, deps.Logger).Register(r)
			}
			if deps.Categorization != nil {
				NewCategoriesHandler(deps.Pool, deps.Categorization, deps.Logger).Register(r)
			}
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
