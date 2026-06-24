package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
)

// Budget is the persisted budget record (BUSINESS_LOGIC ф.4). Mirrors the
// budgets table from migration 0001.
type Budget struct {
	ID             int64
	UserID         int64
	CategoryID     int64
	LimitAmount    domain.Money
	RolloverPolicy domain.RolloverPolicy
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Sentinel errors for budgets.
var (
	// ErrBudgetExists — (user_id, category_id) already has a budget.
	ErrBudgetExists = errors.New("storage: budget already exists")
	// ErrBudgetNotFound — no budget matches (user_id, id).
	ErrBudgetNotFound = errors.New("storage: budget not found")
)

// CreateBudget inserts a budget. One budget per (user_id, category_id); a
// duplicate raises ErrBudgetExists. RolloverPolicy is validated at the domain
// layer (domain.ValidateRolloverPolicy) before this call.
func (p *Pool) CreateBudget(ctx context.Context, b Budget) (Budget, error) {
	if b.UserID == 0 {
		return Budget{}, errors.New("storage: budget requires user_id")
	}
	const q = `
		INSERT INTO budgets (user_id, category_id, limit_amount, rollover_policy, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := p.DB.QueryRowContext(ctx, q,
		b.UserID, b.CategoryID, b.LimitAmount.Decimal(), string(b.RolloverPolicy), b.IsActive,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return Budget{}, translatePgError(err, "budgets_user_id_category_id_key", ErrBudgetExists)
	}
	return b, nil
}

// GetBudget returns one budget scoped to the owner.
func (p *Pool) GetBudget(ctx context.Context, userID, id int64) (Budget, error) {
	const q = `
		SELECT id, user_id, category_id, limit_amount, rollover_policy, is_active, created_at, updated_at
		FROM budgets
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	var (
		b           Budget
		limitRaw    = new(decimalScanner)
		policyRaw   string
	)
	err := p.DB.QueryRowContext(ctx, q, id, userID).Scan(
		&b.ID, &b.UserID, &b.CategoryID, limitRaw, &policyRaw, &b.IsActive, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Budget{}, ErrBudgetNotFound
		}
		return Budget{}, fmt.Errorf("storage: get budget: %w", err)
	}
	b.LimitAmount = domain.FromDecimal(limitRaw.d)
	b.RolloverPolicy = domain.RolloverPolicy(policyRaw)
	return b, nil
}

// ListBudgets returns all non-deleted budgets for the user. Joining categories
// is left to the caller/service — storage stays a thin row reader.
func (p *Pool) ListBudgets(ctx context.Context, userID int64) ([]Budget, error) {
	const q = `
		SELECT id, user_id, category_id, limit_amount, rollover_policy, is_active, created_at, updated_at
		FROM budgets
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY id
	`
	rows, err := p.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: list budgets: %w", err)
	}
	defer rows.Close()

	var out []Budget
	for rows.Next() {
		var (
			b         Budget
			limitRaw  = new(decimalScanner)
			policyRaw string
		)
		if err := rows.Scan(&b.ID, &b.UserID, &b.CategoryID, limitRaw, &policyRaw, &b.IsActive, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("storage: scan budget: %w", err)
		}
		b.LimitAmount = domain.FromDecimal(limitRaw.d)
		b.RolloverPolicy = domain.RolloverPolicy(policyRaw)
		out = append(out, b)
	}
	return out, rows.Err()
}

// UpdateBudget mutates limit / rollover / active. Category is immutable post-
// create (it's part of the identity); to reassign, delete + create.
func (p *Pool) UpdateBudget(ctx context.Context, b Budget) (Budget, error) {
	const q = `
		UPDATE budgets
		SET limit_amount = $1, rollover_policy = $2, is_active = $3
		WHERE id = $4 AND user_id = $5 AND deleted_at IS NULL
		RETURNING id, user_id, category_id, limit_amount, rollover_policy, is_active, created_at, updated_at
	`
	var (
		out       Budget
		limitRaw  = new(decimalScanner)
		policyRaw string
	)
	err := p.DB.QueryRowContext(ctx, q,
		b.LimitAmount.Decimal(), string(b.RolloverPolicy), b.IsActive, b.ID, b.UserID,
	).Scan(&out.ID, &out.UserID, &out.CategoryID, limitRaw, &policyRaw, &out.IsActive, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Budget{}, ErrBudgetNotFound
		}
		return Budget{}, fmt.Errorf("storage: update budget: %w", err)
	}
	out.LimitAmount = domain.FromDecimal(limitRaw.d)
	out.RolloverPolicy = domain.RolloverPolicy(policyRaw)
	return out, nil
}

// DeleteBudget soft-deletes a budget.
func (p *Pool) DeleteBudget(ctx context.Context, userID, id int64) error {
	const q = `UPDATE budgets SET deleted_at = NOW(), is_active = FALSE WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	res, err := p.DB.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("storage: delete budget: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: delete budget rows affected: %w", err)
	}
	if n == 0 {
		return ErrBudgetNotFound
	}
	return nil
}

// SpendForCategory returns the Σ of 'expense' operations on the given category
// over [from, to] for the user. This is the "факт трат" input to the budget
// computation (BUSINESS_LOGIC ф.4). Transfers/refunds are excluded by type.
// Planned operations are excluded — budgets track realized spend.
func (p *Pool) SpendForCategory(ctx context.Context, userID, categoryID int64, from, to time.Time) (domain.Money, error) {
	const q = `
		SELECT COALESCE(SUM(amount), 0)
		FROM operations
		WHERE user_id = $1 AND category_id = $2 AND deleted_at IS NULL AND is_planned = FALSE
		  AND operation_type = 'expense'
	`
	args := []any{userID, categoryID}
	clause, args := applyPeriodBounds("", from, to, args)
	tot := new(decimalScanner)
	if err := p.DB.QueryRowContext(ctx, q+clause, args...).Scan(tot); err != nil {
		return domain.Zero, fmt.Errorf("storage: spend for category: %w", err)
	}
	return domain.FromDecimal(tot.d), nil
}
