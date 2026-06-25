package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Goal is a user's savings target (BUSINESS_LOGIC.md ф.5). Mirrors the goals
// table from migration 0001. Money fields are always domain.Money (decimal,
// scale=2); ExpectedYield is a plain decimal (annual rate, e.g. 0.08).
type Goal struct {
	ID                   int64
	UserID               int64
	Name                 string
	TargetAmount         Money           // сумма, к которой идём
	CurrentAmount        Money           // baseline «уже накоплено до старта трекинга»
	MonthlyContribution  *Money          // nil = регулярного взноса нет
	TargetDate           *time.Time      // nil = без дедлайна
	ExpectedYield        decimal.Decimal // годовая ставка, напр. 0.08
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// GoalContribution is one ad-hoc top-up of a goal (BUSINESS_LOGIC ф.5).
// Idempotency is by (user_id, goal_id, contribution_id) — analogous to calc_id
// in operations (ф.1).
type GoalContribution struct {
	ID               int64
	UserID           int64
	GoalID           int64
	ContributionID   string // client-generated
	Amount           Money
	ContributionDate time.Time
	Comment          string
	CreatedAt        time.Time
}

// GoalStatus is the health label for UI and Projection.
type GoalStatus string

const (
	// StatusGoalOnTrack — взнос/срок достаточны.
	StatusGoalOnTrack GoalStatus = "on_track"
	// StatusGoalAtRisk — на грани (взнос >= 90% требуемого, но ниже него).
	StatusGoalAtRisk GoalStatus = "at_risk"
	// StatusGoalBehind — не успеем накопить к дедлайну (взнос < требуемого или цель убегает).
	StatusGoalBehind GoalStatus = "behind"
	// StatusGoalAchieved — effective current >= target.
	StatusGoalAchieved GoalStatus = "achieved"
	// StatusGoalNoDeadline — target_date не задан и нет monthly_contribution.
	StatusGoalNoDeadline GoalStatus = "no_deadline"
)

// ValidateGoal checks goal invariants before persistence. Returns errors for:
// empty name, non-positive target, negative yield, target_date in the past,
// negative monthly_contribution.
func ValidateGoal(name string, target Money, yield decimal.Decimal, targetDate *time.Time, monthly *Money, now time.Time) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("goal: name required")
	}
	if !target.IsPositive() {
		return errors.New("goal: target_amount must be positive")
	}
	if yield.IsNegative() {
		return errors.New("goal: expected_yield must be >= 0")
	}
	if targetDate != nil && targetDate.Before(now) {
		return errors.New("goal: target_date must be in the future")
	}
	if monthly != nil && monthly.IsNegative() {
		return errors.New("goal: monthly_contribution must be >= 0")
	}
	return nil
}
