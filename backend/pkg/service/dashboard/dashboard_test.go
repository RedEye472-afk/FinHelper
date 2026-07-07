package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/pkg/storage"
)

// fakeRepo is an in-memory dashboard.Repo. It records the bounds it was called
// with so tests can assert period resolution, and returns canned aggregates.
type fakeRepo struct {
	cashflow storage.CashflowTotals
	byCat    []storage.CategorySpend
	netWorth storage.NetWorth
	goals    []storage.GoalProgress
	// lastFrom/lastTo capture the resolved bounds for assertions.
	lastFrom, lastTo time.Time
	failWith        error
}

func (f *fakeRepo) CashflowForPeriod(_ context.Context, _ int64, from, to time.Time) (storage.CashflowTotals, error) {
	if f.failWith != nil {
		return storage.CashflowTotals{}, f.failWith
	}
	f.lastFrom, f.lastTo = from, to
	return f.cashflow, nil
}
func (f *fakeRepo) ExpensesByCategory(_ context.Context, _ int64, from, to time.Time) ([]storage.CategorySpend, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.lastFrom, f.lastTo = from, to
	return f.byCat, nil
}
func (f *fakeRepo) NetWorth(_ context.Context, _ int64) (storage.NetWorth, error) {
	if f.failWith != nil {
		return storage.NetWorth{}, f.failWith
	}
	return f.netWorth, nil
}
func (f *fakeRepo) GoalProgresses(_ context.Context, _ int64) ([]storage.GoalProgress, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	return f.goals, nil
}

// fixedNow pins the service clock so period bounds are deterministic.
func newSvcWithFixedNow(repo Repo, t time.Time) *Service {
	return &Service{repo: repo, now: func() time.Time { return t }}
}

// ----------------------------------------------------------------------------
// resolveBounds — named periods
// ----------------------------------------------------------------------------

func TestResolveBounds_Month(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvcWithFixedNow(repo, time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC))
	from, to, err := svc.resolveBounds(PeriodMonth, CustomRange{})
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	wantFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
	if !from.Equal(wantFrom) {
		t.Errorf("from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Errorf("to = %v, want %v", to, wantTo)
	}
}

func TestResolveBounds_Quarter(t *testing.T) {
	repo := &fakeRepo{}
	// June 2026 → Q2 (Apr–Jun).
	svc := newSvcWithFixedNow(repo, time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC))
	from, to, err := svc.resolveBounds(PeriodQuarter, CustomRange{})
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	wantFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !from.Equal(wantFrom) {
		t.Errorf("from = %v, want %v (Q2 start)", from, wantFrom)
	}
	// Q2 end is end of June.
	if to.Month() != time.June || to.Year() != 2026 {
		t.Errorf("to = %v, want end of June 2026", to)
	}
}

func TestResolveBounds_Year(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvcWithFixedNow(repo, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))
	from, to, err := svc.resolveBounds(PeriodYear, CustomRange{})
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	if from.Month() != time.January || from.Year() != 2026 {
		t.Errorf("from = %v, want Jan 2026", from)
	}
	if to.Year() != 2026 || to.Month() != time.December {
		t.Errorf("to = %v, want Dec 2026", to)
	}
}

func TestResolveBounds_UnknownPeriod(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvcWithFixedNow(repo, time.Now())
	_, _, err := svc.resolveBounds(Period("lifetime"), CustomRange{})
	if err == nil {
		t.Errorf("expected error for unknown period")
	}
}

// ----------------------------------------------------------------------------
// resolveBounds — custom range
// ----------------------------------------------------------------------------

func TestResolveBounds_CustomRange(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvcWithFixedNow(repo, time.Now())
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	gotFrom, gotTo, err := svc.resolveBounds(PeriodMonth, CustomRange{From: from, To: to})
	if err != nil {
		t.Fatalf("resolveBounds: %v", err)
	}
	// Custom range overrides the named period.
	if !gotFrom.Equal(from) || !gotTo.Equal(to) {
		t.Errorf("custom range = (%v, %v), want (%v, %v)", gotFrom, gotTo, from, to)
	}
}

func TestResolveBounds_CustomRange_PartialRejected(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvcWithFixedNow(repo, time.Now())
	_, _, err := svc.resolveBounds("", CustomRange{From: time.Now()})
	if err == nil {
		t.Errorf("expected error for partial custom range")
	}
}

func TestResolveBounds_CustomRange_ReversedRejected(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvcWithFixedNow(repo, time.Now())
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.resolveBounds("", CustomRange{From: from, To: to})
	if err == nil {
		t.Errorf("expected error for reversed range")
	}
}

// ----------------------------------------------------------------------------
// Compute — aggregation + nil-slice normalization
// ----------------------------------------------------------------------------

func TestCompute_AssemblesAllSections(t *testing.T) {
	repo := &fakeRepo{
		cashflow: storage.CashflowTotals{
			Income:  domain.MustParseMoney("100000.00"),
			Expense: domain.MustParseMoney("60000.00"),
			Net:     domain.MustParseMoney("40000.00"),
		},
		byCat: []storage.CategorySpend{
			{CategoryID: 1, CategoryName: "Продукты", Total: domain.MustParseMoney("30000.00")},
		},
		netWorth: storage.NetWorth{
			Assets: domain.MustParseMoney("500000.00"),
			Debts:  domain.MustParseMoney("100000.00"),
			Net:    domain.MustParseMoney("400000.00"),
		},
		goals: []storage.GoalProgress{
			{ID: 1, Name: "Подушка", Target: domain.MustParseMoney("300000.00"),
				Current:  domain.MustParseMoney("150000.00"),
				Progress: domain.FromDecimal(decimal.NewFromFloat(0.5))},
		},
	}
	svc := newSvcWithFixedNow(repo, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))

	summary, err := svc.Compute(context.Background(), 7, PeriodMonth, CustomRange{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !summary.Net.Equal(domain.MustParseMoney("40000.00")) {
		t.Errorf("net = %s, want 40000.00", summary.Net)
	}
	if len(summary.ByCategory) != 1 {
		t.Errorf("by_category len = %d, want 1", len(summary.ByCategory))
	}
	if len(summary.Goals) != 1 || summary.Goals[0].Name != "Подушка" {
		t.Errorf("goals = %+v", summary.Goals)
	}
	if !summary.NetWorth.Net.Equal(domain.MustParseMoney("400000.00")) {
		t.Errorf("net_worth.net = %s, want 400000.00", summary.NetWorth.Net)
	}
	// Bounds forwarded to the repo match the resolved month.
	if repo.lastFrom.Month() != time.June || repo.lastFrom.Year() != 2026 {
		t.Errorf("repo got from = %v, want June 2026", repo.lastFrom)
	}
}

func TestCompute_NormalizesNilSlices(t *testing.T) {
	repo := &fakeRepo{} // all zero → nil slices
	svc := newSvcWithFixedNow(repo, time.Now())

	summary, err := svc.Compute(context.Background(), 7, PeriodMonth, CustomRange{})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	// JSON serialization would turn nil into null; we want [] for a stable API.
	if summary.ByCategory == nil {
		t.Errorf("ByCategory should be non-nil empty slice, got nil")
	}
	if summary.Goals == nil {
		t.Errorf("Goals should be non-nil empty slice, got nil")
	}
	if len(summary.ByCategory) != 0 || len(summary.Goals) != 0 {
		t.Errorf("expected empty slices, got cats=%d goals=%d", len(summary.ByCategory), len(summary.Goals))
	}
}

func TestCompute_PropagatesStorageError(t *testing.T) {
	repo := &fakeRepo{failWith: errors.New("db down")}
	svc := newSvcWithFixedNow(repo, time.Now())

	_, err := svc.Compute(context.Background(), 7, PeriodMonth, CustomRange{})
	if err == nil {
		t.Fatalf("expected error from storage, got nil")
	}
}
