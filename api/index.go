package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
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

// readyHandler holds the fully-initialised router once the background
// init goroutine completes. nil = still warming up.
var readyHandler atomic.Pointer[http.Handler]

// initOnce guarantees the background goroutine launches exactly once.
var initOnce sync.Once

// startInit launches a background goroutine that connects to the DB,
// applies migrations, and builds the full API router. The λ answers
// immediately (see Handler below); callers get 503 "initializing" until
// readyHandler is set.
//
// On failure the goroutine sets a degraded handler instead, so the λ
// always has a valid response path.
func startInit() {
	initOnce.Do(func() {
		go func() {
			h := initHandlerWithRetry()
			readyHandler.Store(&h)
			log.Println("vercel: background init complete")
		}()
	})
}

// initHandlerWithRetry builds the full router, retrying DB connection
// a few times before giving up and returning a degraded router.
func initHandlerWithRetry() http.Handler {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("vercel: config load error, degraded mode: %v", err)
		return buildDegradedRouter("configuration error: " + err.Error())
	}

	logger := applog.New(cfg.Log.Level, cfg.Log.Format)

	// Retry DB connection: Neon cold-start can take a few seconds.
	// Vercel λ stays warm between requests, so by the 2nd-3rd attempt
	// the Neon pooler should respond.
	var pool *storage.Pool
	var dbErr error
	for attempt := 1; attempt <= 3; attempt++ {
		pool, dbErr = storage.Open(context.Background(), cfg.Database.URL)
		if dbErr == nil {
			break
		}
		log.Printf("vercel: db attempt %d/3 failed: %v", attempt, dbErr)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // 2s, 4s
		}
	}
	if dbErr != nil {
		log.Printf("vercel: db unavailable after 3 attempts (degraded): %v", dbErr)
		return buildDegradedRouter("database unavailable")
	}
	log.Println("vercel: db connected")

	// Apply schema migrations synchronously inside the goroutine.
	migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer migrateCancel()
	migrate.Run(migrateCtx, pool)
	log.Println("vercel: migrations completed")

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

	// Diagnostics: trigger migrations manually
	r.Get("/migrate", func(w http.ResponseWriter, r *http.Request) {
		migrateCtx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		migrate.Run(migrateCtx, pool)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","msg":"migrations applied"}`))
	})

	// Diagnostics: trigger migrations manually
	r.Post("/api/v1/migrate", func(w http.ResponseWriter, r *http.Request) {
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

// buildWarmingRouter returns a router that answers 503 "initializing"
// for everything except /healthz (always 200). This lets the λ respond
// instantly on cold start while the DB connects in the background.
func buildWarmingRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}))

	body, _ := json.Marshal(map[string]any{
		"status":      "initializing",
		"retry_after": 2,
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","note":"initializing"}`))
	})
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write(body)
	})
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
//
// Flow:
//  1. Launch background init goroutine (once).
//  2. If full router is ready → delegate to it.
//  3. If not → answer 503 "initializing" (warming router) so the λ
//     returns instantly without exceeding the Vercel Hobby 10s limit.
func Handler(w http.ResponseWriter, r *http.Request) {
	startInit()

	if hPtr := readyHandler.Load(); hPtr != nil {
		(*hPtr).ServeHTTP(w, r)
		return
	}
	buildWarmingRouter().ServeHTTP(w, r)
}
