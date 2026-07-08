package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
)

// Account is the persisted account record (mirrors accounts table row).
type Account struct {
	ID        int64
	UserID    int64
	Name      string
	Type      domain.AccountType
	Currency  string
	Balance   domain.Money
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Sentinel errors for accounts.
var (
	// ErrAccountExists is returned by CreateAccount when (user_id, name) already exists.
	ErrAccountExists = errors.New("storage: account already exists")
	// ErrAccountNotFound is returned when no account row matches the (user_id, id).
	ErrAccountNotFound = errors.New("storage: account not found")
)

// CreateAccount inserts a new account. balance starts at 0; it is maintained
// by the operations service, never set directly by the caller.
func (p *Pool) CreateAccount(ctx context.Context, userID int64, name string, accType domain.AccountType, currency string) (Account, error) {
	const q = `
		INSERT INTO accounts (user_id, name, account_type, currency, balance)
		VALUES ($1, $2, $3, $4, 0)
		RETURNING id, user_id, name, account_type, currency, balance, created_at, updated_at
	`
	var (
		a           Account
		balRaw      decimal.Decimal
		accTypeStr  string
	)
	err := p.DB.QueryRowContext(ctx, q, userID, name, string(accType), currency).Scan(
		&a.ID, &a.UserID, &a.Name, &accTypeStr, &a.Currency, &balRaw, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return Account{}, translatePgError(err, "accounts_user_id_name_key", ErrAccountExists)
	}
	a.Type = domain.AccountType(accTypeStr)
	a.Balance = domain.FromDecimal(balRaw)
	return a, nil
}

// GetAccount returns one account scoped to the owner, or ErrAccountNotFound.
// The user_id filter is mandatory — never query accounts across users.
func (p *Pool) GetAccount(ctx context.Context, userID, id int64) (Account, error) {
	const q = `
		SELECT id, user_id, name, account_type, currency, balance, created_at, updated_at
		FROM accounts
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	var (
		a          Account
		balRaw     decimal.Decimal
		accTypeStr string
	)
	err := p.DB.QueryRowContext(ctx, q, id, userID).Scan(
		&a.ID, &a.UserID, &a.Name, &accTypeStr, &a.Currency, &balRaw, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, ErrAccountNotFound
		}
		return Account{}, fmt.Errorf("storage: get account: %w", err)
	}
	a.Type = domain.AccountType(accTypeStr)
	a.Balance = domain.FromDecimal(balRaw)
	return a, nil
}

// ListAccounts returns all non-deleted accounts for the user, ordered by id.
// Intended for dashboard / account pickers; result set is small per user.
func (p *Pool) ListAccounts(ctx context.Context, userID int64) ([]Account, error) {
	const q = `
		SELECT id, user_id, name, account_type, currency, balance, created_at, updated_at
		FROM accounts
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY id
	`
	rows, err := p.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: list accounts: %w", err)
	}
	defer rows.Close()

	var out []Account
	for rows.Next() {
		var (
			a          Account
			balRaw     decimal.Decimal
			accTypeStr string
		)
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Name, &accTypeStr, &a.Currency, &balRaw, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("storage: scan account: %w", err)
		}
		a.Type = domain.AccountType(accTypeStr)
		a.Balance = domain.FromDecimal(balRaw)
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateAccount updates the account name and/or type. Returns the updated
// account or ErrAccountNotFound.
func (p *Pool) UpdateAccount(ctx context.Context, userID, id int64, name string, accType domain.AccountType) (Account, error) {
	const q = `
		UPDATE accounts
		SET name = $1, account_type = $2, updated_at = now()
		WHERE id = $3 AND user_id = $4 AND deleted_at IS NULL
		RETURNING id, user_id, name, account_type, currency, balance, created_at, updated_at
	`
	var (
		a          Account
		balRaw     decimal.Decimal
		accTypeStr string
	)
	err := p.DB.QueryRowContext(ctx, q, name, string(accType), id, userID).Scan(
		&a.ID, &a.UserID, &a.Name, &accTypeStr, &a.Currency, &balRaw, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, ErrAccountNotFound
		}
		return Account{}, fmt.Errorf("storage: update account: %w", err)
	}
	a.Type = domain.AccountType(accTypeStr)
	a.Balance = domain.FromDecimal(balRaw)
	return a, nil
}

// DeleteAccount soft-deletes an account by setting deleted_at. Returns
// ErrAccountNotFound if the account does not exist or is already deleted.
func (p *Pool) DeleteAccount(ctx context.Context, userID, id int64) error {
	const q = `UPDATE accounts SET deleted_at = now() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	res, err := p.DB.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("storage: delete account: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: delete account rows affected: %w", err)
	}
	if n == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// SetAccountBalance overwrites the cached balance. Used by the operations
// service after recomputing balance from operation history — never by direct
// caller action (BUSINESS_LOGIC.md: balance is derived, not user-input).
func (p *Pool) SetAccountBalance(ctx context.Context, userID, id int64, balance domain.Money) error {
	const q = `UPDATE accounts SET balance = $1 WHERE id = $2 AND user_id = $3 AND deleted_at IS NULL`
	res, err := p.DB.ExecContext(ctx, q, balance.Decimal(), id, userID)
	if err != nil {
		return fmt.Errorf("storage: set account balance: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: set account balance rows affected: %w", err)
	}
	if n == 0 {
		return ErrAccountNotFound
	}
	return nil
}
