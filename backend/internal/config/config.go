// Package config loads application configuration from environment.
//
// All secrets come from environment variables (.env in dev, real env in prod).
// No secrets are ever read from files committed to the repo.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config is the parsed application configuration.
type Config struct {
	Database     DatabaseConfig
	HTTP         HTTPConfig
	JWT          JWTConfig
	UserHashSalt string
	Log          LogConfig
	Email        EmailConfig
}

// DatabaseConfig holds the Postgres connection string.
type DatabaseConfig struct {
	URL string
}

// HTTPConfig holds server address and CORS settings.
type HTTPConfig struct {
	Addr               string
	CORSAllowedOrigins []string
}

// JWTConfig holds secrets and TTL for access/refresh tokens.
type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

// LogConfig controls logging verbosity and output format.
type LogConfig struct {
	Level  string
	Format string
}

// EmailConfig holds multi-provider email API keys.
type EmailConfig struct {
	FromEmail      string
	FromName       string
	ResendAPIKey   string
	SendGridAPIKey string
	BrevoAPIKey    string
	BrevoSender    string
	FrontendURL    string
}

// Load reads configuration from environment variables.
// Returns an error if a required variable is missing or malformed.
func Load() (Config, error) {
	var cfg Config
	var problems []string
	var ok bool
	var key string

	// ----- Database -----
	// DATABASE_URL is optional at config-load time so the binary can boot
	// for health checks / smoke tests without a running Postgres. main()
	// decides what to do when it's empty (typically: /readyz returns 503).
	cfg.Database.URL = os.Getenv("DATABASE_URL")

	// ----- HTTP -----
	cfg.HTTP.Addr = getenvDefault("HTTP_ADDR", ":8080")
	cfg.HTTP.CORSAllowedOrigins = splitCSV(getenvDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173"))

	// ----- JWT -----
	key = "JWT_ACCESS_SECRET"
	cfg.JWT.AccessSecret = os.Getenv(key)
	if len(cfg.JWT.AccessSecret) < 32 {
		problems = append(problems, key+" must be at least 32 characters")
	}
	key = "JWT_REFRESH_SECRET"
	cfg.JWT.RefreshSecret = os.Getenv(key)
	if len(cfg.JWT.RefreshSecret) < 32 {
		problems = append(problems, key+" must be at least 32 characters")
	}
	cfg.JWT.AccessTTL, ok = getenvDuration("JWT_ACCESS_TTL", 15*time.Minute)
	if !ok {
		problems = append(problems, "JWT_ACCESS_TTL is not a valid duration (e.g. 15m, 720h)")
	}
	cfg.JWT.RefreshTTL, ok = getenvDuration("JWT_REFRESH_TTL", 30*24*time.Hour)
	if !ok {
		problems = append(problems, "JWT_REFRESH_TTL is not a valid duration (e.g. 15m, 720h)")
	}

	// ----- User hashing -----
	cfg.UserHashSalt = os.Getenv("USER_HASH_SALT")
	if cfg.UserHashSalt == "" {
		problems = append(problems, "USER_HASH_SALT is required")
	}

	// ----- Logging -----
	cfg.Log.Level = strings.ToLower(getenvDefault("LOG_LEVEL", "info"))
	cfg.Log.Format = strings.ToLower(getenvDefault("LOG_FORMAT", "console"))

	// ----- Email -----
	cfg.Email.FromEmail = getenvDefault("FROM_EMAIL", "onboarding@resend.dev")
	cfg.Email.FromName = getenvDefault("FROM_NAME", "FinHelper")
	cfg.Email.ResendAPIKey = os.Getenv("RESEND_API_KEY")
	cfg.Email.SendGridAPIKey = os.Getenv("SENDGRID_API_KEY")
	cfg.Email.BrevoAPIKey = os.Getenv("BREVO_API_KEY")
	cfg.Email.BrevoSender = os.Getenv("BREVO_SENDER_EMAIL")
	cfg.Email.FrontendURL = getenvDefault("FRONTEND_URL", "")

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return cfg, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvDuration(key string, def time.Duration) (time.Duration, bool) {
	v := os.Getenv(key)
	if v == "" {
		return def, true
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def, false
	}
	return d, true
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
