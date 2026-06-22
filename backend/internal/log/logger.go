// Package log provides structured logging with PII-safe defaults.
//
// Principle (PRIVACY_RULES.md §1): logs MUST use user_hash, never email or
// other PII. This package's API nudges callers in that direction by offering
// a Logger that accepts a context with user identity already hashed.
package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// ctxKey is unexported so only this package can read context-stored fields.
type ctxKey int

const (
	keyUserHash ctxKey = iota
	keyCalcID
	keyRequestID
)

// New builds a slog.Logger at the requested level/format.
//   - format "json" → JSON output for production
//   - format "console" (default) → human-friendly text for development
func New(level, format string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if strings.ToLower(format) == "json" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}

// WithUserHash returns a context carrying the user_hash. Use it at the
// middleware boundary where the JWT is verified, so every downstream log
// line automatically includes user_hash instead of leaking PII.
func WithUserHash(ctx context.Context, userHash string) context.Context {
	return context.WithValue(ctx, keyUserHash, userHash)
}

// WithCalcID attaches a calc_id to the context for tracing a single
// calculation through the system (AI_GUARDRAILS.md §"Логирование").
func WithCalcID(ctx context.Context, calcID string) context.Context {
	return context.WithValue(ctx, keyCalcID, calcID)
}

// WithRequestID attaches a request id for HTTP tracing.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, keyRequestID, requestID)
}

// UserHashFrom extracts the user_hash from the context, or "" if absent.
func UserHashFrom(ctx context.Context) string {
	if v, ok := ctx.Value(keyUserHash).(string); ok {
		return v
	}
	return ""
}

// argsFrom returns the context-derived identity fields as slog key/value
// pairs (...any), matching the variadic signature of slog.Logger.Info etc.
// Order is stable so log lines read predictably.
func argsFrom(ctx context.Context) []any {
	var args []any
	if h := UserHashFrom(ctx); h != "" {
		args = append(args, "user_hash", h)
	}
	if v, ok := ctx.Value(keyCalcID).(string); ok && v != "" {
		args = append(args, "calc_id", v)
	}
	if v, ok := ctx.Value(keyRequestID).(string); ok && v != "" {
		args = append(args, "request_id", v)
	}
	return args
}

// Info logs at Info level, attaching identity fields from ctx.
func Info(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	logger.Info(msg, append(argsFrom(ctx), args...)...)
}

// Error logs at Error level, attaching identity fields from ctx.
func Error(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	logger.Error(msg, append(argsFrom(ctx), args...)...)
}

// Warn logs at Warn level, attaching identity fields from ctx.
func Warn(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	logger.Warn(msg, append(argsFrom(ctx), args...)...)
}

// Debug logs at Debug level, attaching identity fields from ctx.
func Debug(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	logger.Debug(msg, append(argsFrom(ctx), args...)...)
}
