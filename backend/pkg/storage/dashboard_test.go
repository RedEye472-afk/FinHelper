package storage

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
)

// ----------------------------------------------------------------------------
// CashflowForPeriod
// ----------------------------------------------------------------------------

func TestCashflowForPeriod_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(q(`SELECT .* FROM operations WHERE user_id`)).
		WithArgs(int64(7), from, to).
		WillReturnRows(sqlmock.NewRows([]string{"inc", "exp"}).AddRow("100000.00", "60000.00"))

	cf, err := pool.CashflowForPeriod(ctx, 7, from, to)
	if err != nil {
		t.Fatalf("CashflowForPeriod: %v", err)
	}
	if !cf.Income.Equal(domain.MustParseMoney("100000.00")) {
		t.Errorf("income = %s, want 100000.00", cf.Income)
	}
	if !cf.Expense.Equal(domain.MustParseMoney("60000.00")) {
		t.Errorf("expense = %s, want 60000.00", cf.Expense)
	}
	if !cf.Net.Equal(domain.MustParseMoney("40000.00")) {
		t.Errorf("net = %s, want 40000.00", cf.Net)
	}
}

func TestCashflowForPeriod_NoRows_Zero(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT .* FROM operations WHERE user_id`)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"inc", "exp"}).AddRow("0", "0"))

	cf, err := pool.CashflowForPeriod(ctx, 7, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("CashflowForPeriod: %v", err)
	}
	if !cf.Net.Equal(domain.Zero) {
		t.Errorf("net = %s, want 0 on no rows", cf.Net)
	}
}

// ----------------------------------------------------------------------------
// ExpensesByCategory
// ----------------------------------------------------------------------------

func TestExpensesByCategory_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"category_id", "category_name", "total"}).
		AddRow(int64(1), "Продукты", "30000.00").
		AddRow(int64(0), "Без категории", "5000.00")
	mock.ExpectQuery(q(`SELECT .* FROM operations o LEFT JOIN categories`)).
		WithArgs(int64(7)).
		WillReturnRows(rows)

	cats, err := pool.ExpensesByCategory(ctx, 7, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ExpensesByCategory: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
	if cats[0].CategoryName != "Продукты" || !cats[0].Total.Equal(domain.MustParseMoney("30000.00")) {
		t.Errorf("first row = %+v", cats[0])
	}
	// Uncategorised bucket has id 0.
	if cats[1].CategoryID != 0 || cats[1].CategoryName != "Без категории" {
		t.Errorf("uncategorised row = %+v", cats[1])
	}
}

// ----------------------------------------------------------------------------
// NetWorth
// ----------------------------------------------------------------------------

func TestNetWorth_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT .* FROM accounts WHERE user_id`)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"assets", "debts"}).AddRow("500000.00", "100000.00"))

	nw, err := pool.NetWorth(ctx, 7)
	if err != nil {
		t.Fatalf("NetWorth: %v", err)
	}
	if !nw.Assets.Equal(domain.MustParseMoney("500000.00")) {
		t.Errorf("assets = %s, want 500000.00", nw.Assets)
	}
	if !nw.Debts.Equal(domain.MustParseMoney("100000.00")) {
		t.Errorf("debts = %s, want 100000.00", nw.Debts)
	}
	if !nw.Net.Equal(domain.MustParseMoney("400000.00")) {
		t.Errorf("net = %s, want 400000.00", nw.Net)
	}
}

// ----------------------------------------------------------------------------
// GoalProgresses
// ----------------------------------------------------------------------------

func TestGoalProgresses_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	// Hybrid model column names: effective_current (baseline + Σ contributions).
	rows := sqlmock.NewRows([]string{"id", "name", "target_amount", "effective_current", "progress"}).
		AddRow(int64(1), "Подушка", "300000.00", "150000.00", "0.5000000000000000")
	mock.ExpectQuery(q(`SELECT .* FROM goals g LEFT JOIN goal_contributions`)).
		WithArgs(int64(7)).
		WillReturnRows(rows)

	goals, err := pool.GoalProgresses(ctx, 7)
	if err != nil {
		t.Fatalf("GoalProgresses: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
	if goals[0].Name != "Подушка" {
		t.Errorf("name = %s", goals[0].Name)
	}
	if !goals[0].Progress.Equal(domain.FromDecimal(decimal.NewFromFloat(0.5))) {
		t.Errorf("progress = %s, want 0.5", goals[0].Progress)
	}
}

func TestGoalProgresses_ZeroTarget_NoDivision(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	// target=0 → progress=0 (CASE guards against division by zero).
	rows := sqlmock.NewRows([]string{"id", "name", "target_amount", "effective_current", "progress"}).
		AddRow(int64(2), "Без суммы", "0", "0", "0")
	mock.ExpectQuery(q(`SELECT .* FROM goals g LEFT JOIN goal_contributions`)).
		WithArgs(int64(7)).
		WillReturnRows(rows)

	goals, err := pool.GoalProgresses(ctx, 7)
	if err != nil {
		t.Fatalf("GoalProgresses: %v", err)
	}
	if !goals[0].Progress.Equal(domain.Zero) {
		t.Errorf("progress = %s, want 0 for zero-target goal", goals[0].Progress)
	}
}
