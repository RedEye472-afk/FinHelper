// Package dashboard implements BUSINESS_LOGIC.md ф.3 — the "Финансовое
// здоровье" summary.
//
// The service is pure orchestration over four storage aggregates:
//   - CashflowForPeriod (income/expense/net, transfers excluded)
//   - ExpensesByCategory (breakdown for the chart)
//   - NetWorth (assets − debts snapshot)
//   - GoalProgresses (progress bars)
//
// It owns the period-resolution logic (month/quarter/year → time bounds) so
// the SQL layer stays agnostic of calendar concepts. Money is decimal
// end-to-end; the service never touches float64.
package dashboard

import (
	"context"
	"errors"
	"time"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/pkg/storage"
)

// Repo is the storage contract the dashboard depends on. Satisfied by
// *storage.Pool; tests substitute a fake.
type Repo interface {
	CashflowForPeriod(ctx context.Context, userID int64, from, to time.Time) (storage.CashflowTotals, error)
	ExpensesByCategory(ctx context.Context, userID int64, from, to time.Time) ([]storage.CategorySpend, error)
	NetWorth(ctx context.Context, userID int64) (storage.NetWorth, error)
	GoalProgresses(ctx context.Context, userID int64) ([]storage.GoalProgress, error)
}

// Service is the dashboard business layer. Construct once at boot, share
// across requests — it holds no per-request state.
type Service struct {
	repo Repo
	now  func() time.Time
}

// NewService returns a Service. repo must be non-nil.
func NewService(repo Repo) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Sentinel errors.
var (
	// ErrInvalidArgument — request failed validation (e.g. unknown period).
	ErrInvalidArgument = errors.New("dashboard: invalid argument")
)

// Period is the dashboard's time window (BUSINESS_LOGIC ф.3: "Период
// (месяц/квартал/год)").
type Period string

const (
	PeriodMonth    Period = "month"
	PeriodQuarter  Period = "quarter"
	PeriodYear     Period = "year"
)

// CustomRange lets a caller pass explicit [from, to] bounds instead of a named
// period. When non-empty, it takes precedence over Period in Compute.
type CustomRange struct {
	From, To time.Time
}

// Summary is the full dashboard payload (BUSINESS_LOGIC ф.3 "С (Результат)").
type Summary struct {
	Period         Period     `json:"period"`
	From           time.Time  `json:"from"`
	To             time.Time  `json:"to"`
	Income         domain.Money `json:"income"`
	Expense        domain.Money `json:"expense"`
	Net            domain.Money `json:"net"`
	ByCategory     []storage.CategorySpend `json:"by_category"`
	NetWorth       storage.NetWorth `json:"net_worth"`
	Goals          []storage.GoalProgress `json:"goals"`
}

// Compute assembles the dashboard summary for the user over the given period.
// If rng is non-zero, it overrides period with explicit bounds.
func (s *Service) Compute(ctx context.Context, userID int64, period Period, rng CustomRange) (Summary, error) {
	if userID == 0 {
		return Summary{}, errors.New("dashboard: user_id required")
	}
	from, to, err := s.resolveBounds(period, rng)
	if err != nil {
		return Summary{}, err
	}

	cf, err := s.repo.CashflowForPeriod(ctx, userID, from, to)
	if err != nil {
		return Summary{}, err
	}
	byCat, err := s.repo.ExpensesByCategory(ctx, userID, from, to)
	if err != nil {
		return Summary{}, err
	}
	nw, err := s.repo.NetWorth(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	goals, err := s.repo.GoalProgresses(ctx, userID)
	if err != nil {
		return Summary{}, err
	}

	if byCat == nil {
		byCat = []storage.CategorySpend{} // stable JSON: [] not null
	}
	if goals == nil {
		goals = []storage.GoalProgress{}
	}
	return Summary{
		Period: period, From: from, To: to,
		Income: cf.Income, Expense: cf.Expense, Net: cf.Net,
		ByCategory: byCat, NetWorth: nw, Goals: goals,
	}, nil
}

// resolveBounds converts a named period into [from, to) bounds anchored at
// "now". Month/quarter/year are calendar-bounded and include the whole of the
// current unit. A custom range passes through after a validity check.
func (s *Service) resolveBounds(period Period, rng CustomRange) (time.Time, time.Time, error) {
	if !rng.From.IsZero() || !rng.To.IsZero() {
		if rng.From.IsZero() || rng.To.IsZero() {
			return time.Time{}, time.Time{}, errors.New("custom range requires both from and to")
		}
		if rng.To.Before(rng.From) {
			return time.Time{}, time.Time{}, errors.New("custom range: to before from")
		}
		return rng.From, rng.To, nil
	}
	now := s.now()
	switch period {
	case PeriodMonth:
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to := from.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return from, to, nil
	case PeriodQuarter:
		q := (int(now.Month())-1)/3 // 0..3
		from := time.Date(now.Year(), time.Month(q*3+1), 1, 0, 0, 0, 0, now.Location())
		to := from.AddDate(0, 3, 0).Add(-time.Nanosecond)
		return from, to, nil
	case PeriodYear:
		from := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
		to := from.AddDate(1, 0, 0).Add(-time.Nanosecond)
		return from, to, nil
	default:
		return time.Time{}, time.Time{}, errors.New("dashboard: unknown period")
	}
}
