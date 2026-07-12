// Package storage manages database connectivity for FinHelper.
//
// We use database/sql with pgx as the driver. The *sql.DB returned by Open
// is safe for concurrent use and is intended to be a long-lived singleton
// owned by main. database/sql owns connection pooling; we only tune its
// parameters here.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// Pool is a thin wrapper around *sql.DB so callers don't import pgx directly.
// It also gives us a single place to attach observability later (tracing,
// slow-query logging, etc.).
type Pool struct {
	DB *sql.DB
}

// Open creates a connection pool to the given Postgres URL and pings
// immediately to verify connectivity.
//   - max 25 open connections, 25 idle
//   - conn max lifetime 30 min, idle 5 min
//
// The caller MUST call Close when done.
func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	pool, err := OpenLazy(databaseURL)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.DB.PingContext(pingCtx); err != nil {
		_ = pool.DB.Close()
		return nil, fmt.Errorf("storage: ping database: %w", err)
	}
	return pool, nil
}

// OpenLazy creates a connection pool WITHOUT connecting. The first
// query triggers the actual TCP + SCRAM handshake. Use this in
// serverless environments where init time is limited (e.g. Vercel
// Hobby 10s timeout). Callers should handle connection errors at
// query time instead.
func OpenLazy(databaseURL string) (*Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("storage: DATABASE_URL is empty")
	}

	// Add connection tuning for serverless (Vercel λ):
	// - connect_timeout: fail fast instead of hanging
	// - prefer_simple_protocol: skip SCRAM-SHA-256 handshake (faster cold start)
	// - sslmode=require: faster than verify-full (no cert chain validation)
	connStr := databaseURL
	if !strings.Contains(connStr, "connect_timeout") {
		connStr += "&connect_timeout=5"
	}
	if !strings.Contains(connStr, "prefer_simple_protocol") {
		connStr += "&prefer_simple_protocol=true"
	}
	if !strings.Contains(connStr, "sslmode=") {
		connStr += "&sslmode=require"
	}

	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("storage: parse database url: %w", err)
	}

	// Minimal pool for serverless: no idle connections
	cfg.MaxConns = 5
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 5 * time.Minute
	cfg.MaxConnIdleTime = 30 * time.Second

	db := stdlib.OpenDB(*cfg.ConnConfig)
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(30 * time.Second)

	return &Pool{DB: db}, nil
}

// Close releases all database resources.
func (p *Pool) Close() error {
	if p == nil || p.DB == nil {
		return nil
	}
	return p.DB.Close()
}
