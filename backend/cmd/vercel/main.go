// Command vercel is the Vercel Serverless Function entry point for FinHelper.
//
// It builds a full chi router and wraps it with the Vercel Go Bridge
// for Lambda compatibility.
//
// Build: go build -o ../../api/bootstrap ./cmd/vercel/
package main

import (
	"context"
	"log"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/chi/v5"
	"github.com/vercel/go-bridge/go/bridge"

	"github.com/RedEye472-afk/FinHelper/pkg/auth"
	"github.com/RedEye472-afk/FinHelper/pkg/config"
	"github.com/RedEye472-afk/FinHelper/pkg/email"
	applog "github.com/RedEye472-afk/FinHelper/pkg/log"
	"github.com/RedEye472-afk/FinHelper/pkg/ratelimit"
	"github.com/RedEye472-afk/FinHelper/pkg/service/budget"
	"github.com/RedEye472-afk/FinHelper/pkg/service/categorization"
	"github.com/RedEye472-afk/FinHelper/pkg/service/credit"
	"github.com/RedEye472-afk/FinHelper/pkg/service/dashboard"
	"github.com/RedEye472-afk/FinHelper/pkg/service/goals"
	"github.com/RedEye472-afk/FinHelper/pkg/service/operations"
	"github.com/RedEye472-afk/FinHelper/pkg/storage"
	transporthttp "github.com/RedEye472-afk/FinHelper/pkg/transport/http"
)

var h http.Handler

func init() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("vercel: config load error, degraded: %v", err)
		h = degradedHandler("config error: " + err.Error())
		return
	}

	logger := applog.New(cfg.Log.Level, cfg.Log.Format)
	ctx := context.Background()

	pool, dbErr := storage.Open(ctx, cfg.Database.URL)
	if dbErr != nil {
		log.Printf("vercel: db unavailable (degraded): %v", dbErr)
		h = degradedHandler("db unavailable")
		return
	}

	issuer, err := auth.NewJWTIssuer(
		cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL,
	)
	if err != nil {
		log.Printf("vercel: jwt issuer error, degraded: %v", err)
		h = degradedHandler("jwt issuer error")
		return
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
		Credit:         credSvc,
	}, authMW)

	r.Mount("/", apiRouter)
	h = r
}

func Handler(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTP(w, r)
}

func degradedHandler(reason string) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}))

	msg := `{"status":"degraded","note":"` + reason + `"}`
	r.Get("/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(msg))
	})
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(msg))
	})
	return r
}

func main() {
	bridge.Start(http.HandlerFunc(Handler))
}
