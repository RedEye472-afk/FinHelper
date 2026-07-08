package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrations holds the SQL statements to apply, keyed by name.
var migrations = map[string]string{}

func init() {
	// 0001_init.sql - trimmed to essential DDL
	migrations["0001_init_schema"] = `
-- ENUMS
CREATE TYPE IF NOT EXISTS operation_type AS ENUM ('income','expense','transfer','currency_exchange','refund');
CREATE TYPE IF NOT EXISTS income_subtype AS ENUM ('salary','fee','gift','investment','loan_repayment');
CREATE TYPE IF NOT EXISTS account_type AS ENUM ('cash','bank','savings','investment','crypto','debt');

-- USERS
CREATE TABLE IF NOT EXISTS users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    user_hash TEXT NOT NULL UNIQUE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    verification_code TEXT,
    verification_expires TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- REFRESH TOKENS
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ACCOUNTS
CREATE TABLE IF NOT EXISTS accounts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    account_type account_type NOT NULL DEFAULT 'cash',
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    balance NUMERIC(28,2) NOT NULL DEFAULT 0 CHECK (balance = ROUND(balance,2)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- CATEGORIES
CREATE TABLE IF NOT EXISTS categories (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, name)
);

-- OPERATIONS
CREATE TABLE IF NOT EXISTS operations (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    calc_id TEXT NOT NULL,
    operation_type operation_type NOT NULL,
    amount NUMERIC(28,2) NOT NULL CHECK (amount > 0 AND amount = ROUND(amount,2)),
    amount_dst NUMERIC(28,2) CHECK (amount_dst IS NULL OR (amount_dst > 0 AND amount_dst = ROUND(amount_dst,2))),
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    account_dst_id BIGINT REFERENCES accounts(id) ON DELETE RESTRICT,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    income_subtype income_subtype,
    counterparty TEXT,
    description TEXT,
    operation_date DATE NOT NULL,
    is_planned BOOLEAN NOT NULL DEFAULT FALSE,
    category_confidence NUMERIC(4,3) CHECK (category_confidence IS NULL OR (category_confidence >= 0 AND category_confidence <= 1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, calc_id)
);

-- GOALS
CREATE TABLE IF NOT EXISTS goals (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    target_amount NUMERIC(28,2) NOT NULL CHECK (target_amount > 0),
    current_amount NUMERIC(28,2) NOT NULL DEFAULT 0 CHECK (current_amount >= 0),
    monthly_contribution NUMERIC(28,2) CHECK (monthly_contribution IS NULL OR monthly_contribution >= 0),
    target_date DATE,
    expected_yield NUMERIC(8,5) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- BUDGETS
CREATE TABLE IF NOT EXISTS budgets (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    limit_amount NUMERIC(28,2) NOT NULL CHECK (limit_amount > 0),
    rollover_policy TEXT NOT NULL DEFAULT 'none' CHECK (rollover_policy IN ('none','unlimited','months_3')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, category_id)
);
`
}

func main() {
	urlStr := os.Getenv("DATABASE_URL")
	if urlStr == "" {
		fmt.Println("DATABASE_URL not set. Usage: DATABASE_URL=postgres://... go run main.go")
		fmt.Println("  or:  DATABASE_URL=... ./migrator")
		// Check local docker
		urlStr = "postgresql://finhelper:finhelper_pass@localhost:5432/finhelper?sslmode=disable"
		fmt.Printf("Falling back to local dev DB: %s\n", redact(urlStr))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, urlStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	var now string
	if err := pool.QueryRow(ctx, "SELECT NOW()::text").Scan(&now); err != nil {
		fmt.Fprintf(os.Stderr, "ping: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Connected. Server time: %s\n", now)

	// Check existing tables
	var tables []string
	rows, _ := pool.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name")
	for rows.Next() {
		var t string
		rows.Scan(&t)
		tables = append(tables, t)
	}
	rows.Close()
	fmt.Printf("Existing tables (%d): %s\n", len(tables), strings.Join(tables, ", "))

	// Apply migrations
	for name, sql := range migrations {
		if tableExists(tables, name) {
			fmt.Printf("[SKIP] %s — already applied\n", name)
			continue
		}

		fmt.Printf("[RUN]  %s...\n", name)
		start := time.Now()

		stmts := splitSQL(sql)
		applied := 0
		for _, stmt := range stmts {
			if ctx.Err() != nil {
				fmt.Fprintf(os.Stderr, "  context cancelled, stopping early\n")
				break
			}
			_, err := pool.Exec(ctx, stmt)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "already exists") {
					fmt.Printf("  [skip already exists] %s\n", truncate(stmt, 60))
					applied++
					continue
				}
				fmt.Fprintf(os.Stderr, "  [ERROR] %s: %v\n", truncate(stmt, 60), err)
				continue
			}
			applied++
		}
		fmt.Printf("  → %d/%d statements in %v\n", applied, len(stmts), time.Since(start).Round(time.Millisecond).String())
	}

	// Final schema report
	fmt.Println("\n=== Final Schema ===")
	finalRows, err := pool.Query(ctx, `
		SELECT table_name, COUNT(*)::int AS cols
		FROM information_schema.columns
		WHERE table_schema='public'
		GROUP BY table_name
		ORDER BY table_name
	`)
	if err == nil {
		for finalRows.Next() {
			var t string
			var c int
			finalRows.Scan(&t, &c)
			fmt.Printf("  %s (%d cols)\n", t, c)
		}
		finalRows.Close()
	}
}

func tableExists(tables []string, name string) bool {
	// Determine check table name from migration name
	checkNames := map[string]string{
		"0001_init_schema": "users",
	}
	checkName := checkNames[name]
	if checkName == "" {
		return false
	}
	for _, t := range tables {
		if t == checkName {
			return true
		}
	}
	return false
}

func splitSQL(sql string) []string {
	var stmts []string
	var buf strings.Builder
	inLine := false
	inBlock := false

	runes := []rune(sql)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if inLine {
			if ch == '\n' {
				inLine = false
			}
			continue
		}
		if inBlock {
			if ch == '*' && i+1 < len(runes) && runes[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if ch == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			inLine = true
			continue
		}
		if ch == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			inBlock = true
			i++
			continue
		}
		if ch == ';' {
			stmt := strings.TrimSpace(buf.String())
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			buf.Reset()
			continue
		}
		buf.WriteRune(ch)
	}
	stmt := strings.TrimSpace(buf.String())
	if stmt != "" {
		stmts = append(stmts, stmt)
	}
	return stmts
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func redact(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	if u.User != nil {
		u.User = url.User(u.User.Username())
	}
	return u.String()
}

func init() {
	// Override JSON marshaling
	_ = json.Marshal
}
