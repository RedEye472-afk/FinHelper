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

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	"github.com/RedEye472-afk/FinHelper/internal/config"
	applog "github.com/RedEye472-afk/FinHelper/internal/log"
	"github.com/RedEye472-afk/FinHelper/internal/service/categorization"
	"github.com/RedEye472-afk/FinHelper/internal/service/operations"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
	transporthttp "github.com/RedEye472-afk/FinHelper/internal/transport/http"
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
		operationsSvc := operations.NewService(pool)
		// Categorizer shares the pool; attaching it to the operations service
		// turns on auto-categorization on create (BUSINESS_LOGIC ф.2).
		categorizationSvc := categorization.NewService(pool)
		operationsSvc.SetCategorizer(categorizationSvc)
		r.Mount("/", transporthttp.NewRouter(transporthttp.Deps{
			Pool:           pool,
			Issuer:         issuer,
			Salt:           cfg.UserHashSalt,
			Logger:         logger,
			Operations:     operationsSvc,
			Categorization: categorizationSvc,
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
