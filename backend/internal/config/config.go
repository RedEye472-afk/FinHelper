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
	Database        DatabaseConfig
	HTTP            HTTPConfig
	JWT             JWTConfig
	UserHashSalt    string
	Log             LogConfig
}

type DatabaseConfig struct {
	// URL is the full Postgres connection string, e.g.
	// postgres://user:pass@host:5432/db?sslmode=disable
	URL string
}

type HTTPConfig struct {
	Addr               string
	CORSAllowedOrigins []string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

// Load reads configuration from environment variables.
// Returns an error if a required variable is missing or malformed.
func Load() (Config, error) {
	var cfg Config
	var problems []string
	var ok bool

	// ----- Database -----
	// DATABASE_URL is optional at config-load time so the binary can boot
	// for health checks / smoke tests without a running Postgres. main()
	// decides what to do when it's empty (typically: /readyz returns 503).
	cfg.Database.URL = os.Getenv("DATABASE_URL")

	// ----- HTTP -----
	cfg.HTTP.Addr = getenvDefault("HTTP_ADDR", ":8080")
	cfg.HTTP.CORSAllowedOrigins = splitCSV(getenvDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173"))

	// ----- JWT -----
	cfg.JWT.AccessSecret = os.Getenv("JWT_ACCESS_SECRET")
	if len(cfg.JWT.AccessSecret) < 32 {
		problems = append(problems, "JWT_ACCESS_SECRET must be at least 32 characters")
	}
	cfg.JWT.RefreshSecret = os.Getenv("JWT_REFRESH_SECRET")
	if len(cfg.JWT.RefreshSecret) < 32 {
		problems = append(problems, "JWT_REFRESH_SECRET must be at least 32 characters")
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
