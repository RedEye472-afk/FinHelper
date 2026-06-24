package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
)

// ----------------------------------------------------------------------------
// CreateBudget
// ----------------------------------------------------------------------------

func TestCreateBudget_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(q(`INSERT INTO budgets`)).
		WithArgs(int64(7), int64(10), sqlmock.AnyArg(), "none", true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(1), now, now))

	b, err := pool.CreateBudget(ctx, Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverNone, IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}
	if b.ID != 1 {
		t.Errorf("id = %d, want 1", b.ID)
	}
}

func TestCreateBudget_Duplicate(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`INSERT INTO budgets`)).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "budgets_user_id_category_id_key", Message: "dup"})

	_, err := pool.CreateBudget(ctx, Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
	})
	if !errors.Is(err, ErrBudgetExists) {
		t.Fatalf("expected ErrBudgetExists, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// GetBudget
// ----------------------------------------------------------------------------

func TestGetBudget_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT .* FROM budgets WHERE id`)).
		WithArgs(int64(99), int64(7)).
		WillReturnError(sql.ErrNoRows)

	_, err := pool.GetBudget(ctx, 7, 99)
	if !errors.Is(err, ErrBudgetNotFound) {
		t.Fatalf("expected ErrBudgetNotFound, got %v", err)
	}
}

func TestGetBudget_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(q(`SELECT .* FROM budgets WHERE id`)).
		WithArgs(int64(1), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "category_id", "limit_amount", "rollover_policy", "is_active", "created_at", "updated_at"}).
			AddRow(int64(1), int64(7), int64(10), "15000.00", "months_3", true, now, now))

	b, err := pool.GetBudget(ctx, 7, 1)
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if b.RolloverPolicy != domain.RolloverMonths3 {
		t.Errorf("rollover = %s, want months_3", b.RolloverPolicy)
	}
	if !b.LimitAmount.Equal(domain.MustParseMoney("15000.00")) {
		t.Errorf("limit = %s, want 15000.00", b.LimitAmount)
	}
}

// ----------------------------------------------------------------------------
// ListBudgets
// ----------------------------------------------------------------------------

func TestListBudgets_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"id", "user_id", "category_id", "limit_amount", "rollover_policy", "is_active", "created_at", "updated_at"}).
		AddRow(int64(1), int64(7), int64(10), "15000.00", "none", true, now, now).
		AddRow(int64(2), int64(7), int64(20), "5000.00", "months_3", true, now, now)
	mock.ExpectQuery(q(`SELECT .* FROM budgets WHERE user_id`)).
		WithArgs(int64(7)).
		WillReturnRows(rows)

	bs, err := pool.ListBudgets(ctx, 7)
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(bs) != 2 {
		t.Fatalf("expected 2 budgets, got %d", len(bs))
	}
}

// ----------------------------------------------------------------------------
// UpdateBudget
// ----------------------------------------------------------------------------

func TestUpdateBudget_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(q(`UPDATE budgets SET`)).
		WithArgs(sqlmock.AnyArg(), "unlimited", false, int64(1), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "category_id", "limit_amount", "rollover_policy", "is_active", "created_at", "updated_at"}).
			AddRow(int64(1), int64(7), int64(10), "20000.00", "unlimited", false, now, now))

	b, err := pool.UpdateBudget(ctx, Budget{
		ID: 1, UserID: 7, LimitAmount: domain.MustParseMoney("20000.00"),
		RolloverPolicy: domain.RolloverUnlimited, IsActive: false,
	})
	if err != nil {
		t.Fatalf("UpdateBudget: %v", err)
	}
	if !b.LimitAmount.Equal(domain.MustParseMoney("20000.00")) {
		t.Errorf("limit = %s, want 20000.00", b.LimitAmount)
	}
	if b.IsActive {
		t.Errorf("expected inactive after update")
	}
}

func TestUpdateBudget_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`UPDATE budgets SET`)).
		WillReturnError(sql.ErrNoRows)

	_, err := pool.UpdateBudget(ctx, Budget{
		ID: 999, UserID: 7, LimitAmount: domain.MustParseMoney("1.00"),
	})
	if !errors.Is(err, ErrBudgetNotFound) {
		t.Fatalf("expected ErrBudgetNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// DeleteBudget
// ----------------------------------------------------------------------------

func TestDeleteBudget_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectExec(q(`UPDATE budgets SET deleted_at`)).
		WithArgs(int64(99), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := pool.DeleteBudget(ctx, 7, 99)
	if !errors.Is(err, ErrBudgetNotFound) {
		t.Fatalf("expected ErrBudgetNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// SpendForCategory
// ----------------------------------------------------------------------------

func TestSpendForCategory_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(q(`SELECT COALESCE\(SUM\(amount\), 0\) FROM operations`)).
		WithArgs(int64(7), int64(10), from, to).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow("12000.00"))

	spend, err := pool.SpendForCategory(ctx, 7, 10, from, to)
	if err != nil {
		t.Fatalf("SpendForCategory: %v", err)
	}
	if !spend.Equal(domain.MustParseMoney("12000.00")) {
		t.Errorf("spend = %s, want 12000.00", spend)
	}
}

func TestSpendForCategory_NoRows_Zero(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT COALESCE\(SUM\(amount\), 0\) FROM operations`)).
		WithArgs(int64(7), int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow("0"))

	spend, err := pool.SpendForCategory(ctx, 7, 10, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("SpendForCategory: %v", err)
	}
	if !spend.Equal(domain.Zero) {
		t.Errorf("spend = %s, want 0", spend)
	}
}
