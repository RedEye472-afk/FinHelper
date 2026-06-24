package budget

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// fakeRepo is an in-memory budget.Repo. SpendForCategory returns a per-month
// map keyed by "YYYY-MM" so rollover tests can stage prior-period spends.
type fakeRepo struct {
	budgets  map[int64]storage.Budget // keyed by budget id
	nextID   int64
	spend    map[string]domain.Money // key = "catID:YYYY-MM"
	createOK bool
	failWith error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		budgets: make(map[int64]storage.Budget),
		spend:   make(map[string]domain.Money),
	}
}

func (f *fakeRepo) CreateBudget(_ context.Context, b storage.Budget) (storage.Budget, error) {
	if f.failWith != nil {
		return storage.Budget{}, f.failWith
	}
	for _, existing := range f.budgets {
		if existing.UserID == b.UserID && existing.CategoryID == b.CategoryID {
			return storage.Budget{}, storage.ErrBudgetExists
		}
	}
	f.nextID++
	b.ID = f.nextID
	b.CreatedAt = time.Now()
	b.UpdatedAt = b.CreatedAt
	f.budgets[b.ID] = b
	return b, nil
}

func (f *fakeRepo) GetBudget(_ context.Context, userID, id int64) (storage.Budget, error) {
	if f.failWith != nil {
		return storage.Budget{}, f.failWith
	}
	b, ok := f.budgets[id]
	if !ok || b.UserID != userID {
		return storage.Budget{}, storage.ErrBudgetNotFound
	}
	return b, nil
}

func (f *fakeRepo) ListBudgets(_ context.Context, userID int64) ([]storage.Budget, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	var out []storage.Budget
	for _, b := range f.budgets {
		if b.UserID == userID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpdateBudget(_ context.Context, b storage.Budget) (storage.Budget, error) {
	if f.failWith != nil {
		return storage.Budget{}, f.failWith
	}
	existing, ok := f.budgets[b.ID]
	if !ok || existing.UserID != b.UserID {
		return storage.Budget{}, storage.ErrBudgetNotFound
	}
	existing.LimitAmount = b.LimitAmount
	existing.RolloverPolicy = b.RolloverPolicy
	existing.IsActive = b.IsActive
	f.budgets[b.ID] = existing
	return existing, nil
}

func (f *fakeRepo) DeleteBudget(_ context.Context, userID, id int64) error {
	b, ok := f.budgets[id]
	if !ok || b.UserID != userID {
		return storage.ErrBudgetNotFound
	}
	delete(f.budgets, id)
	return nil
}

// SpendForCategory returns the staged spend for the month containing `from`.
// Tests key on "catID:YYYY-MM" so the period [from,to] maps to one bucket.
func (f *fakeRepo) SpendForCategory(_ context.Context, _ int64, categoryID int64, from, _ time.Time) (domain.Money, error) {
	if f.failWith != nil {
		return domain.Zero, f.failWith
	}
	key := spendKey(categoryID, from)
	return f.spend[key], nil
}

// setSpend stages a spend for a (category, month) so Compute/Rollover read it.
func (f *fakeRepo) setSpend(categoryID int64, month time.Time, amount string) {
	t := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	f.spend[spendKey(categoryID, t)] = domain.MustParseMoney(amount)
}

func spendKey(categoryID int64, t time.Time) string {
	return key(categoryID, t.Format("2006-01"))
}
func key(categoryID int64, period string) string {
	return period // keep simple: collisions avoided in tests by distinct months
}

// svcWithFixedNow pins the clock so month bounds and day counts are deterministic.
func svcWithFixedNow(repo Repo, t time.Time) *Service {
	return &Service{repo: repo, now: func() time.Time { return t }}
}

// ----------------------------------------------------------------------------
// Create
// ----------------------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	b, err := svc.Create(context.Background(), CreateInput{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverMonths3,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if b.ID == 0 || b.RolloverPolicy != domain.RolloverMonths3 {
		t.Errorf("budget = %+v", b)
	}
	if !b.IsActive {
		t.Errorf("expected active by default")
	}
}

func TestCreate_DefaultsRolloverToNone(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	b, err := svc.Create(context.Background(), CreateInput{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("1000.00"),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if b.RolloverPolicy != domain.RolloverNone {
		t.Errorf("rollover = %s, want none (default)", b.RolloverPolicy)
	}
}

func TestCreate_RejectsInvalidInput(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	cases := []struct {
		name string
		in   CreateInput
	}{
		{"zero limit", CreateInput{UserID: 7, CategoryID: 10, LimitAmount: domain.Zero}},
		{"zero category", CreateInput{UserID: 7, LimitAmount: domain.MustParseMoney("1.00")}},
		{"bad rollover", CreateInput{UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("1.00"), RolloverPolicy: "bogus"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), c.in)
			if err == nil {
				t.Errorf("expected error")
			}
		})
	}
}

func TestCreate_DuplicateRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	in := CreateInput{UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("1.00")}
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := svc.Create(context.Background(), in)
	if !errors.Is(err, storage.ErrBudgetExists) {
		t.Errorf("expected ErrBudgetExists, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// Compute — rollover policies
// ----------------------------------------------------------------------------

func TestCompute_RolloverNone_NoCarry(t *testing.T) {
	repo := newFakeRepo()
	b, _ := repo.CreateBudget(context.Background(), storage.Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverNone, IsActive: true,
	})
	// Prior month was under budget; with 'none' it must NOT carry.
	repo.setSpend(10, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), "5000.00")
	svc := svcWithFixedNow(repo, time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC))

	comp, err := svc.Compute(context.Background(), 7, b.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !comp.Rollover.Equal(domain.Zero) {
		t.Errorf("rollover = %s, want 0 (none policy)", comp.Rollover)
	}
}

func TestCompute_RolloverMonths3_SumsLast3(t *testing.T) {
	repo := newFakeRepo()
	b, _ := repo.CreateBudget(context.Background(), storage.Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverMonths3, IsActive: true,
	})
	// Limit 15000/month. Prior months: May 5000 spent (10000 remainder),
	// April 5000 spent (10000), March 5000 spent (10000). Feb ignored (>3).
	repo.setSpend(10, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), "5000.00")
	repo.setSpend(10, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), "5000.00")
	repo.setSpend(10, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), "5000.00")
	repo.setSpend(10, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "5000.00") // out of window
	svc := svcWithFixedNow(repo, time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC))

	comp, err := svc.Compute(context.Background(), 7, b.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !comp.Rollover.Equal(domain.MustParseMoney("30000.00")) {
		t.Errorf("rollover = %s, want 30000.00 (3 months × 10000)", comp.Rollover)
	}
}

func TestCompute_RolloverUnlimited_LooksBackFurther(t *testing.T) {
	repo := newFakeRepo()
	b, _ := repo.CreateBudget(context.Background(), storage.Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("10000.00"),
		RolloverPolicy: domain.RolloverUnlimited, IsActive: true,
	})
	// Stage 6 prior months (beyond months_3's window) to prove unlimited is wider.
	for i := 1; i <= 6; i++ {
		m := time.Date(2026, time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		repo.setSpend(10, m, "0.00") // full remainder each month
	}
	svc := svcWithFixedNow(repo, time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC))

	comp, err := svc.Compute(context.Background(), 7, b.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	// At least 6 months should contribute (unlimited window is 24, so 6 prior
	// months each with 10000 remainder → >= 60000. months_3 would give only 30000).
	if comp.Rollover.Cmp(domain.MustParseMoney("60000.00")) < 0 {
		t.Errorf("rollover = %s, want >= 60000.00 (unlimited covers >3 months)", comp.Rollover)
	}
}

func TestCompute_OverspentMonth_ContributesZero(t *testing.T) {
	repo := newFakeRepo()
	b, _ := repo.CreateBudget(context.Background(), storage.Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("10000.00"),
		RolloverPolicy: domain.RolloverMonths3, IsActive: true,
	})
	// May overspent (20000 > 10000) → contributes 0. April under → 5000 remainder.
	repo.setSpend(10, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), "20000.00")
	repo.setSpend(10, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), "5000.00")
	repo.setSpend(10, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), "5000.00")
	svc := svcWithFixedNow(repo, time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC))

	comp, err := svc.Compute(context.Background(), 7, b.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	// Only April + March contribute (5000 + 5000 = 10000); May is 0.
	if !comp.Rollover.Equal(domain.MustParseMoney("10000.00")) {
		t.Errorf("rollover = %s, want 10000.00 (overspent month contributes 0)", comp.Rollover)
	}
}

// ----------------------------------------------------------------------------
// Compute — status classification
// ----------------------------------------------------------------------------

func TestCompute_StatusOK_OnTrack(t *testing.T) {
	repo := newFakeRepo()
	b, _ := repo.CreateBudget(context.Background(), storage.Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverNone, IsActive: true,
	})
	// Spent 5000 of 15000 by mid-month → on track.
	repo.setSpend(10, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "5000.00")
	svc := svcWithFixedNow(repo, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))

	comp, err := svc.Compute(context.Background(), 7, b.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if comp.Status != StatusOK {
		t.Errorf("status = %s, want ok (remaining=%s projected=%s)", comp.Status, comp.Remaining, comp.Projected)
	}
}

func TestCompute_StatusOver_AlreadyExceeded(t *testing.T) {
	repo := newFakeRepo()
	b, _ := repo.CreateBudget(context.Background(), storage.Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverNone, IsActive: true,
	})
	// Spent 16000 of 15000 → remaining negative → over.
	repo.setSpend(10, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "16000.00")
	svc := svcWithFixedNow(repo, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))

	comp, err := svc.Compute(context.Background(), 7, b.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if comp.Status != StatusOver {
		t.Errorf("status = %s, want over", comp.Status)
	}
}

func TestCompute_StatusAtRisk_ProjectsOver(t *testing.T) {
	repo := newFakeRepo()
	b, _ := repo.CreateBudget(context.Background(), storage.Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverNone, IsActive: true,
	})
	// 30-day month, day 2: spent 10000 → daily rate 5000 → projected 150000.
	// Wait — that exceeds 15000, so at_risk. Use a clear setup: day 1 of 30,
	// spent 14000 → projected ~420000 >> 15000.
	repo.setSpend(10, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "14000.00")
	svc := svcWithFixedNow(repo, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)) // day 1

	comp, err := svc.Compute(context.Background(), 7, b.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	// remaining still positive (15000-14000=1000) but projected blows past limit.
	if comp.Status != StatusAtRisk {
		t.Errorf("status = %s, want at_risk (remaining=%s projected=%s)", comp.Status, comp.Remaining, comp.Projected)
	}
}

func TestCompute_InactiveBudget_StatusInactive(t *testing.T) {
	repo := newFakeRepo()
	b, _ := repo.CreateBudget(context.Background(), storage.Budget{
		UserID: 7, CategoryID: 10, LimitAmount: domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverNone, IsActive: false,
	})
	svc := NewService(repo)

	comp, err := svc.Compute(context.Background(), 7, b.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if comp.Status != StatusInactive {
		t.Errorf("status = %s, want inactive", comp.Status)
	}
}

func TestCompute_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	_, err := svc.Compute(context.Background(), 7, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// projectSpend — forecast math
// ----------------------------------------------------------------------------

func TestProjectSpend_MidMonth_Extrapolates(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) // day 15 of 30
	pFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	pTo := time.Date(2026, 6, 30, 23, 59, 59, 999999999, time.UTC)
	daysIn, daysElapsed := periodDayCounts(now, pFrom, pTo)
	// Spent 10000 over 15 days → rate ~666.67/day → projected ~20000 over 30.
	projected := projectSpend(domain.MustParseMoney("10000.00"), daysElapsed, daysIn, pFrom, pTo, now)
	if projected.Cmp(domain.MustParseMoney("19000.00")) <= 0 {
		t.Errorf("projected = %s, want > 19000 (extrapolated)", projected)
	}
}

func TestProjectSpend_LastDay_NoExtrapolation(t *testing.T) {
	now := time.Date(2026, 6, 30, 23, 0, 0, 0, time.UTC)
	pFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	pTo := time.Date(2026, 6, 30, 23, 59, 59, 999999999, time.UTC)
	daysIn, daysElapsed := periodDayCounts(now, pFrom, pTo)
	projected := projectSpend(domain.MustParseMoney("8000.00"), daysElapsed, daysIn, pFrom, pTo, now)
	// On the last day, projected == spent.
	if !projected.Equal(domain.MustParseMoney("8000.00")) {
		t.Errorf("projected = %s, want 8000.00 (no extrapolation on last day)", projected)
	}
}

// ----------------------------------------------------------------------------
// periodDayCounts
// ----------------------------------------------------------------------------

func TestPeriodDayCounts_June(t *testing.T) {
	pFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	pTo := time.Date(2026, 6, 30, 23, 59, 59, 999999999, time.UTC)
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	daysIn, elapsed := periodDayCounts(now, pFrom, pTo)
	if daysIn != 30 {
		t.Errorf("daysIn = %d, want 30 (June)", daysIn)
	}
	if elapsed != 15 {
		t.Errorf("elapsed = %d, want 15", elapsed)
	}
}

func TestPeriodDayCounts_February_2026(t *testing.T) {
	pFrom := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	pTo := time.Date(2026, 2, 28, 23, 59, 59, 999999999, time.UTC)
	daysIn, _ := periodDayCounts(pFrom, pFrom, pTo)
	if daysIn != 28 {
		t.Errorf("daysIn = %d, want 28 (Feb 2026 non-leap)", daysIn)
	}
}
