// Package ratelimit provides a simple in-memory rate limiter (10 req/min per IP).
// Suitable for auth endpoints to prevent brute-force attacks.
//
// Source: adapted from C:\Users\user\Documents\finhelper\api\index.js
package ratelimit

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	window = 1 * time.Minute
	maxReq = 10
)

type entry struct {
	window time.Time
	count  int
}

// Limiter is an in-memory per-IP rate limiter.
type Limiter struct {
	mu     sync.Mutex
	store  map[string]*entry
	logger *slog.Logger
}

// New creates a Limiter. The cleanup goroutine runs every 5 minutes to purge
// stale entries.
func New(logger *slog.Logger) *Limiter {
	l := &Limiter{
		store:  make(map[string]*entry),
		logger: logger,
	}
	go l.cleanup()
	return l
}

// Middleware returns an HTTP middleware that rejects requests exceeding the
// rate limit with 429 Too Many Requests.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.allow(ip) {
			l.logger.Warn("rate limit exceeded", "ip", ip)
			http.Error(w, `{"error":"Too many requests. Please try again later."}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	e, exists := l.store[ip]
	if !exists || now.Sub(e.window) > window {
		l.store[ip] = &entry{window: now, count: 1}
		return true
	}
	if e.count >= maxReq {
		return false
	}
	e.count++
	return true
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for ip, e := range l.store {
			if now.Sub(e.window) > window {
				delete(l.store, ip)
			}
		}
		l.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	// Vercel / proxy: x-forwarded-for
	if fwd := r.Header.Get("x-forwarded-for"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	// Direct connection
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > 0 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
