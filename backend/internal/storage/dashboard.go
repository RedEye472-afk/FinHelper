package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
)

// Dashboard aggregates (BUSINESS_LOGIC.md ф.3). Each method is one SQL query
// scoped to the user and the requested [from, to] period. Transfers /
// currency_exchange / refund are excluded from cashflow via AffectsCashflow
// semantics (domain.OperationType), inlined into the SQL so the aggregation
// stays a single round-trip.

// CashflowTotals is the period income/expense/net (BUSINESS_LOGIC ф.3 "Баланс").
// Only income/expense count; transfers/exchanges/refunds are excluded.
type CashflowTotals struct {
	Income  domain.Money // Σ income.amount
	Expense domain.Money // Σ expense.amount (positive magnitude)
	Net     domain.Money // Income − Expense
}

// CashflowForPeriod returns the income/expense/net totals for the user over
// [from, to]. Both bounds are inclusive; a zero value means "no bound" on that
// side. Planned operations (dated in the future) are excluded — the dashboard
// shows realized money only.
func (p *Pool) CashflowForPeriod(ctx context.Context, userID int64, from, to time.Time) (CashflowTotals, error) {
	const q = `
		SELECT
			COALESCE(SUM(CASE WHEN operation_type = 'income'  THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN operation_type = 'expense' THEN amount ELSE 0 END), 0)
		FROM operations
		WHERE user_id = $1 AND deleted_at IS NULL AND is_planned = FALSE
		  AND operation_type IN ('income', 'expense')
	`
	args := []any{userID}
	clause, args := applyPeriodBounds("", from, to, args)
	row := p.DB.QueryRowContext(ctx, q+clause, args...)
	var (
		inc = new(decimalScanner)
		exp = new(decimalScanner)
	)
	if err := row.Scan(inc, exp); err != nil {
		return CashflowTotals{}, fmt.Errorf("storage: cashflow: %w", err)
	}
	income := domain.FromDecimal(inc.d)
	expense := domain.FromDecimal(exp.d)
	return CashflowTotals{Income: income, Expense: expense, Net: income.Sub(expense)}, nil
}

// CategorySpend is one row of the "expenses by category" breakdown.
type CategorySpend struct {
	CategoryID   int64
	CategoryName string // "Без категории" when category_id IS NULL
	Total        domain.Money
}

// ExpensesByCategory returns expense totals grouped by category for the
// period, descending by total. Uncategorised expenses are bucketed under
// CategoryName "Без категории" (id 0). Only 'expense' operations count.
func (p *Pool) ExpensesByCategory(ctx context.Context, userID int64, from, to time.Time) ([]CategorySpend, error) {
	const q = `
		SELECT
			COALESCE(c.id, 0)        AS category_id,
			COALESCE(c.name, 'Без категории') AS category_name,
			COALESCE(SUM(o.amount), 0) AS total
		FROM operations o
		LEFT JOIN categories c ON c.id = o.category_id
		WHERE o.user_id = $1 AND o.deleted_at IS NULL AND o.is_planned = FALSE
		  AND o.operation_type = 'expense'
	`
	args := []any{userID}
	clause, args := applyPeriodBounds("o.", from, to, args)
	tail := `
		GROUP BY c.id, c.name
		ORDER BY total DESC
	`
	rows, err := p.DB.QueryContext(ctx, q+clause+tail, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: expenses by category: %w", err)
	}
	defer rows.Close()

	var out []CategorySpend
	for rows.Next() {
		var (
			r       CategorySpend
			idRaw   int64
			nameRaw string
			totRaw  = new(decimalScanner)
		)
		if err := rows.Scan(&idRaw, &nameRaw, totRaw); err != nil {
			return nil, fmt.Errorf("storage: scan category spend: %w", err)
		}
		r.CategoryID = idRaw
		r.CategoryName = nameRaw
		r.Total = domain.FromDecimal(totRaw.d)
		out = append(out, r)
	}
	return out, rows.Err()
}

// NetWorth is assets − debts (BUSINESS_LOGIC ф.3 "Чистая стоимость"). Assets
// are the Σ positive balances across non-debt accounts; debts are the Σ
// negative balances on debt accounts (stored as the magnitude owed). Net is a
// snapshot, not period-scoped.
type NetWorth struct {
	Assets domain.Money
	Debts  domain.Money // positive magnitude of what is owed
	Net    domain.Money // Assets − Debts
}

// NetWorth computes the user's net worth from cached account balances. Account
// types 'debt' contribute to Debts (their balance magnitude); all other types
// contribute to Assets when their balance is positive. A negative non-debt
// balance (overdraft) reduces Assets.
func (p *Pool) NetWorth(ctx context.Context, userID int64) (NetWorth, error) {
	const q = `
		SELECT
			COALESCE(SUM(CASE WHEN account_type != 'debt' AND balance > 0 THEN balance ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN account_type = 'debt' THEN ABS(balance) ELSE 0 END), 0)
		FROM accounts
		WHERE user_id = $1 AND deleted_at IS NULL
	`
	var (
		assets = new(decimalScanner)
		debts  = new(decimalScanner)
	)
	if err := p.DB.QueryRowContext(ctx, q, userID).Scan(assets, debts); err != nil {
		return NetWorth{}, fmt.Errorf("storage: net worth: %w", err)
	}
	a := domain.FromDecimal(assets.d)
	d := domain.FromDecimal(debts.d)
	return NetWorth{Assets: a, Debts: d, Net: a.Sub(d)}, nil
}

// GoalProgress is one row of the goals snapshot (BUSINESS_LOGIC ф.3 "Прогресс целей").
type GoalProgress struct {
	ID        int64
	Name      string
	Target    domain.Money
	Current   domain.Money
	Progress  domain.Money // Current / Target, as a fraction (0..1+). Zero-safe.
}

// GoalProgresses returns the user's non-deleted goals with a computed progress
// fraction. Progress > 1 means the goal is over-funded; the UI clamps the bar.
//
// Hybrid current-amount model (design spec §3.2): the effective amount saved
// is goals.current_amount (baseline) + Σ goal_contributions.amount (ad-hoc
// top-ups). A LEFT JOIN + COALESCE keeps goals without contributions in the
// result set (their effective current equals the baseline). The progress
// fraction is effective / target, zero-safe for target = 0.
func (p *Pool) GoalProgresses(ctx context.Context, userID int64) ([]GoalProgress, error) {
	const q = `
		SELECT g.id, g.name, g.target_amount,
		       g.current_amount + COALESCE(SUM(gc.amount), 0) AS effective_current,
		       CASE WHEN g.target_amount = 0 THEN 0
		            ELSE (g.current_amount + COALESCE(SUM(gc.amount), 0))::NUMERIC
		                 / g.target_amount END AS progress
		FROM goals g
		LEFT JOIN goal_contributions gc ON gc.goal_id = g.id AND gc.user_id = g.user_id
		WHERE g.user_id = $1 AND g.deleted_at IS NULL
		GROUP BY g.id, g.name, g.target_amount, g.current_amount, g.target_date
		ORDER BY g.target_date NULLS LAST, g.id
	`
	rows, err := p.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: goal progresses: %w", err)
	}
	defer rows.Close()

	var out []GoalProgress
	for rows.Next() {
		var (
			g       GoalProgress
			target  = new(decimalScanner)
			current = new(decimalScanner)
			prog    = new(decimalScanner)
		)
		if err := rows.Scan(&g.ID, &g.Name, target, current, prog); err != nil {
			return nil, fmt.Errorf("storage: scan goal: %w", err)
		}
		g.Target = domain.FromDecimal(target.d)
		g.Current = domain.FromDecimal(current.d)
		g.Progress = domain.FromDecimal(prog.d)
		out = append(out, g)
	}
	return out, rows.Err()
}

// applyPeriodBounds appends a "AND {prefix}operation_date >= $N [AND ... <= $M]"
// clause based on non-zero bounds, returning the clause text and the extended
// args slice. prefix is the table alias for operation_date (e.g. "o.").
func applyPeriodBounds(prefix string, from, to time.Time, args []any) (string, []any) {
	var clauses []string
	if !from.IsZero() {
		args = append(args, from)
		clauses = append(clauses, fmt.Sprintf("AND %soperation_date >= $%d", prefix, len(args)))
	}
	if !to.IsZero() {
		args = append(args, to)
		clauses = append(clauses, fmt.Sprintf("AND %soperation_date <= $%d", prefix, len(args)))
	}
	clause := ""
	for _, c := range clauses {
		clause += " " + c
	}
	return clause, args
}
