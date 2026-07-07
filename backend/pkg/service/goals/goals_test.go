package goals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/pkg/storage"
)

// fakeRepo is an in-memory goals.Repo. Goals are keyed by id; contributions
// are keyed by clientKey(userID, goalID, contributionID) for idempotency.
type fakeRepo struct {
	goals         map[int64]domain.Goal
	contributions map[string]domain.GoalContribution
	nextGoalID    int64
	nextContribID int64
	failWith      error
	// Hooks for narrow fault injection (set per-test).
	getGoalFail            error
	sumContributionsFail   error
	createContributionFail error
	// Counters let tests assert that Simulate never touched the repo.
	getGoalCalls int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		goals:         make(map[int64]domain.Goal),
		contributions: make(map[string]domain.GoalContribution),
	}
}

func (f *fakeRepo) CreateGoal(_ context.Context, g domain.Goal) (domain.Goal, error) {
	if f.failWith != nil {
		return domain.Goal{}, f.failWith
	}
	f.nextGoalID++
	g.ID = f.nextGoalID
	g.CreatedAt = time.Now()
	g.UpdatedAt = g.CreatedAt
	f.goals[g.ID] = g
	return g, nil
}

func (f *fakeRepo) GetGoal(_ context.Context, userID, id int64) (domain.Goal, error) {
	f.getGoalCalls++
	if f.getGoalFail != nil {
		return domain.Goal{}, f.getGoalFail
	}
	g, ok := f.goals[id]
	if !ok || g.UserID != userID {
		return domain.Goal{}, storage.ErrGoalNotFound
	}
	return g, nil
}

func (f *fakeRepo) ListGoals(_ context.Context, userID int64) ([]domain.Goal, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	var out []domain.Goal
	for _, g := range f.goals {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpdateGoal(_ context.Context, g domain.Goal) (domain.Goal, error) {
	if f.failWith != nil {
		return domain.Goal{}, f.failWith
	}
	existing, ok := f.goals[g.ID]
	if !ok || existing.UserID != g.UserID {
		return domain.Goal{}, storage.ErrGoalNotFound
	}
	g.CreatedAt = existing.CreatedAt
	g.UpdatedAt = time.Now()
	f.goals[g.ID] = g
	return g, nil
}

func (f *fakeRepo) DeleteGoal(_ context.Context, userID, id int64) error {
	if f.failWith != nil {
		return f.failWith
	}
	g, ok := f.goals[id]
	if !ok || g.UserID != userID {
		return storage.ErrGoalNotFound
	}
	delete(f.goals, id)
	return nil
}

func (f *fakeRepo) SumContributions(_ context.Context, userID, goalID int64) (domain.Money, error) {
	if f.sumContributionsFail != nil {
		return domain.Zero, f.sumContributionsFail
	}
	sum := domain.Zero
	for _, c := range f.contributions {
		if c.UserID == userID && c.GoalID == goalID {
			sum = sum.Add(c.Amount)
		}
	}
	return sum, nil
}

func (f *fakeRepo) CreateContribution(_ context.Context, c domain.GoalContribution) (domain.GoalContribution, error) {
	if f.createContributionFail != nil {
		return domain.GoalContribution{}, f.createContributionFail
	}
	k := clientKey(c.UserID, c.GoalID, c.ContributionID)
	if _, exists := f.contributions[k]; exists {
		return domain.GoalContribution{}, storage.ErrContributionExists
	}
	f.nextContribID++
	c.ID = f.nextContribID
	c.CreatedAt = time.Now()
	f.contributions[k] = c
	return c, nil
}

func (f *fakeRepo) GetContributionByClientID(_ context.Context, userID, goalID int64, contributionID string) (domain.GoalContribution, error) {
	k := clientKey(userID, goalID, contributionID)
	c, ok := f.contributions[k]
	if !ok {
		return domain.GoalContribution{}, storage.ErrContributionNotFound
	}
	return c, nil
}

func (f *fakeRepo) ListContributions(_ context.Context, userID, goalID int64) ([]domain.GoalContribution, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	var out []domain.GoalContribution
	for _, c := range f.contributions {
		if c.UserID == userID && c.GoalID == goalID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) DeleteContribution(_ context.Context, userID, goalID, id int64) error {
	if f.failWith != nil {
		return f.failWith
	}
	for k, c := range f.contributions {
		if c.UserID == userID && c.GoalID == goalID && c.ID == id {
			delete(f.contributions, k)
			return nil
		}
	}
	return storage.ErrContributionNotFound
}

// svcWithFixedNow pins the clock so month arithmetic and target_date
// validation are deterministic. Mirrors budget_test.go's helper.
func svcWithFixedNow(repo Repo, t time.Time) *Service {
	return &Service{repo: repo, now: func() time.Time { return t }}
}

// fixedNow is the canonical "now" for projection tests. Chosen so a
// target_date one year out yields 12 months via monthsBetween.
var fixedNow = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

// ptrTime and ptrMoney are helpers for the *time.Time / *Money optionals.
func ptrTime(t time.Time) *time.Time { return &t }
func ptrMoney(m domain.Money) *domain.Money { return &m }

// validCreateInput is a baseline that passes all validation rules.
func validCreateInput() CreateInput {
	return CreateInput{
		UserID:        7,
		Name:          "Emergency fund",
		TargetAmount:  domain.MustParseMoney("100000.00"),
		CurrentAmount: domain.Zero,
		ExpectedYield: decimal.NewFromFloat(0.08),
	}
}

// ----------------------------------------------------------------------------
// Create
// ----------------------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, err := svc.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.ID == 0 {
		t.Errorf("expected assigned id, got 0")
	}
	if g.Name != "Emergency fund" {
		t.Errorf("name = %q", g.Name)
	}
	if !g.TargetAmount.Equal(domain.MustParseMoney("100000.00")) {
		t.Errorf("target = %s", g.TargetAmount)
	}
}

func TestCreate_UserIDRequired(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	in := validCreateInput()
	in.UserID = 0
	_, err := svc.Create(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error for missing user_id")
	}
}

func TestCreate_EmptyNameRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	in := validCreateInput()
	in.Name = "   "
	_, err := svc.Create(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error for empty name")
	}
}

func TestCreate_ZeroTargetRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	in := validCreateInput()
	in.TargetAmount = domain.Zero
	_, err := svc.Create(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error for zero target")
	}
}

func TestCreate_NegativeYieldRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	in := validCreateInput()
	in.ExpectedYield = decimal.NewFromFloat(-0.05)
	_, err := svc.Create(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error for negative yield")
	}
}

func TestCreate_PastTargetDateRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	in := validCreateInput()
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	in.TargetDate = &past
	_, err := svc.Create(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error for past target_date")
	}
}

func TestCreate_NegativeCurrentAmountRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	in := validCreateInput()
	in.CurrentAmount = domain.MustParseMoney("-100.00")
	_, err := svc.Create(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error for negative current_amount")
	}
}

func TestCreate_FutureTargetDateAccepted(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	in := validCreateInput()
	in.TargetDate = ptrTime(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("Create with future date: %v", err)
	}
}

// ----------------------------------------------------------------------------
// Get
// ----------------------------------------------------------------------------

func TestGet_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	got, err := svc.Get(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != g.ID {
		t.Errorf("id = %d, want %d", got.ID, g.ID)
	}
}

func TestGet_NotFoundMapped(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, err := svc.Get(context.Background(), 7, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// List
// ----------------------------------------------------------------------------

func TestList_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, _ = svc.Create(context.Background(), validCreateInput())
	_, _ = svc.Create(context.Background(), validCreateInput())
	gs, err := svc.List(context.Background(), 7)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(gs) != 2 {
		t.Errorf("len = %d, want 2", len(gs))
	}
}

func TestList_EmptyNormalised(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	gs, err := svc.List(context.Background(), 7)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gs == nil {
		t.Fatalf("expected non-nil empty slice, got nil")
	}
	if len(gs) != 0 {
		t.Errorf("len = %d, want 0", len(gs))
	}
}

// ----------------------------------------------------------------------------
// Update
// ----------------------------------------------------------------------------

func TestUpdate_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	updated, err := svc.Update(context.Background(), UpdateInput{
		UserID: g.UserID, ID: g.ID,
		Name:           "New name",
		TargetAmount:   domain.MustParseMoney("200000.00"),
		CurrentAmount:  domain.MustParseMoney("50000.00"),
		ExpectedYield:  decimal.NewFromFloat(0.05),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New name" {
		t.Errorf("name = %q", updated.Name)
	}
	if !updated.TargetAmount.Equal(domain.MustParseMoney("200000.00")) {
		t.Errorf("target = %s", updated.TargetAmount)
	}
}

func TestUpdate_IdentityRequired(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, err := svc.Update(context.Background(), UpdateInput{
		UserID: 7, ID: 0, Name: "x", TargetAmount: domain.MustParseMoney("1.00"),
	})
	if err == nil {
		t.Fatalf("expected error for missing id")
	}
}

func TestUpdate_ValidateFailure(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	_, err := svc.Update(context.Background(), UpdateInput{
		UserID: g.UserID, ID: g.ID, Name: "", TargetAmount: domain.MustParseMoney("1.00"),
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, err := svc.Update(context.Background(), UpdateInput{
		UserID: 7, ID: 999, Name: "x", TargetAmount: domain.MustParseMoney("1.00"),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// Delete
// ----------------------------------------------------------------------------

func TestDelete_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	if err := svc.Delete(context.Background(), g.UserID, g.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), g.UserID, g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, Get err = %v, want ErrNotFound", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	err := svc.Delete(context.Background(), 7, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// AddContribution
// ----------------------------------------------------------------------------

func TestAddContribution_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	c, dup, err := svc.AddContribution(context.Background(), AddContributionInput{
		UserID: g.UserID, GoalID: g.ID, ContributionID: "c-1",
		Amount: domain.MustParseMoney("5000.00"),
	})
	if err != nil {
		t.Fatalf("AddContribution: %v", err)
	}
	if dup {
		t.Errorf("dup = true, want false on first insert")
	}
	if c.Amount.Equal(domain.MustParseMoney("5000.00")) == false {
		t.Errorf("amount = %s", c.Amount)
	}
}

func TestAddContribution_IdentityRequired(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	cases := []struct {
		name string
		in   AddContributionInput
	}{
		{"missing user_id", AddContributionInput{GoalID: 1, ContributionID: "c", Amount: domain.MustParseMoney("1.00")}},
		{"missing goal_id", AddContributionInput{UserID: 1, ContributionID: "c", Amount: domain.MustParseMoney("1.00")}},
		{"missing contribution_id", AddContributionInput{UserID: 1, GoalID: 1, Amount: domain.MustParseMoney("1.00")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := svc.AddContribution(context.Background(), c.in)
			if err == nil {
				t.Errorf("expected error")
			}
		})
	}
}

func TestAddContribution_NonPositiveAmount(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	_, _, err := svc.AddContribution(context.Background(), AddContributionInput{
		UserID: g.UserID, GoalID: g.ID, ContributionID: "c", Amount: domain.Zero,
	})
	if err == nil {
		t.Fatalf("expected error for zero amount")
	}
}

func TestAddContribution_OwnershipCheck(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, _, err := svc.AddContribution(context.Background(), AddContributionInput{
		UserID: 7, GoalID: 999, ContributionID: "c", Amount: domain.MustParseMoney("1.00"),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing goal, got %v", err)
	}
}

func TestAddContribution_IdempotentReplay(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	in := AddContributionInput{
		UserID: g.UserID, GoalID: g.ID, ContributionID: "dup-1",
		Amount: domain.MustParseMoney("5000.00"),
	}
	first, dupFirst, err := svc.AddContribution(context.Background(), in)
	if err != nil {
		t.Fatalf("first AddContribution: %v", err)
	}
	if dupFirst {
		t.Fatalf("first call: dup = true, want false")
	}

	replay, dupReplay, err := svc.AddContribution(context.Background(), in)
	if err != nil {
		t.Fatalf("replay AddContribution: %v", err)
	}
	if !dupReplay {
		t.Errorf("replay: dup = false, want true")
	}
	if replay.ID != first.ID {
		t.Errorf("replay id = %d, want %d (same row)", replay.ID, first.ID)
	}
}

// ----------------------------------------------------------------------------
// ListContributions
// ----------------------------------------------------------------------------

func TestListContributions_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	_, _, _ = svc.AddContribution(context.Background(), AddContributionInput{
		UserID: g.UserID, GoalID: g.ID, ContributionID: "a", Amount: domain.MustParseMoney("1.00"),
	})
	_, _, _ = svc.AddContribution(context.Background(), AddContributionInput{
		UserID: g.UserID, GoalID: g.ID, ContributionID: "b", Amount: domain.MustParseMoney("2.00"),
	})
	cs, err := svc.ListContributions(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("ListContributions: %v", err)
	}
	if len(cs) != 2 {
		t.Errorf("len = %d, want 2", len(cs))
	}
}

func TestListContributions_EmptyNormalised(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	cs, err := svc.ListContributions(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("ListContributions: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected non-nil empty slice, got nil")
	}
	if len(cs) != 0 {
		t.Errorf("len = %d, want 0", len(cs))
	}
}

// ----------------------------------------------------------------------------
// DeleteContribution
// ----------------------------------------------------------------------------

func TestDeleteContribution_Success(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	c, _, _ := svc.AddContribution(context.Background(), AddContributionInput{
		UserID: g.UserID, GoalID: g.ID, ContributionID: "c", Amount: domain.MustParseMoney("1.00"),
	})
	if err := svc.DeleteContribution(context.Background(), g.UserID, g.ID, c.ID); err != nil {
		t.Fatalf("DeleteContribution: %v", err)
	}
}

func TestDeleteContribution_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), validCreateInput())
	err := svc.DeleteContribution(context.Background(), g.UserID, g.ID, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// Compute — achieved short-circuit & deadline paths
// ----------------------------------------------------------------------------

func TestCompute_AchievedShortCircuit(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "done",
		TargetAmount:  domain.MustParseMoney("1000.00"),
		CurrentAmount: domain.MustParseMoney("1200.00"),
	})
	proj, err := svc.Compute(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.Status != domain.StatusGoalAchieved {
		t.Errorf("status = %s, want achieved", proj.Status)
	}
	if !proj.EffectiveCurrent.GreaterThanOrEqual(proj.TargetEffective) {
		t.Errorf("effective %s < target %s", proj.EffectiveCurrent, proj.TargetEffective)
	}
}

func TestCompute_AchievedViaContributions(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	// current 0, target 1000, contribution 1000 → effective 1000 ≥ target.
	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "via contrib",
		TargetAmount: domain.MustParseMoney("1000.00"),
	})
	_, _, _ = svc.AddContribution(context.Background(), AddContributionInput{
		UserID: g.UserID, GoalID: g.ID, ContributionID: "full",
		Amount: domain.MustParseMoney("1000.00"),
	})
	proj, err := svc.Compute(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.Status != domain.StatusGoalAchieved {
		t.Errorf("status = %s, want achieved", proj.Status)
	}
}

func TestCompute_WithDeadline_NoInflation(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	due := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC) // 12 months out
	// Compute passes nil inflation, so the target is NOT inflated; this test
	// verifies the deadline+SolveContribution path runs and classifies status.
	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "deadline",
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.MustParseMoney("10000.00"),
		MonthlyContribution: ptrMoney(domain.MustParseMoney("8000.00")),
		TargetDate:          &due,
		ExpectedYield:       decimal.NewFromFloat(0.05),
	})
	proj, err := svc.Compute(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.MonthsLeft != 12 {
		t.Errorf("months_left = %d, want 12", proj.MonthsLeft)
	}
	if !proj.TargetEffective.Equal(domain.MustParseMoney("100000.00")) {
		t.Errorf("target_effective = %s, want 100000 (no inflation in Compute)", proj.TargetEffective)
	}
	if proj.Status == domain.StatusGoalAchieved || proj.Status == domain.StatusGoalNoDeadline {
		t.Errorf("status = %s, unexpected for deadline path", proj.Status)
	}
}

func TestCompute_WithDeadline_OnTrack(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	due := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC) // 12 months
	// No inflation → target stays 100000. Pledge a large monthly so that
	// pledge >= required_monthly → on_track.
	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "on track",
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.MustParseMoney("50000.00"),
		MonthlyContribution: ptrMoney(domain.MustParseMoney("10000.00")),
		TargetDate:          &due,
		ExpectedYield:       decimal.NewFromFloat(0.08),
	})
	proj, err := svc.Compute(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.Status != domain.StatusGoalOnTrack {
		t.Errorf("status = %s, want on_track (required=%s)", proj.Status, proj.RequiredMonthly)
	}
	if !proj.RequiredMonthly.IsPositive() {
		t.Errorf("required_monthly should be positive for deadline path")
	}
}

func TestCompute_WithDeadline_AtRisk(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	due := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC) // 12 months
	// Compute required, then pledge exactly 90% of it by first reading the
	// required monthly and adjusting. Since we can't precompute SolveContribution
	// here, we use a high-yield + near-zero current setup and assert the status
	// is one of the deadline-aware statuses; the at_risk threshold (90%) is
	// covered explicitly in TestClassifyStatus_AtRisk below.
	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "at risk candidate",
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.Zero,
		MonthlyContribution: ptrMoney(domain.MustParseMoney("1.00")),
		TargetDate:          &due,
		ExpectedYield:       decimal.NewFromFloat(0.0),
	})
	proj, err := svc.Compute(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.Status != domain.StatusGoalBehind {
		t.Errorf("status = %s, want behind (pledge 1 << required %s)", proj.Status, proj.RequiredMonthly)
	}
}

func TestCompute_ExpiredDeadline_Behind(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	// Bypass the future-date validator by writing straight to the fake.
	past := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo.nextGoalID++
	gid := repo.nextGoalID
	repo.goals[gid] = domain.Goal{
		ID: gid, UserID: 7, Name: "expired",
		TargetAmount:  domain.MustParseMoney("100000.00"),
		CurrentAmount: domain.MustParseMoney("1000.00"),
		TargetDate:    &past,
	}
	proj, err := svc.Compute(context.Background(), 7, gid)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.MonthsLeft != 0 {
		t.Errorf("months_left = %d, want 0 (expired)", proj.MonthsLeft)
	}
	if proj.Status != domain.StatusGoalBehind {
		t.Errorf("status = %s, want behind (expired)", proj.Status)
	}
}

func TestCompute_NoDeadline_WithMonthly_OnTrack(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "no deadline, monthly",
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.MustParseMoney("10000.00"),
		MonthlyContribution: ptrMoney(domain.MustParseMoney("8000.00")),
		ExpectedYield:       decimal.NewFromFloat(0.05),
	})
	proj, err := svc.Compute(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.EstimatedMonths <= 0 {
		t.Errorf("estimated_months = %d, want > 0", proj.EstimatedMonths)
	}
	if proj.Status != domain.StatusGoalOnTrack {
		t.Errorf("status = %s, want on_track", proj.Status)
	}
}

func TestCompute_NoDeadline_NoMonthly_NoDeadline(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "bare goal",
		TargetAmount:  domain.MustParseMoney("100000.00"),
		CurrentAmount: domain.MustParseMoney("10000.00"),
	})
	proj, err := svc.Compute(context.Background(), g.UserID, g.ID)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if proj.Status != domain.StatusGoalNoDeadline {
		t.Errorf("status = %s, want no_deadline", proj.Status)
	}
}

func TestCompute_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, err := svc.Compute(context.Background(), 7, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCompute_IdentityRequired(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, err := svc.Compute(context.Background(), 0, 0)
	if err == nil {
		t.Fatalf("expected error for missing identity")
	}
}

// ----------------------------------------------------------------------------
// Simulate (stateless what-if)
// ----------------------------------------------------------------------------

func TestSimulate_ZeroTargetRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, err := svc.Simulate(context.Background(), SimulateInput{
		TargetAmount: domain.Zero,
	})
	if err == nil {
		t.Fatalf("expected error for zero target")
	}
}

func TestSimulate_NoStorageRead(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	before := repo.getGoalCalls
	_, err := svc.Simulate(context.Background(), SimulateInput{
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.MustParseMoney("10000.00"),
		MonthlyContribution: ptrMoney(domain.MustParseMoney("8000.00")),
		ExpectedYield:       decimal.NewFromFloat(0.05),
	})
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	if repo.getGoalCalls != before {
		t.Errorf("Simulate touched repo: getGoalCalls delta = %d, want 0", repo.getGoalCalls-before)
	}
}

func TestSimulate_NoDeadlineNoMonthly_NoDeadline(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	proj, err := svc.Simulate(context.Background(), SimulateInput{
		TargetAmount:  domain.MustParseMoney("100000.00"),
		CurrentAmount: domain.MustParseMoney("10000.00"),
	})
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	if proj.Status != domain.StatusGoalNoDeadline {
		t.Errorf("status = %s, want no_deadline", proj.Status)
	}
}

// ----------------------------------------------------------------------------
// SimulateSaved (stored goal + what-if overrides)
// ----------------------------------------------------------------------------

func TestSimulateSaved_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	_, err := svc.SimulateSaved(context.Background(), 7, 999, SimulateInput{
		TargetAmount: domain.MustParseMoney("1.00"),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSimulateSaved_OverridesTarget(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "base",
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.MustParseMoney("10000.00"),
		MonthlyContribution: ptrMoney(domain.MustParseMoney("5000.00")),
		ExpectedYield:       decimal.NewFromFloat(0.05),
	})
	proj, err := svc.SimulateSaved(context.Background(), g.UserID, g.ID, SimulateInput{
		TargetAmount: domain.MustParseMoney("500000.00"),
	})
	if err != nil {
		t.Fatalf("SimulateSaved: %v", err)
	}
	if !proj.Goal.TargetAmount.Equal(domain.MustParseMoney("500000.00")) {
		t.Errorf("target = %s, want 500000 (overridden)", proj.Goal.TargetAmount)
	}
}

func TestSimulateSaved_OverridesMonthly(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "base",
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.MustParseMoney("10000.00"),
		MonthlyContribution: ptrMoney(domain.MustParseMoney("5000.00")),
		ExpectedYield:       decimal.NewFromFloat(0.05),
	})
	proj, err := svc.SimulateSaved(context.Background(), g.UserID, g.ID, SimulateInput{
		MonthlyContribution: ptrMoney(domain.MustParseMoney("9000.00")),
	})
	if err != nil {
		t.Fatalf("SimulateSaved: %v", err)
	}
	if proj.Goal.MonthlyContribution == nil ||
		!proj.Goal.MonthlyContribution.Equal(domain.MustParseMoney("9000.00")) {
		t.Errorf("monthly override not applied: %+v", proj.Goal.MonthlyContribution)
	}
}

func TestSimulateSaved_OverridesTargetDate(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	due := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "base",
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.MustParseMoney("10000.00"),
		MonthlyContribution: ptrMoney(domain.MustParseMoney("5000.00")),
		ExpectedYield:       decimal.NewFromFloat(0.05),
	})
	newDue := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)
	proj, err := svc.SimulateSaved(context.Background(), g.UserID, g.ID, SimulateInput{
		TargetDate: &newDue,
	})
	if err != nil {
		t.Fatalf("SimulateSaved: %v", err)
	}
	if proj.Goal.TargetDate == nil || !proj.Goal.TargetDate.Equal(newDue) {
		t.Errorf("target_date override not applied")
	}
	_ = due
}

func TestSimulateSaved_InflationOverride(t *testing.T) {
	repo := newFakeRepo()
	svc := svcWithFixedNow(repo, fixedNow)

	due := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC) // 12 months
	g, _ := svc.Create(context.Background(), CreateInput{
		UserID: 7, Name: "inflated",
		TargetAmount:        domain.MustParseMoney("100000.00"),
		CurrentAmount:       domain.MustParseMoney("10000.00"),
		MonthlyContribution: ptrMoney(domain.MustParseMoney("8000.00")),
		TargetDate:          &due,
		ExpectedYield:       decimal.NewFromFloat(0.05),
	})
	proj, err := svc.SimulateSaved(context.Background(), g.UserID, g.ID, SimulateInput{
		Inflation: decimal.NewFromFloat(0.10),
	})
	if err != nil {
		t.Fatalf("SimulateSaved: %v", err)
	}
	if !proj.TargetEffective.GreaterThan(domain.MustParseMoney("100000.00")) {
		t.Errorf("target_effective = %s, want > 100000 (inflated at 10%%)", proj.TargetEffective)
	}
}

// ----------------------------------------------------------------------------
// classifyStatus (pure)
// ----------------------------------------------------------------------------

func TestClassifyStatus_NilAndZeroRequired_OnTrack(t *testing.T) {
	if got := classifyStatus(nil, domain.Zero); got != domain.StatusGoalOnTrack {
		t.Errorf("nil + 0 required = %s, want on_track", got)
	}
	zero := domain.Zero
	if got := classifyStatus(&zero, domain.Zero); got != domain.StatusGoalOnTrack {
		t.Errorf("0 pledge + 0 required = %s, want on_track", got)
	}
}

func TestClassifyStatus_NilAndPositiveRequired_Behind(t *testing.T) {
	if got := classifyStatus(nil, domain.MustParseMoney("100.00")); got != domain.StatusGoalBehind {
		t.Errorf("nil + positive required = %s, want behind", got)
	}
}

func TestClassifyStatus_PledgeMeetsRequired_OnTrack(t *testing.T) {
	pledge := domain.MustParseMoney("1000.00")
	if got := classifyStatus(&pledge, domain.MustParseMoney("1000.00")); got != domain.StatusGoalOnTrack {
		t.Errorf("pledge == required = %s, want on_track", got)
	}
	if got := classifyStatus(&pledge, domain.MustParseMoney("900.00")); got != domain.StatusGoalOnTrack {
		t.Errorf("pledge > required = %s, want on_track", got)
	}
}

func TestClassifyStatus_AtRisk(t *testing.T) {
	// pledge = 95, required = 100 → 0.95 ≥ 0.9 → at_risk.
	pledge := domain.MustParseMoney("95.00")
	if got := classifyStatus(&pledge, domain.MustParseMoney("100.00")); got != domain.StatusGoalAtRisk {
		t.Errorf("pledge 0.95×required = %s, want at_risk", got)
	}
}

func TestClassifyStatus_BelowThreshold_Behind(t *testing.T) {
	// pledge = 80, required = 100 → 0.8 < 0.9 → behind.
	pledge := domain.MustParseMoney("80.00")
	if got := classifyStatus(&pledge, domain.MustParseMoney("100.00")); got != domain.StatusGoalBehind {
		t.Errorf("pledge 0.8×required = %s, want behind", got)
	}
}

// ----------------------------------------------------------------------------
// monthsBetween (pure)
// ----------------------------------------------------------------------------

func TestMonthsBetween_SameMonthZero(t *testing.T) {
	a := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	b := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	if got := monthsBetween(a, b); got != 0 {
		t.Errorf("same month = %d, want 0", got)
	}
}

func TestMonthsBetween_Jan31ToFeb28_Zero(t *testing.T) {
	from := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	if got := monthsBetween(from, to); got != 0 {
		t.Errorf("Jan 31 → Feb 28 = %d, want 0 (day not reached)", got)
	}
}

func TestMonthsBetween_Jan1ToMar1_Two(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if got := monthsBetween(from, to); got != 2 {
		t.Errorf("Jan 1 → Mar 1 = %d, want 2", got)
	}
}

func TestMonthsBetween_ReversedZero(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := monthsBetween(from, to); got != 0 {
		t.Errorf("reversed = %d, want 0", got)
	}
}

func TestMonthsBetween_OneYearTwelve(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	if got := monthsBetween(from, to); got != 12 {
		t.Errorf("one year = %d, want 12", got)
	}
}
