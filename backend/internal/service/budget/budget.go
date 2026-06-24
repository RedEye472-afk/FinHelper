// Package budget implements BUSINESS_LOGIC.md ф.4 — per-category limits with
// a rolling carry-over and an overspend forecast.
//
// The service owns three pieces of logic that storage can't express:
//
//  1. Rollover. The unused balance from previous periods carries into the
//     current one per the policy:
//       - none      → unused amount expires (no carry)
//       - unlimited → every prior unspent period adds up indefinitely
//       - months_3  → only the last 3 months count
//     Overspent past periods contribute zero (you don't go "into debt" on a
//     budget); under-spent periods contribute their positive remainder.
//
//  2. Overspend forecast. With the current spend rate and days remaining in
//     the period, we project the end-of-period spend. If it exceeds the
//     effective limit, the status is "at_risk" / "over".
//
//  3. Period resolution. A budget period is a calendar month anchored at
//     "now" (matching the dashboard's PeriodMonth granularity). Past periods
//     for rollover are whole calendar months.
//
// Money is decimal end-to-end; the service never touches float64.
package budget

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// decimalFromInt converts an int day count into a decimal for rate math.
func decimalFromInt(n int) decimal.Decimal {
	return decimal.NewFromInt(int64(n))
}

// Repo is the storage contract. Satisfied by *storage.Pool; tests substitute
// a fake.
type Repo interface {
	CreateBudget(ctx context.Context, b storage.Budget) (storage.Budget, error)
	GetBudget(ctx context.Context, userID, id int64) (storage.Budget, error)
	ListBudgets(ctx context.Context, userID int64) ([]storage.Budget, error)
	UpdateBudget(ctx context.Context, b storage.Budget) (storage.Budget, error)
	DeleteBudget(ctx context.Context, userID, id int64) error
	SpendForCategory(ctx context.Context, userID, categoryID int64, from, to time.Time) (domain.Money, error)
}

// Service is the budget business layer. Construct once at boot, share across
// requests — it holds no per-request state.
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
	// ErrInvalidArgument — request failed validation.
	ErrInvalidArgument = errors.New("budget: invalid argument")
	// ErrNotFound — budget not found (or belongs to another user).
	ErrNotFound = errors.New("budget: not found")
)

// CreateInput is what the caller supplies to create a budget.
type CreateInput struct {
	UserID         int64
	CategoryID     int64
	LimitAmount    domain.Money
	RolloverPolicy domain.RolloverPolicy
}

// Create validates and persists a budget. Default rollover is 'none'
// (matches the budgets table default); default is_active = true.
func (s *Service) Create(ctx context.Context, in CreateInput) (storage.Budget, error) {
	if in.UserID == 0 {
		return storage.Budget{}, errors.New("budget: user_id required")
	}
	if in.CategoryID <= 0 {
		return storage.Budget{}, errors.New("budget: category_id required")
	}
	if !in.LimitAmount.IsPositive() {
		return storage.Budget{}, errors.New("budget: limit must be positive")
	}
	policy := in.RolloverPolicy
	if policy == "" {
		policy = domain.RolloverNone
	}
	if err := domain.ValidateRolloverPolicy(policy); err != nil {
		return storage.Budget{}, err
	}
	b, err := s.repo.CreateBudget(ctx, storage.Budget{
		UserID: in.UserID, CategoryID: in.CategoryID,
		LimitAmount: in.LimitAmount, RolloverPolicy: policy, IsActive: true,
	})
	if err != nil {
		return storage.Budget{}, mapStorageErr(err)
	}
	return b, nil
}

// Get returns one budget scoped to the owner.
func (s *Service) Get(ctx context.Context, userID, id int64) (storage.Budget, error) {
	b, err := s.repo.GetBudget(ctx, userID, id)
	if err != nil {
		return storage.Budget{}, mapStorageErr(err)
	}
	return b, nil
}

// List returns all budgets for the user.
func (s *Service) List(ctx context.Context, userID int64) ([]storage.Budget, error) {
	bs, err := s.repo.ListBudgets(ctx, userID)
	if err != nil {
		return nil, mapStorageErr(err)
	}
	if bs == nil {
		bs = []storage.Budget{}
	}
	return bs, nil
}

// UpdateInput mutates a budget. CategoryID is identity (immutable).
type UpdateInput struct {
	UserID         int64
	ID             int64
	LimitAmount    domain.Money
	RolloverPolicy domain.RolloverPolicy
	IsActive       bool
}

// Update validates and applies a budget mutation.
func (s *Service) Update(ctx context.Context, in UpdateInput) (storage.Budget, error) {
	if in.UserID == 0 || in.ID == 0 {
		return storage.Budget{}, errors.New("budget: user_id and id required")
	}
	if !in.LimitAmount.IsPositive() {
		return storage.Budget{}, errors.New("budget: limit must be positive")
	}
	if err := domain.ValidateRolloverPolicy(in.RolloverPolicy); err != nil {
		return storage.Budget{}, err
	}
	b, err := s.repo.UpdateBudget(ctx, storage.Budget{
		ID: in.ID, UserID: in.UserID, LimitAmount: in.LimitAmount,
		RolloverPolicy: in.RolloverPolicy, IsActive: in.IsActive,
	})
	if err != nil {
		return storage.Budget{}, mapStorageErr(err)
	}
	return b, nil
}

// Delete soft-deletes a budget.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	if err := s.repo.DeleteBudget(ctx, userID, id); err != nil {
		return mapStorageErr(err)
	}
	return nil
}

// Status is the budget health label for the current period.
type Status string

const (
	StatusOK      Status = "ok"       // on track
	StatusAtRisk  Status = "at_risk"  // projected to exceed
	StatusOver    Status = "over"     // already over the effective limit
	StatusInactive Status = "inactive"
)

// Computation is the result of evaluating one budget against the current
// period (BUSINESS_LOGIC ф.4 "С (Результат)").
type Computation struct {
	Budget         storage.Budget `json:"budget"`
	PeriodFrom     time.Time      `json:"period_from"`
	PeriodTo       time.Time      `json:"period_to"`
	Spent          domain.Money   `json:"spent"`
	Rollover       domain.Money   `json:"rollover"`        // carry-in from prior periods
	EffectiveLimit domain.Money   `json:"effective_limit"` // limit + rollover
	Remaining      domain.Money   `json:"remaining"`       // effective − spent (can be negative)
	Projected      domain.Money   `json:"projected"`       // end-of-period spend forecast
	Status         Status         `json:"status"`
	DaysInPeriod   int            `json:"days_in_period"`
	DaysElapsed    int            `json:"days_elapsed"`
}

// Compute evaluates a single budget for the current month. Rollover is
// computed from prior whole-month periods per the budget's policy.
func (s *Service) Compute(ctx context.Context, userID, budgetID int64) (Computation, error) {
	if userID == 0 || budgetID == 0 {
		return Computation{}, errors.New("budget: user_id and id required")
	}
	b, err := s.repo.GetBudget(ctx, userID, budgetID)
	if err != nil {
		return Computation{}, mapStorageErr(err)
	}
	if !b.IsActive {
		return s.inactiveComputation(b), nil
	}

	now := s.now()
	pFrom, pTo := currentMonthBounds(now)
	spent, err := s.repo.SpendForCategory(ctx, userID, b.CategoryID, pFrom, pTo)
	if err != nil {
		return Computation{}, err
	}
	rollover, err := s.computeRollover(ctx, userID, b, now)
	if err != nil {
		return Computation{}, err
	}
	effective := b.LimitAmount.Add(rollover)
	remaining := effective.Sub(spent)

	daysIn, daysElapsed := periodDayCounts(now, pFrom, pTo)
	projected := projectSpend(spent, daysElapsed, daysIn, pFrom, pTo, now)

	status := classify(remaining, projected, effective)

	return Computation{
		Budget: b, PeriodFrom: pFrom, PeriodTo: pTo,
		Spent: spent, Rollover: rollover, EffectiveLimit: effective,
		Remaining: remaining, Projected: projected, Status: status,
		DaysInPeriod: daysIn, DaysElapsed: daysElapsed,
	}, nil
}

// computeRollover sums the positive remainders (limit − spent) of prior whole
// calendar months, bounded by the policy. None → zero. Unlimited → all months
// back to the budget's creation (approximated by 24 months to bound cost).
// months_3 → last 3 months. Overspent months contribute zero.
func (s *Service) computeRollover(ctx context.Context, userID int64, b storage.Budget, now time.Time) (domain.Money, error) {
	switch b.RolloverPolicy {
	case domain.RolloverNone:
		return domain.Zero, nil
	case domain.RolloverUnlimited, domain.RolloverMonths3:
		// fall through to the loop
	default:
		return domain.Zero, nil
	}

	months := 0
	switch b.RolloverPolicy {
	case domain.RolloverMonths3:
		months = 3
	case domain.RolloverUnlimited:
		months = 24 // bounded lookback; effectively "indefinite" for a UI
	}

	rollover := domain.Zero
	// Start from the month immediately before the current one and go back.
	t := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for i := 0; i < months; i++ {
		t = t.AddDate(0, -1, 0) // first day of the prior month
		mFrom := t
		mTo := t.AddDate(0, 1, 0).Add(-time.Nanosecond)
		spent, err := s.repo.SpendForCategory(ctx, userID, b.CategoryID, mFrom, mTo)
		if err != nil {
			return domain.Zero, err
		}
		remainder := b.LimitAmount.Sub(spent)
		if remainder.IsPositive() {
			rollover = rollover.Add(remainder)
		}
	}
	return rollover, nil
}

// inactiveComputation short-circuits for disabled budgets.
func (s *Service) inactiveComputation(b storage.Budget) Computation {
	return Computation{Budget: b, Status: StatusInactive}
}

// projectSpend forecasts the end-of-period spend by extending the daily rate.
// If no days have elapsed (start of month), it returns the current spend as-is
// (no extrapolation possible). Guarded against divide-by-zero.
func projectSpend(spent domain.Money, daysElapsed, daysIn int, pFrom, pTo time.Time, now time.Time) domain.Money {
	if daysIn <= 0 {
		return spent
	}
	if daysElapsed <= 0 {
		return spent
	}
	// If today is the last day, projected == spent (no remaining rate).
	if now.After(pTo) || now.Equal(pTo) {
		return spent
	}
	// daily rate = spent / daysElapsed; projected = rate * daysIn.
	// Decimal division keeps this exact to scale.
	spentDec := spent.Decimal()
	daily := spentDec.Div(decimalFromInt(daysElapsed))
	projected := daily.Mul(decimalFromInt(daysIn))
	return domain.FromDecimal(projected)
}

// classify maps (remaining, projected, effective) to a Status label.
func classify(remaining, projected, effective domain.Money) Status {
	if remaining.IsNegative() {
		return StatusOver
	}
	// projected > effective → at risk.
	if projected.Cmp(effective) > 0 {
		return StatusAtRisk
	}
	return StatusOK
}

// currentMonthBounds returns [first day, last nanosecond] of now's month.
func currentMonthBounds(now time.Time) (time.Time, time.Time) {
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return from, to
}

// periodDayCounts returns (days in the month, days elapsed including today).
// Elapsed is clamped to [1, daysIn].
func periodDayCounts(now, pFrom, pTo time.Time) (int, int) {
	daysIn := int(pTo.Sub(pFrom).Hours()/24) + 1
	elapsed := int(now.Sub(pFrom).Hours()/24) + 1
	if elapsed < 1 {
		elapsed = 1
	}
	if elapsed > daysIn {
		elapsed = daysIn
	}
	return daysIn, elapsed
}

func mapStorageErr(err error) error {
	switch {
	case errors.Is(err, storage.ErrBudgetExists):
		return err
	case errors.Is(err, storage.ErrBudgetNotFound):
		return ErrNotFound
	default:
		return err
	}
}
