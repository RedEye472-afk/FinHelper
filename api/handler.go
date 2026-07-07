package handler

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	"github.com/RedEye472-afk/FinHelper/internal/config"
	"github.com/RedEye472-afk/FinHelper/internal/email"
	applog "github.com/RedEye472-afk/FinHelper/internal/log"
	"github.com/RedEye472-afk/FinHelper/internal/ratelimit"
	"github.com/RedEye472-afk/FinHelper/internal/service/budget"
	"github.com/RedEye472-afk/FinHelper/internal/service/categorization"
	"github.com/RedEye472-afk/FinHelper/internal/service/credit"
	"github.com/RedEye472-afk/FinHelper/internal/service/dashboard"
	"github.com/RedEye472-afk/FinHelper/internal/service/goals"
	"github.com/RedEye472-afk/FinHelper/internal/service/operations"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
	transporthttp "github.com/RedEye472-afk/FinHelper/internal/transport/http"
)

var h http.Handler

func init() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("vercel: config load: %v", err)
	}

	logger := applog.New(cfg.Log.Level, cfg.Log.Format)
	ctx := context.Background()

	// Build a default error handler for when DB is unavailable.
	pool, dbErr := storage.Open(ctx, cfg.Database.URL)
	if dbErr != nil {
		log.Printf("vercel: db unavailable (serving 503): %v", dbErr)
	}

	issuer, err := auth.NewJWTIssuer(
		cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL,
	)
	if err != nil {
		log.Fatalf("vercel: jwt issuer: %v", err)
	}

	authMW := transporthttp.NewAuthMiddleware(issuer, logger)

	var mailer *email.Sender
	if cfg.Email.ResendAPIKey != "" || cfg.Email.SendGridAPIKey != "" || cfg.Email.BrevoAPIKey != "" {
		mailer = email.NewSender(
			logger,
			cfg.Email.FromEmail,
			cfg.Email.FromName,
			cfg.Email.ResendAPIKey,
			cfg.Email.SendGridAPIKey,
			cfg.Email.BrevoAPIKey,
			cfg.Email.BrevoSender,
		)
	}

	rl := ratelimit.New(logger)

	if pool != nil {
		// Full app with database.
		opsSvc := operations.NewService(pool)
		catSvc := categorization.NewService(pool)
		opsSvc.SetCategorizer(catSvc)
		dashSvc := dashboard.NewService(pool)
		budSvc := budget.NewService(pool)
		goalsSvc := goals.NewService(pool)
		credSvc := credit.NewService()

		h = buildAppRouter(cfg, logger, pool, issuer, authMW, mailer, rl,
			opsSvc, catSvc, dashSvc, budSvc, goalsSvc, credSvc)
	} else {
		// Degraded mode — all routes return 503.
		h = buildDegradedRouter()
	}

	log.Println("vercel: handler ready")
}

// buildAppRouter constructs the full chi router with all services.
func buildAppRouter(
	cfg config.Config,
	logger *slog.Logger,
	pool *storage.Pool,
	issuer *auth.JWTIssuer,
	authMW *transporthttp.AuthMiddleware,
	mailer *email.Sender,
	rl *ratelimit.Limiter,
	opsSvc *operations.Service,
	catSvc *categorization.Service,
	dashSvc *dashboard.Service,
	budSvc *budget.Service,
	goalsSvc *goals.Service,
	credSvc *credit.Service,
) http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.Email.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Idempotency-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	apiRouter := transporthttp.NewRouter(transporthttp.Deps{
		Pool:           pool,
		Issuer:         issuer,
		Salt:           cfg.UserHashSalt,
		Logger:         logger,
		Mailer:         mailer,
		RateLimiter:    rl,
		FrontendURL:    cfg.Email.FrontendURL,
		Operations:     opsSvc,
		Categorization: catSvc,
		Dashboard:      dashSvc,
		Budget:         budSvc,
		Goals:          goalsSvc,
		Credit:         credSvc,
	}, authMW)

	r.Mount("/", apiRouter)
	return r
}

// buildDegradedRouter returns a router that responds 503 to everything.
func buildDegradedRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"degraded","note":"database unavailable"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unavailable","note":"database unavailable"}`))
	})
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"db_unavailable","message":"Service unavailable — database not configured"}`))
	})
	return r
}

// Handler is the Vercel Serverless Function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTP(w, r)
}
