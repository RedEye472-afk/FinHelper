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

// Open creates a connection pool to the given Postgres URL.
//   - max 25 open connections, 25 idle
//   - conn max lifetime 30 min, idle 5 min
//
// The caller MUST call Close when done.
func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("storage: DATABASE_URL is empty")
	}

	// ParseConfig accepts the same URL forms as pgxpool and gives us a
	// *pgx.ConnConfig that stdlib can wrap into a *sql.DB.
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("storage: parse database url: %w", err)
	}
	db := stdlib.OpenDB(*cfg.ConnConfig)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("storage: ping database: %w", err)
	}
	return &Pool{DB: db}, nil
}

// Close releases all database resources.
func (p *Pool) Close() error {
	if p == nil || p.DB == nil {
		return nil
	}
	return p.DB.Close()
}
