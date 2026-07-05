// Package goals implements BUSINESS_LOGIC.md ф.5 — savings-goal tracker with
// recurring contributions, ad-hoc top-ups, a status projection, and a
// stateless what-if simulation, all built on the sinking-fund formulas in
// internal/mathcore/goals (Копнова Гл. 3.3.3).
//
// Layers (symmetric with the budget package):
//   - Repo interface → unit tests substitute an in-memory fakeRepo, production
//     wires *storage.Pool.
//   - The service owns validation, idempotency, and the projection math.
//   - Storage is a thin row reader/writer; it has no business rules.
//
// Hybrid current-amount model (design spec §3.2):
//
//	effective_current = goal.current_amount (baseline) + Σ goal_contributions
//
// Every Projection / Simulate reads effective_current, never current_amount
// alone. INSERT/DELETE on contributions auto-changes projection on the next
// read (self-healing, like accounts.balance via full recompute in ф.1).
//
// Money is decimal end-to-end; the service never touches float64.
package goals

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/RedEye472-afk/FinHelper/internal/mathcore/goals"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// Repo is the storage contract. Satisfied by *storage.Pool; tests substitute
// a fake. Methods mirror storage.Pool 1:1.
type Repo interface {
	CreateGoal(ctx context.Context, g domain.Goal) (domain.Goal, error)
	GetGoal(ctx context.Context, userID, id int64) (domain.Goal, error)
	ListGoals(ctx context.Context, userID int64) ([]domain.Goal, error)
	UpdateGoal(ctx context.Context, g domain.Goal) (domain.Goal, error)
	DeleteGoal(ctx context.Context, userID, id int64) error
	SumContributions(ctx context.Context, userID, goalID int64) (domain.Money, error)
	CreateContribution(ctx context.Context, c domain.GoalContribution) (domain.GoalContribution, error)
	GetContributionByClientID(ctx context.Context, userID, goalID int64, contributionID string) (domain.GoalContribution, error)
	ListContributions(ctx context.Context, userID, goalID int64) ([]domain.GoalContribution, error)
	DeleteContribution(ctx context.Context, userID, goalID, id int64) error
}

// Service is the goals business layer. Construct once at boot, share across
// requests — holds no per-request state.
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
	ErrInvalidArgument = errors.New("goals: invalid argument")
	// ErrNotFound — goal/contribution not found or belongs to another user.
	ErrNotFound = errors.New("goals: not found")
)

// CreateInput is what the caller supplies to create a goal.
type CreateInput struct {
	UserID              int64
	Name                string
	TargetAmount        domain.Money
	CurrentAmount       domain.Money
	MonthlyContribution *domain.Money
	TargetDate          *time.Time
	ExpectedYield       decimal.Decimal
}

// Create validates and persists a goal. CurrentAmount defaults to Zero when
// the caller omits it (UX: «начать с нуля»). ExpectedYield defaults to 0.
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Goal, error) {
	if in.UserID == 0 {
		return domain.Goal{}, errors.New("goals: user_id required")
	}
	if err := domain.ValidateGoal(in.Name, in.TargetAmount, in.ExpectedYield, in.TargetDate, in.MonthlyContribution, s.now()); err != nil {
		return domain.Goal{}, err
	}
	if in.CurrentAmount.IsNegative() {
		return domain.Goal{}, errors.New("goals: current_amount must be >= 0")
	}
	return s.repo.CreateGoal(ctx, domain.Goal{
		UserID:              in.UserID,
		Name:                in.Name,
		TargetAmount:        in.TargetAmount,
		CurrentAmount:       in.CurrentAmount,
		MonthlyContribution: in.MonthlyContribution,
		TargetDate:          in.TargetDate,
		ExpectedYield:       in.ExpectedYield,
	})
}

// Get returns one goal scoped to the owner.
func (s *Service) Get(ctx context.Context, userID, id int64) (domain.Goal, error) {
	g, err := s.repo.GetGoal(ctx, userID, id)
	if err != nil {
		return domain.Goal{}, mapStorageErr(err)
	}
	return g, nil
}

// List returns all goals for the user. nil slice is normalised to empty so
// the JSON response is `{"items":[]}`, not `{"items":null}`.
func (s *Service) List(ctx context.Context, userID int64) ([]domain.Goal, error) {
	gs, err := s.repo.ListGoals(ctx, userID)
	if err != nil {
		return nil, mapStorageErr(err)
	}
	if gs == nil {
		gs = []domain.Goal{}
	}
	return gs, nil
}

// UpdateInput mutates a goal. id/user_id are identity.
type UpdateInput struct {
	UserID              int64
	ID                  int64
	Name                string
	TargetAmount        domain.Money
	CurrentAmount       domain.Money
	MonthlyContribution *domain.Money
	TargetDate          *time.Time
	ExpectedYield       decimal.Decimal
}

// Update validates and applies a mutation. TargetDate and MonthlyContribution
// can be nilled by passing nil — they are always overwritten, not merged.
func (s *Service) Update(ctx context.Context, in UpdateInput) (domain.Goal, error) {
	if in.UserID == 0 || in.ID == 0 {
		return domain.Goal{}, errors.New("goals: user_id and id required")
	}
	if err := domain.ValidateGoal(in.Name, in.TargetAmount, in.ExpectedYield, in.TargetDate, in.MonthlyContribution, s.now()); err != nil {
		return domain.Goal{}, err
	}
	if in.CurrentAmount.IsNegative() {
		return domain.Goal{}, errors.New("goals: current_amount must be >= 0")
	}
	g, err := s.repo.UpdateGoal(ctx, domain.Goal{
		ID:                  in.ID,
		UserID:              in.UserID,
		Name:                in.Name,
		TargetAmount:        in.TargetAmount,
		CurrentAmount:       in.CurrentAmount,
		MonthlyContribution: in.MonthlyContribution,
		TargetDate:          in.TargetDate,
		ExpectedYield:       in.ExpectedYield,
	})
	if err != nil {
		return domain.Goal{}, mapStorageErr(err)
	}
	return g, nil
}

// Delete soft-deletes a goal.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	if err := s.repo.DeleteGoal(ctx, userID, id); err != nil {
		return mapStorageErr(err)
	}
	return nil
}

// AddContributionInput is the body of POST /goals/{id}/contributions.
type AddContributionInput struct {
	UserID         int64
	GoalID         int64
	ContributionID string
	Amount         domain.Money
	Date           time.Time // zero → s.now()
	Comment        string
}

// AddContribution records an ad-hoc top-up with idempotency (style of ф.1):
// on ErrContributionExists it fetches the original row by client-generated
// contribution_id and returns it with duplicate=true so the handler can
// respond 200 (not 409) — a replayed POST must look like the first one.
func (s *Service) AddContribution(ctx context.Context, in AddContributionInput) (domain.GoalContribution, bool, error) {
	if in.UserID == 0 || in.GoalID == 0 {
		return domain.GoalContribution{}, false, errors.New("goals: user_id and goal_id required")
	}
	if in.ContributionID == "" {
		return domain.GoalContribution{}, false, errors.New("goals: contribution_id required")
	}
	if !in.Amount.IsPositive() {
		return domain.GoalContribution{}, false, errors.New("goals: amount must be positive")
	}
	// Verify ownership before insert: a contribution to another user's goal
	// must surface as ErrNotFound, not as a FK violation or a silent success.
	if _, err := s.repo.GetGoal(ctx, in.UserID, in.GoalID); err != nil {
		return domain.GoalContribution{}, false, mapStorageErr(err)
	}
	date := in.Date
	if date.IsZero() {
		date = s.now()
	}
	c, err := s.repo.CreateContribution(ctx, domain.GoalContribution{
		UserID:           in.UserID,
		GoalID:           in.GoalID,
		ContributionID:   in.ContributionID,
		Amount:           in.Amount,
		ContributionDate: date,
		Comment:          in.Comment,
	})
	if err != nil {
		if errors.Is(err, storage.ErrContributionExists) {
			orig, gErr := s.repo.GetContributionByClientID(ctx, in.UserID, in.GoalID, in.ContributionID)
			if gErr != nil {
				return domain.GoalContribution{}, false, gErr
			}
			return orig, true, nil
		}
		return domain.GoalContribution{}, false, err
	}
	return c, false, nil
}

// ListContributions returns the journal for a goal (newest first by date).
func (s *Service) ListContributions(ctx context.Context, userID, goalID int64) ([]domain.GoalContribution, error) {
	cs, err := s.repo.ListContributions(ctx, userID, goalID)
	if err != nil {
		return nil, mapStorageErr(err)
	}
	if cs == nil {
		cs = []domain.GoalContribution{}
	}
	return cs, nil
}

// DeleteContribution removes a top-up by row id. Caller recomputes projection.
func (s *Service) DeleteContribution(ctx context.Context, userID, goalID, id int64) error {
	if err := s.repo.DeleteContribution(ctx, userID, goalID, id); err != nil {
		return mapStorageErr(err)
	}
	return nil
}

// --- Projection & Simulation (Task 11) ---

// Projection is the result of evaluating one goal against time and the
// contribution journal (BUSINESS_LOGIC ф.5 "С").
type Projection struct {
	Goal             domain.Goal     `json:"goal"`
	EffectiveCurrent domain.Money    `json:"effective_current"` // baseline + Σ contributions
	TargetEffective  domain.Money    `json:"target_effective"`  // target, possibly inflated
	Progress         decimal.Decimal `json:"progress"`          // effective / target [0..1+]
	MonthsLeft       int             `json:"months_left"`       // to target_date (0 if none/expired)
	RequiredMonthly  domain.Money    `json:"required_monthly"`  // A to hit target by deadline
	EstimatedMonths  int             `json:"estimated_months"`  // n at current contribution (0 if none)
	Status           domain.GoalStatus `json:"status"`
	AsOfDate         time.Time       `json:"as_of"`
}

// Compute builds the projection for a stored goal. Reads Σ contributions for
// the hybrid effective-current model, then runs the pure projectWith.
func (s *Service) Compute(ctx context.Context, userID, goalID int64) (Projection, error) {
	if userID == 0 || goalID == 0 {
		return Projection{}, errors.New("goals: user_id and id required")
	}
	g, err := s.repo.GetGoal(ctx, userID, goalID)
	if err != nil {
		return Projection{}, mapStorageErr(err)
	}
	sum, err := s.repo.SumContributions(ctx, userID, goalID)
	if err != nil {
		return Projection{}, err
	}
	return s.projectWith(g, sum, s.now(), nil), nil
}

// SimulateInput is the body of POST /calc/goal and /goals/{id}/simulate.
// All fields are optional except TargetAmount (a what-if needs a target to
// aim at). For /simulate against a stored goal, body fields override the
// stored ones; absent fields fall back to the goal's stored value.
type SimulateInput struct {
	CurrentAmount       domain.Money
	TargetAmount        domain.Money
	MonthlyContribution *domain.Money
	TargetDate          *time.Time
	ExpectedYield       decimal.Decimal
	Inflation           decimal.Decimal // optional; 0 = no inflation adjustment
}

// Simulate runs a stateless what-if: no storage read, no persistence. Used by
// POST /calc/goal for prospects who want to play with numbers before signing
// up, and by the "что если" simulator (ф.12 future hook).
func (s *Service) Simulate(ctx context.Context, in SimulateInput) (Projection, error) {
	if !in.TargetAmount.IsPositive() {
		return Projection{}, errors.New("goals: target_amount must be positive")
	}
	now := s.now()
	g := domain.Goal{
		Name: "simulate", TargetAmount: in.TargetAmount, CurrentAmount: in.CurrentAmount,
		MonthlyContribution: in.MonthlyContribution, TargetDate: in.TargetDate,
		ExpectedYield: in.ExpectedYield,
	}
	inflation := in.Inflation
	return s.projectWith(g, domain.Zero, now, &inflation), nil
}

// SimulateSaved combines a stored goal with a what-if body. Body fields
// override the stored goal where they are non-zero/non-nil — this lets the
// user ask "what if I bump my monthly contribution to 60k?" without losing
// the rest of the goal's settings.
func (s *Service) SimulateSaved(ctx context.Context, userID, goalID int64, in SimulateInput) (Projection, error) {
	g, err := s.repo.GetGoal(ctx, userID, goalID)
	if err != nil {
		return Projection{}, mapStorageErr(err)
	}
	sum, err := s.repo.SumContributions(ctx, userID, goalID)
	if err != nil {
		return Projection{}, err
	}
	if in.TargetAmount.IsPositive() {
		g.TargetAmount = in.TargetAmount
	}
	if in.MonthlyContribution != nil {
		g.MonthlyContribution = in.MonthlyContribution
	}
	if in.TargetDate != nil {
		g.TargetDate = in.TargetDate
	}
	if in.CurrentAmount.IsPositive() {
		g.CurrentAmount = in.CurrentAmount
	}
	inflation := in.Inflation
	return s.projectWith(g, sum, s.now(), &inflation), nil
}

// projectWith is a pure (nearly) function over (goal, sum, now, inflation).
// All mathcore calls live here; errors from goals.SolveContribution/Term are
// treated as "behind/unreachable" rather than surfacing to the caller — a
// projection always returns a Status, never an error from the math layer.
func (s *Service) projectWith(g domain.Goal, sum domain.Money, now time.Time, inflation *decimal.Decimal) Projection {
	effective := g.CurrentAmount.Add(sum)
	i := g.ExpectedYield.Div(decimal.NewFromInt(12)) // monthly rate

	// Already reached? Short-circuit.
	if effective.GreaterThanOrEqual(g.TargetAmount) {
		return Projection{
			Goal: g, EffectiveCurrent: effective, TargetEffective: g.TargetAmount,
			Progress: decimal.NewFromInt(1), Status: domain.StatusGoalAchieved, AsOfDate: now,
		}
	}

	// Inflate the target if a deadline and a positive inflation rate are given.
	// inflation is *decimal.Decimal (may be nil from Simulate/SimulateSaved);
	// InflateTarget accepts a value, so dereference after the nil-check.
	targetEff := g.TargetAmount
	if g.TargetDate != nil && inflation != nil && inflation.IsPositive() {
		months := monthsBetween(now, *g.TargetDate)
		if inflated, err := goals.InflateTarget(g.TargetAmount.Decimal(), *inflation, months); err == nil {
			targetEff = domain.FromDecimal(inflated)
		}
	}

	progress := effective.Decimal().Div(targetEff.Decimal())

	// With a deadline: solve for the required monthly contribution.
	if g.TargetDate != nil {
		monthsLeft := monthsBetween(now, *g.TargetDate)
		if monthsLeft <= 0 {
			return Projection{
				Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
				Progress: progress, MonthsLeft: 0, Status: domain.StatusGoalBehind, AsOfDate: now,
			}
		}
		req, err := goals.SolveContribution(effective.Decimal(), targetEff.Decimal(), i, monthsLeft)
		if err != nil {
			// Solver refused (e.g. effective already > target via growth, or periods edge).
			// Surface as behind; the user can edit the goal.
			return Projection{
				Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
				Progress: progress, MonthsLeft: monthsLeft, Status: domain.StatusGoalBehind, AsOfDate: now,
			}
		}
		reqMoney := domain.FromDecimal(req)
		status := classifyStatus(g.MonthlyContribution, reqMoney)
		return Projection{
			Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
			Progress: progress, MonthsLeft: monthsLeft, RequiredMonthly: reqMoney,
			Status: status, AsOfDate: now,
		}
	}

	// No deadline, but a recurring contribution: solve for the term.
	if g.MonthlyContribution != nil && g.MonthlyContribution.IsPositive() {
		n, err := goals.SolveTerm(effective.Decimal(), targetEff.Decimal(), g.MonthlyContribution.Decimal(), i)
		if err != nil {
			return Projection{
				Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
				Progress: progress, Status: domain.StatusGoalBehind, AsOfDate: now,
			}
		}
		// Round up to whole periods: 18.3 months → 19 (you need the 19th to actually hit target).
		estMonths := int(n.Round(0).IntPart())
		if n.Sub(decimal.NewFromInt(int64(estMonths))).IsPositive() {
			estMonths++
		}
		return Projection{
			Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
			Progress: progress, EstimatedMonths: estMonths,
			Status: domain.StatusGoalOnTrack, AsOfDate: now,
		}
	}

	// No deadline, no contribution: nothing to project.
	return Projection{
		Goal: g, EffectiveCurrent: effective, TargetEffective: targetEff,
		Progress: progress, Status: domain.StatusGoalNoDeadline, AsOfDate: now,
	}
}

// classifyStatus maps the user's pledged monthly contribution against the
// required one to a UI label.
//   - nil/0 contribution with a 0 required → on_track (target reachable by
//     capital growth alone);
//   - nil/0 contribution with required > 0 → behind;
//   - pledge ≥ required → on_track;
//   - pledge ≥ 0.9 × required → at_risk (close, but a single missed payment
//     tips it over);
//   - otherwise → behind.
func classifyStatus(monthly *domain.Money, required domain.Money) domain.GoalStatus {
	if monthly == nil || !monthly.IsPositive() {
		if required.IsZero() {
			return domain.StatusGoalOnTrack
		}
		return domain.StatusGoalBehind
	}
	if monthly.GreaterThanOrEqual(required) {
		return domain.StatusGoalOnTrack
	}
	threshold := required.Mul(decimal.NewFromFloat(0.9))
	if monthly.GreaterThanOrEqual(threshold) {
		return domain.StatusGoalAtRisk
	}
	return domain.StatusGoalBehind
}

// monthsBetween is the count of whole calendar months from `from` to `to`,
// clamped to >= 0. Used for both deadline distance and the inflation horizon.
// Computed by month arithmetic on year/month, then nudged down by one if the
// day-of-month hasn't yet been reached (so 31 Jan → 28 Feb is 0, not 1).
func monthsBetween(from, to time.Time) int {
	if !to.After(from) {
		return 0
	}
	months := (to.Year()-from.Year())*12 + int(to.Month()-from.Month())
	if to.Day() < from.Day() {
		// The anniversary of `from` hasn't occurred yet in the final month.
		months--
	}
	if months < 0 {
		months = 0
	}
	return months
}

// mapStorageErr translates storage sentinels into service sentinels so the
// HTTP layer can switch on a single taxonomy. Unknown errors pass through.
func mapStorageErr(err error) error {
	switch {
	case errors.Is(err, storage.ErrGoalNotFound):
		return ErrNotFound
	case errors.Is(err, storage.ErrContributionNotFound):
		return ErrNotFound
	default:
		return err
	}
}

// clientKey is the idempotency key for contributions, used by the fakeRepo in
// tests. Kept here (not in storage) because it's a service-level concept and
// shared with the test fake. Format: "userID:goalID:contributionID".
func clientKey(userID, goalID int64, contributionID string) string {
	return strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(goalID, 10) + ":" + contributionID
}
