package migrate

import (
	"context"
	"embed"
	"log"
	"strings"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Run applies SQL migrations idempotently.
// Safe to call multiple times — each migration is skipped if already applied.
func Run(ctx context.Context, pool *storage.Pool) {
	db := pool.DB // *sql.DB

	migrations := []struct {
		name  string
		check string
	}{
		{
			name:  "0001_init",
			check: "SELECT 1 FROM pg_tables WHERE tablename = 'users'",
		},
		{
			name:  "0002_categorization",
			check: "SELECT 1 FROM pg_tables WHERE tablename = 'category_rules'",
		},
		{
			name:  "0003_goals_contributions",
			check: "SELECT 1 FROM pg_tables WHERE tablename = 'goals'",
		},
		{
			name:  "0004_verification",
			check: "SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'verified'",
		},
	}

	for _, m := range migrations {
		// Bail out if context was cancelled between migrations
		if ctx.Err() != nil {
			log.Printf("migration: context cancelled, stopping")
			break
		}

		var ok int
		err := db.QueryRowContext(ctx, m.check).Scan(&ok)
		if err == nil && ok == 1 {
			log.Printf("migration %s already applied, skipping", m.name)
			continue
		}

		data, err := migrationsFS.ReadFile("migrations/" + m.name + ".sql")
		if err != nil {
			log.Printf("migration %s: cannot read embed file: %v", m.name, err)
			continue
		}

		sql := string(data)
		stmts := splitSQL(sql)
		applied := 0
		for _, stmt := range stmts {
			// Bail out of remaining statements if context was cancelled
			if ctx.Err() != nil {
				log.Printf("migration %s: context cancelled, stopping early", m.name)
				break
			}
			_, err := db.ExecContext(ctx, stmt)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "already exists") {
					log.Printf("migration %s: ignoring '%s': %s", m.name, truncate(stmt, 60), errStr)
					applied++
					continue
				}
				log.Printf("migration %s ERROR on '%s': %v", m.name, truncate(stmt, 60), err)
				continue
			}
			applied++
		}
		log.Printf("migration %s: %d/%d statements applied", m.name, applied, len(stmts))
	}

	// Verify core tables exist
	verifySchema(ctx, pool)
}

// splitSQL splits a SQL migration file into individual statements.
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

// verifySchema checks that 6 core tables exist — logs a warning if not.
func verifySchema(ctx context.Context, pool *storage.Pool) {
	db := pool.DB
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('users', 'accounts', 'categories', 'operations', 'budgets', 'goals')").
		Scan(&count)
	if err != nil {
		log.Printf("schema verify error: %v", err)
		return
	}
	log.Printf("schema: %d/6 core tables confirmed", count)
	if count < 6 {
		log.Printf("schema verify: only %d/6 tables found, continuing anyway", count)
	}
}
