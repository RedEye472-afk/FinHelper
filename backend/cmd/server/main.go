// Command finhelper-server is the HTTP entry point of FinHelper.
//
// Wires: config → logger → storage → JWT issuer → HTTP router with auth
// middleware. Math endpoints and feature services are added in subsequent
// tasks (see PROGRESS.md).
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/RedEye472-afk/FinHelper/pkg/auth"
	"github.com/RedEye472-afk/FinHelper/pkg/config"
	"github.com/RedEye472-afk/FinHelper/pkg/email"
	applog "github.com/RedEye472-afk/FinHelper/pkg/log"
	"github.com/RedEye472-afk/FinHelper/pkg/ratelimit"
	"github.com/RedEye472-afk/FinHelper/pkg/service/budget"
	"github.com/RedEye472-afk/FinHelper/pkg/service/credit"
	"github.com/RedEye472-afk/FinHelper/pkg/service/categorization"
	"github.com/RedEye472-afk/FinHelper/pkg/service/dashboard"
	"github.com/RedEye472-afk/FinHelper/pkg/service/goals"
	"github.com/RedEye472-afk/FinHelper/pkg/service/operations"
	"github.com/RedEye472-afk/FinHelper/pkg/storage"
	transporthttp "github.com/RedEye472-afk/FinHelper/pkg/transport/http"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "finhelper: fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// ----- Config -----
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// ----- Logger -----
	logger := applog.New(cfg.Log.Level, cfg.Log.Format)

	// ----- Storage (optional at this stage: if DB is down, we still boot
	// /healthz so the binary is testable without a running Postgres) -----
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pool *storage.Pool
	if cfg.Database.URL != "" {
		pool, err = storage.Open(ctx, cfg.Database.URL)
		if err != nil {
			applog.Warn(ctx, logger, "database unavailable, starting without it",
				"error", err.Error())
		} else {
			applog.Info(ctx, logger, "database connected")
			defer pool.Close()
		}
	}

	// ----- HTTP server -----
	r := chi.NewRouter()

	// CORS middleware — allows the frontend (default: http://localhost:5173) to
	// call the API from a different origin.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.HTTP.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			http.Error(w, `{"status":"not_ready","reason":"no_database"}`, http.StatusServiceUnavailable)
			return
		}
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.DB.PingContext(pingCtx); err != nil {
			http.Error(w, `{"status":"not_ready","reason":"db_ping_failed"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// ----- JWT issuer + v1 API router (auth + operations endpoints) -----
	// Auth requires a DB pool. If we booted without one (smoke/CI), skip
	// mounting /api/v1 rather than crashing — /healthz still works.
	if pool != nil {
		issuer, err := auth.NewJWTIssuer(
			cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret,
			cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL,
		)
		if err != nil {
			return fmt.Errorf("jwt issuer: %w", err)
		}
		authMW := transporthttp.NewAuthMiddleware(issuer, logger)

		// Email sender (optional — if no API keys configured, Mailer stays nil
		// and email features are gracefully skipped).
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
			applog.Info(ctx, logger, "email sender configured")
		} else {
			applog.Warn(ctx, logger, "no email api keys: verification/password reset disabled")
		}

		// Rate limiter (10 req/min per IP for auth endpoints).
		rl := ratelimit.New(logger)
		_ = rl // used by router

		operationsSvc := operations.NewService(pool)
		// Categorizer shares the pool; attaching it to the operations service
		// turns on auto-categorization on create (BUSINESS_LOGIC ф.2).
		categorizationSvc := categorization.NewService(pool)
		operationsSvc.SetCategorizer(categorizationSvc)
		// Dashboard service (BUSINESS_LOGIC ф.3) — pure orchestration over the
		// pool's aggregate queries.
		dashboardSvc := dashboard.NewService(pool)
		// Budget service (BUSINESS_LOGIC ф.4) — per-category limits + rollover.
		budgetSvc := budget.NewService(pool)
		// Goals service (BUSINESS_LOGIC ф.5) — savings-goal tracker with
		// contributions journal, projection, and what-if simulation.
		goalsSvc := goals.NewService(pool)
		// Credit calculator service (BUSINESS_LOGIC ф.7) — stateless loan calc.
		creditSvc := credit.NewService()
		r.Mount("/", transporthttp.NewRouter(transporthttp.Deps{
			Pool:           pool,
			Issuer:         issuer,
			Salt:           cfg.UserHashSalt,
			Logger:         logger,
			Mailer:         mailer,
			RateLimiter:    rl,
			FrontendURL:    cfg.Email.FrontendURL,
			Operations:     operationsSvc,
			Categorization: categorizationSvc,
			Dashboard:      dashboardSvc,
			Budget:         budgetSvc,
			Goals:          goalsSvc,
			Credit:         creditSvc,
		}, authMW))
		applog.Info(ctx, logger, "api mounted",
			"access_ttl", cfg.JWT.AccessTTL.String(),
			"refresh_ttl", cfg.JWT.RefreshTTL.String())
	} else {
		applog.Warn(ctx, logger, "no database: /api/v1 endpoints not mounted")
	}

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// ----- Graceful shutdown -----
	go func() {
		applog.Info(ctx, logger, "http server listening", "addr", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			applog.Error(ctx, logger, "http server error", "error", err.Error())
			cancel()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-sig:
		applog.Info(ctx, logger, "shutdown signal received", "signal", s.String())
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	applog.Info(ctx, logger, "shutdown complete")
	return nil
}
