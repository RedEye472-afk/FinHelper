package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
)

// ----------------------------------------------------------------------------
// CreateAccount
// ----------------------------------------------------------------------------

func TestCreateAccount_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"id", "user_id", "name", "account_type", "currency", "balance", "created_at", "updated_at"}).
		AddRow(int64(1), int64(7), "Наличные", "cash", "RUB", "0", now, now)

	mock.ExpectQuery(q(`INSERT INTO accounts`)).
		WithArgs(int64(7), "Наличные", "cash", "RUB").
		WillReturnRows(rows)

	a, err := pool.CreateAccount(ctx, 7, "Наличные", domain.AccountCash, "RUB")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if a.ID != 1 || a.Type != domain.AccountCash || !a.Balance.Equal(domain.Zero) {
		t.Errorf("unexpected account: %+v", a)
	}
}

func TestCreateAccount_Duplicate(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`INSERT INTO accounts`)).
		WithArgs(int64(7), "Дупль", "cash", "RUB").
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "accounts_user_id_name_key", Message: "duplicate"})

	_, err := pool.CreateAccount(ctx, 7, "Дупль", domain.AccountCash, "RUB")
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("expected ErrAccountExists, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// GetAccount
// ----------------------------------------------------------------------------

func TestGetAccount_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT id, user_id, name, account_type, currency, balance, created_at, updated_at FROM accounts`)).
		WithArgs(int64(99), int64(7)).
		WillReturnError(sql.ErrNoRows)

	_, err := pool.GetAccount(ctx, 7, 99)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// SetAccountBalance
// ----------------------------------------------------------------------------

func TestSetAccountBalance_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectExec(q(`UPDATE accounts SET balance`)).
		WithArgs("123.45", int64(1), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := pool.SetAccountBalance(ctx, 7, 1, domain.MustParseMoney("123.45")); err != nil {
		t.Fatalf("SetAccountBalance: %v", err)
	}
}

func TestSetAccountBalance_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectExec(q(`UPDATE accounts SET balance`)).
		WithArgs("0", int64(99), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := pool.SetAccountBalance(ctx, 7, 99, domain.Zero)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// ListAccounts
// ----------------------------------------------------------------------------

func TestListAccounts_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"id", "user_id", "name", "account_type", "currency", "balance", "created_at", "updated_at"}).
		AddRow(int64(1), int64(7), "Наличные", "cash", "RUB", "100.00", now, now).
		AddRow(int64(2), int64(7), "Банк", "bank", "RUB", "5000.00", now, now)
	mock.ExpectQuery(q(`SELECT .* FROM accounts WHERE user_id`)).
		WithArgs(int64(7)).
		WillReturnRows(rows)

	accs, err := pool.ListAccounts(ctx, 7)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(accs) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accs))
	}
	if !accs[0].Balance.Equal(domain.MustParseMoney("100.00")) {
		t.Errorf("first balance = %s, want 100.00", accs[0].Balance)
	}
}
