package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
)

// CategorizationRule is a keyword → category mapping (BUSINESS_LOGIC ф.2,
// Level 2). Keyword is matched case-insensitively as a substring against the
// (PII-masked) counterparty + description.
type CategorizationRule struct {
	ID         int64
	UserID     int64
	Keyword    string // normalized: lowercased + trimmed
	CategoryID int64
	Source     domain.RuleSource
	Priority   int
	IsEnabled  bool
}

// CounterpartyOverride is an exact counterparty → category mapping with a
// confirmation counter (BUSINESS_LOGIC ф.2, Level 3 simplified). Stands in for
// the ML model under the MVP scope-lock; a pure counter keeps it deterministic.
type CounterpartyOverride struct {
	ID            int64
	UserID        int64
	Counterparty  string // normalized: lowercased + trimmed
	CategoryID    int64
	Confirmations int
}

// Sentinel errors for categorization.
var (
	// ErrRuleExists — (user_id, keyword) already exists.
	ErrRuleExists = errors.New("storage: categorization rule already exists")
	// ErrRuleNotFound — no row matches (user_id, id).
	ErrRuleNotFound = errors.New("storage: categorization rule not found")
	// ErrOverrideExists — (user_id, counterparty) already exists (used by upsert path).
	ErrOverrideExists = errors.New("storage: counterparty override already exists")
	// ErrOverrideNotFound — no override row matches (user_id, id) or (user_id, counterparty).
	ErrOverrideNotFound = errors.New("storage: counterparty override not found")
)

// CreateRule inserts a keyword rule. keyword is normalized to lower+trim by
// the caller (service) so matching and uniqueness agree.
func (p *Pool) CreateRule(ctx context.Context, r CategorizationRule) (CategorizationRule, error) {
	if r.UserID == 0 {
		return CategorizationRule{}, errors.New("storage: rule requires user_id")
	}
	const q = `
		INSERT INTO categorization_rules (user_id, keyword, category_id, source, priority, is_enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	priority := r.Priority
	if priority == 0 && r.Source == domain.RuleUser {
		// User rules outrank system rules so a manual correction wins.
		priority = 100
	}
	err := p.DB.QueryRowContext(ctx, q,
		r.UserID, r.Keyword, r.CategoryID, string(r.Source), priority, r.IsEnabled,
	).Scan(&r.ID)
	if err != nil {
		return CategorizationRule{}, translatePgError(err, "categorization_rules_user_id_keyword_key", ErrRuleExists)
	}
	r.Priority = priority
	return r, nil
}

// ListRules returns all enabled rules for the user, highest priority first.
// Disabled and soft-deleted rows are excluded — they're not match candidates.
func (p *Pool) ListRules(ctx context.Context, userID int64) ([]CategorizationRule, error) {
	const q = `
		SELECT id, user_id, keyword, category_id, source, priority, is_enabled
		FROM categorization_rules
		WHERE user_id = $1 AND is_enabled = TRUE AND deleted_at IS NULL
		ORDER BY priority DESC, id
	`
	rows, err := p.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: list rules: %w", err)
	}
	defer rows.Close()

	var out []CategorizationRule
	for rows.Next() {
		var (
			r     CategorizationRule
			src   string
		)
		if err := rows.Scan(&r.ID, &r.UserID, &r.Keyword, &r.CategoryID, &src, &r.Priority, &r.IsEnabled); err != nil {
			return nil, fmt.Errorf("storage: scan rule: %w", err)
		}
		r.Source = domain.RuleSource(src)
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteRule soft-deletes a rule. System rules can be disabled this way too —
// the user effectively "unsubscribes" from a default mapping.
func (p *Pool) DeleteRule(ctx context.Context, userID, id int64) error {
	const q = `UPDATE categorization_rules SET deleted_at = NOW(), is_enabled = FALSE WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	res, err := p.DB.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("storage: delete rule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: delete rule rows affected: %w", err)
	}
	if n == 0 {
		return ErrRuleNotFound
	}
	return nil
}

// SeedSystemRules inserts the default keyword set for a new user inside the
// caller's transaction. Idempotent on conflict (UNIQUE user_id+keyword).
// pairs is an ordered (keyword → categoryName) list; the resolver translates
// names to category ids using the categories seeded in the same tx.
func (p *Pool) SeedSystemRules(ctx context.Context, tx *sql.Tx, userID int64, pairs []SystemRuleSeed) error {
	if tx == nil {
		return errors.New("storage: SeedSystemRules requires a tx")
	}
	const q = `
		INSERT INTO categorization_rules (user_id, keyword, category_id, source, priority, is_enabled)
		VALUES ($1, $2, $3, 'system', 0, TRUE)
		ON CONFLICT (user_id, keyword) DO NOTHING
	`
	for _, s := range pairs {
		if _, err := tx.ExecContext(ctx, q, userID, s.Keyword, s.CategoryID); err != nil {
			return fmt.Errorf("storage: seed rule %q: %w", s.Keyword, err)
		}
	}
	return nil
}

// SystemRuleSeed pairs a keyword with its target category id for seeding.
type SystemRuleSeed struct {
	Keyword    string
	CategoryID int64
}

// GetOverride returns the override for a normalized counterparty, or
// ErrOverrideNotFound. Scoping by user_id is mandatory.
func (p *Pool) GetOverride(ctx context.Context, userID int64, counterparty string) (CounterpartyOverride, error) {
	const q = `
		SELECT id, user_id, counterparty, category_id, confirmations
		FROM counterparty_overrides
		WHERE user_id = $1 AND counterparty = $2
	`
	var o CounterpartyOverride
	err := p.DB.QueryRowContext(ctx, q, userID, counterparty).Scan(
		&o.ID, &o.UserID, &o.Counterparty, &o.CategoryID, &o.Confirmations,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CounterpartyOverride{}, ErrOverrideNotFound
		}
		return CounterpartyOverride{}, fmt.Errorf("storage: get override: %w", err)
	}
	return o, nil
}

// UpsertOverrideConfirmation increments the confirmation counter for an
// existing override, or creates a new one with confirmations=1. Returns the
// resulting row. This is the path the service uses when the user confirms or
// corrects a category — each confirmation moves the row toward the
// LearnThreshold (BUSINESS_LOGIC ф.2).
func (p *Pool) UpsertOverrideConfirmation(ctx context.Context, userID int64, counterparty string, categoryID int64) (CounterpartyOverride, error) {
	const q = `
		INSERT INTO counterparty_overrides (user_id, counterparty, category_id, confirmations)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (user_id, counterparty)
		DO UPDATE SET
			category_id   = EXCLUDED.category_id,
			confirmations = counterparty_overrides.confirmations + 1
		RETURNING id, user_id, counterparty, category_id, confirmations
	`
	var o CounterpartyOverride
	err := p.DB.QueryRowContext(ctx, q, userID, counterparty, categoryID).Scan(
		&o.ID, &o.UserID, &o.Counterparty, &o.CategoryID, &o.Confirmations,
	)
	if err != nil {
		return CounterpartyOverride{}, fmt.Errorf("storage: upsert override: %w", err)
	}
	return o, nil
}

// SetOverrideCategory resets confirmations to 1 when the user explicitly picks
// a different category for an existing counterparty. We reset (not increment)
// because the old confirmations applied to a now-discarded mapping.
func (p *Pool) SetOverrideCategory(ctx context.Context, userID, id int64, categoryID int64) (CounterpartyOverride, error) {
	const q = `
		UPDATE counterparty_overrides
		SET category_id = $1, confirmations = 1
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, counterparty, category_id, confirmations
	`
	var o CounterpartyOverride
	err := p.DB.QueryRowContext(ctx, q, categoryID, id, userID).Scan(
		&o.ID, &o.UserID, &o.Counterparty, &o.CategoryID, &o.Confirmations,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CounterpartyOverride{}, ErrOverrideNotFound
		}
		return CounterpartyOverride{}, fmt.Errorf("storage: set override category: %w", err)
	}
	return o, nil
}

// SeedDefaultsForUser seeds keyword rules inside the caller's registration tx.
// keywordRules is the (keyword → categoryName) default set (see
// service/categorization.SystemKeywordRules). Names are resolved to category
// ids using the system categories seeded in the same tx; any name without a
// matching category is skipped (seed never blocks registration).
//
// This is the single seeding entry point used by AuthHandler.Register so the
// handler stays free of SQL — it just passes the default lists through.
func (p *Pool) SeedDefaultsForUser(ctx context.Context, tx *sql.Tx, userID int64, keywordRules []KeywordRuleSeed) error {
	if tx == nil {
		return errors.New("storage: SeedDefaultsForUser requires a tx")
	}
	if len(keywordRules) == 0 {
		return nil
	}
	// Collect the distinct category names referenced by rules.
	nameSet := make(map[string]struct{}, len(keywordRules))
	for _, r := range keywordRules {
		nameSet[r.CategoryName] = struct{}{}
	}
	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}
	ids, err := resolveCategoryIDs(ctx, tx, userID, names)
	if err != nil {
		return err
	}
	// Translate rules to id-keyed seeds, dropping any whose category is missing.
	seeds := make([]SystemRuleSeed, 0, len(keywordRules))
	for _, r := range keywordRules {
		catID, ok := ids[r.CategoryName]
		if !ok {
			continue
		}
		seeds = append(seeds, SystemRuleSeed{Keyword: r.Keyword, CategoryID: catID})
	}
	return p.SeedSystemRules(ctx, tx, userID, seeds)
}

// KeywordRuleSeed pairs a keyword with its target category NAME (resolved to
// id inside the seeding tx). Lives in storage so transport/auth can pass it
// without importing the categorization service.
type KeywordRuleSeed struct {
	Keyword      string
	CategoryName string
}

// resolveCategoryIDs maps category names to ids for the seeded set, within the
// caller's transaction. Names not present in the user's categories are dropped
// silently — seeding must never block registration on a missing default.
func resolveCategoryIDs(ctx context.Context, tx *sql.Tx, userID int64, names []string) (map[string]int64, error) {
	if len(names) == 0 {
		return map[string]int64{}, nil
	}
	placeholders := make([]string, len(names))
	args := make([]any, 0, len(names)+1)
	args = append(args, userID)
	for i, n := range names {
		args = append(args, n)
		placeholders[i] = fmt.Sprintf("$%d", i+2)
	}
	q := fmt.Sprintf(`
		SELECT name, id FROM categories
		WHERE user_id = $1 AND is_system = TRUE AND deleted_at IS NULL
		  AND name IN (%s)
	`, strings.Join(placeholders, ","))
	rows, err := tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: resolve category ids: %w", err)
	}
	defer rows.Close()
	out := make(map[string]int64, len(names))
	for rows.Next() {
		var (
			name string
			id   int64
		)
		if err := rows.Scan(&name, &id); err != nil {
			return nil, fmt.Errorf("storage: scan category id: %w", err)
		}
		out[name] = id
	}
	return out, rows.Err()
}
