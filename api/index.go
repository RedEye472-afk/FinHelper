package handler

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/auth"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/config"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/email"
	applog "github.com/RedEye472-afk/FinHelper/backend/pkg/log"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/ratelimit"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/budget"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/categorization"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/credit"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/dashboard"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/deposit"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/goals"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/migrate"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/operations"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
	transporthttp "github.com/RedEye472-afk/FinHelper/backend/pkg/transport/http"
)

var (
	h    http.Handler
	once sync.Once
)

func getHandler() http.Handler {
	once.Do(func() {
		h = initHandler()
	})
	return h
}

// initHandler builds the full API router. DB connection is LAZY:
// database/sql doesn't connect until the first query, so the λ
// starts in <1s. Migrations are deferred to POST /api/v1/migrate.
//
// This avoids the Vercel Hobby 10s timeout: no blocking ping,
// no synchronous migration at init.
func initHandler() http.Handler {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("vercel: config load error, degraded mode: %v", err)
		return buildDegradedRouter("configuration error: " + err.Error())
	}

	logger := applog.New(cfg.Log.Level, cfg.Log.Format)

	// OpenLazy creates a *sql.DB without connecting. First query
	// triggers the actual TCP+SCRAM handshake.
	pool, dbErr := storage.OpenLazy(cfg.Database.URL)
	if dbErr != nil {
		log.Printf("vercel: pool creation error, degraded mode: %v", dbErr)
		return buildDegradedRouter("database config error: " + dbErr.Error())
	}
	log.Println("vercel: lazy pool created (no connection yet)")

	issuer, err := auth.NewJWTIssuer(
		cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL,
	)
	if err != nil {
		log.Printf("vercel: jwt issuer error, degraded mode: %v", err)
		return buildDegradedRouter("jwt issuer error: " + err.Error())
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

	opsSvc := operations.NewService(pool)
	catSvc := categorization.NewService(pool)
	opsSvc.SetCategorizer(catSvc)
	dashSvc := dashboard.NewService(pool)
	budSvc := budget.NewService(pool)
	goalsSvc := goals.NewService(pool)
	credSvc := credit.NewService()
	depSvc := deposit.NewService()

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.HTTP.CORSAllowedOrigins,
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
		Deposit:        depSvc,
		Credit:         credSvc,
	}, authMW)

	// Healthz: always OK (λ is alive, DB connection is lazy)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Readyz: probe DB with a short-timeout query
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		probeCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := pool.DB.PingContext(probeCtx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","db":"unreachable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ready","db":"connected"}`))
	})

	// Migrations: trigger manually after deploy
	r.Post("/api/v1/migrate", func(w http.ResponseWriter, r *http.Request) {
		migrateCtx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		migrate.Run(migrateCtx, pool)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","msg":"migrations applied"}`))
	})
	r.Get("/migrate", func(w http.ResponseWriter, r *http.Request) {
		migrateCtx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		migrate.Run(migrateCtx, pool)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","msg":"migrations applied"}`))
	})

	r.Mount("/", apiRouter)
	log.Println("vercel: handler ready")
	return r
}

func buildDegradedRouter(reason string) http.Handler {
	msg := `{"status":"degraded","note":"` + reason + `"}`
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(msg))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(msg))
	})
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(msg))
	})
	return r
}

// Handler is the Vercel Serverless Function entry point.
func Handler(w http.ResponseWriter, r *http.Request) {
	getHandler().ServeHTTP(w, r)
}
