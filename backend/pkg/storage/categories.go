package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Category is the persisted category record.
type Category struct {
	ID       int64
	UserID   int64
	Name     string
	ParentID *int64
	IsSystem bool
}

// Sentinel errors for categories.
var (
	// ErrCategoryExists is returned when (user_id, name) already exists.
	ErrCategoryExists = errors.New("storage: category already exists")
	// ErrCategoryNotFound when no row matches (user_id, id).
	ErrCategoryNotFound = errors.New("storage: category not found")
)

// CreateCategory inserts a new user category. System categories are seeded
// separately at registration (see SeedSystemCategories, added later).
func (p *Pool) CreateCategory(ctx context.Context, userID int64, name string, parentID *int64) (Category, error) {
	const q = `
		INSERT INTO categories (user_id, name, parent_id, is_system)
		VALUES ($1, $2, $3, FALSE)
		RETURNING id, user_id, name, parent_id, is_system
	`
	var c Category
	err := p.DB.QueryRowContext(ctx, q, userID, name, parentID).Scan(
		&c.ID, &c.UserID, &c.Name, &c.ParentID, &c.IsSystem,
	)
	if err != nil {
		return Category{}, translatePgError(err, "categories_user_id_name_key", ErrCategoryExists)
	}
	return c, nil
}

// GetCategory returns one category scoped to the owner.
func (p *Pool) GetCategory(ctx context.Context, userID, id int64) (Category, error) {
	const q = `
		SELECT id, user_id, name, parent_id, is_system
		FROM categories
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	var c Category
	err := p.DB.QueryRowContext(ctx, q, id, userID).Scan(
		&c.ID, &c.UserID, &c.Name, &c.ParentID, &c.IsSystem,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, ErrCategoryNotFound
		}
		return Category{}, fmt.Errorf("storage: get category: %w", err)
	}
	return c, nil
}

// ListCategories returns all non-deleted categories for the user.
func (p *Pool) ListCategories(ctx context.Context, userID int64) ([]Category, error) {
	const q = `
		SELECT id, user_id, name, parent_id, is_system
		FROM categories
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY is_system DESC, name
	`
	rows, err := p.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: list categories: %w", err)
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.ParentID, &c.IsSystem); err != nil {
			return nil, fmt.Errorf("storage: scan category: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SeedSystemCategories inserts the default category set for a new user inside
// the caller's transaction. Idempotent on conflict (UNIQUE user_id+name).
// Returns nil if all rows already existed (e.g. re-call after partial failure).
func (p *Pool) SeedSystemCategories(ctx context.Context, tx *sql.Tx, userID int64, names []string) error {
	if tx == nil {
		return errors.New("storage: SeedSystemCategories requires a tx")
	}
	const q = `
		INSERT INTO categories (user_id, name, parent_id, is_system)
		VALUES ($1, $2, NULL, TRUE)
		ON CONFLICT (user_id, name) DO NOTHING
	`
	for _, name := range names {
		if _, err := tx.ExecContext(ctx, q, userID, name); err != nil {
			return fmt.Errorf("storage: seed category %q: %w", name, err)
		}
	}
	return nil
}
