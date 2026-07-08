package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
)

// Sentinel errors for goals + contributions. Callers branch on errors.Is.
var (
	// ErrGoalNotFound — no goal matches (user_id, id) or the row is soft-deleted.
	ErrGoalNotFound = errors.New("storage: goal not found")
	// ErrContributionExists — (user_id, goal_id, contribution_id) already has a row.
	// Idempotency sentinel, style of operations.ErrOperationExists (ф.1).
	ErrContributionExists = errors.New("storage: contribution already exists")
	// ErrContributionNotFound — no contribution row matches the (user_id, goal_id, id).
	ErrContributionNotFound = errors.New("storage: contribution not found")
)

// CreateGoal inserts a goal and returns the row with id/timestamps filled in.
// Idempotency is NOT on the goal itself (user can have many goals); it lives on
// contributions (see CreateContribution). Owner scoping happens via g.UserID.
func (p *Pool) CreateGoal(ctx context.Context, g domain.Goal) (domain.Goal, error) {
	if g.UserID == 0 {
		return domain.Goal{}, errors.New("storage: goal requires user_id")
	}
	const q = `
		INSERT INTO goals (user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	monthlyNull, dateNull := goalNullables(g.MonthlyContribution, g.TargetDate)
	err := p.DB.QueryRowContext(ctx, q,
		g.UserID, g.Name, g.TargetAmount.Decimal(), g.CurrentAmount.Decimal(),
		monthlyNull, dateNull, g.ExpectedYield,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return domain.Goal{}, fmt.Errorf("storage: create goal: %w", err)
	}
	return g, nil
}

// GetGoal returns one goal scoped to the owner, or ErrGoalNotFound.
// The user_id filter is mandatory — never query goals across users.
func (p *Pool) GetGoal(ctx context.Context, userID, id int64) (domain.Goal, error) {
	const q = `
		SELECT id, user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield, created_at, updated_at
		FROM goals
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	row := p.DB.QueryRowContext(ctx, q, id, userID)
	g, err := scanGoal(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Goal{}, ErrGoalNotFound
		}
		return domain.Goal{}, fmt.Errorf("storage: get goal: %w", err)
	}
	return g, nil
}

// ListGoals returns all non-deleted goals for the user, ordered by id.
func (p *Pool) ListGoals(ctx context.Context, userID int64) ([]domain.Goal, error) {
	const q = `
		SELECT id, user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield, created_at, updated_at
		FROM goals
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY id
	`
	rows, err := p.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: list goals: %w", err)
	}
	defer rows.Close()

	var out []domain.Goal
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("storage: scan goal: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// UpdateGoal mutates the editable fields of a goal. id/user_id are identity and
// must match an existing non-deleted row; otherwise ErrGoalNotFound.
func (p *Pool) UpdateGoal(ctx context.Context, g domain.Goal) (domain.Goal, error) {
	const q = `
		UPDATE goals
		SET name = $1, target_amount = $2, current_amount = $3,
		    monthly_contribution = $4, target_date = $5, expected_yield = $6
		WHERE id = $7 AND user_id = $8 AND deleted_at IS NULL
		RETURNING id, user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield, created_at, updated_at
	`
	monthlyNull, dateNull := goalNullables(g.MonthlyContribution, g.TargetDate)
	row := p.DB.QueryRowContext(ctx, q,
		g.Name, g.TargetAmount.Decimal(), g.CurrentAmount.Decimal(),
		monthlyNull, dateNull, g.ExpectedYield, g.ID, g.UserID,
	)
	out, err := scanGoal(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Goal{}, ErrGoalNotFound
		}
		return domain.Goal{}, fmt.Errorf("storage: update goal: %w", err)
	}
	return out, nil
}

// DeleteGoal soft-deletes a goal (sets deleted_at). Contributions are cascaded
// physically via ON DELETE CASCADE in migration 0003 only on hard delete; the
// service layer reads goals with deleted_at IS NULL so soft-deleted goals
// disappear from projection/listing while their contributions stay on disk
// (recoverable). Returns ErrGoalNotFound if no row was touched.
func (p *Pool) DeleteGoal(ctx context.Context, userID, id int64) error {
	const q = `UPDATE goals SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	res, err := p.DB.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("storage: delete goal: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: delete goal rows affected: %w", err)
	}
	if n == 0 {
		return ErrGoalNotFound
	}
	return nil
}

// SumContributions returns Σ amount of all contributions for a goal. Used by
// the projection layer for the hybrid effective-current model
// (design spec §3.2): effective = goal.current_amount + Σ contributions.
// Zero when the goal has no contributions (COALESCE in SQL).
func (p *Pool) SumContributions(ctx context.Context, userID, goalID int64) (domain.Money, error) {
	const q = `SELECT COALESCE(SUM(amount), 0) FROM goal_contributions WHERE user_id = $1 AND goal_id = $2`
	tot := new(decimalScanner)
	if err := p.DB.QueryRowContext(ctx, q, userID, goalID).Scan(tot); err != nil {
		return domain.Zero, fmt.Errorf("storage: sum contributions: %w", err)
	}
	return domain.FromDecimal(tot.d), nil
}

// CreateContribution inserts an ad-hoc top-up. Idempotency: (user_id, goal_id,
// contribution_id) UNIQUE → 23505 translates to ErrContributionExists, style
// of operations.CreateOperation + translatePgError (ф.1). The service layer
// fetches the original by (user, goal, contribution_id) on conflict so it can
// return the identical row to the caller (200, not 409).
func (p *Pool) CreateContribution(ctx context.Context, c domain.GoalContribution) (domain.GoalContribution, error) {
	if c.UserID == 0 || c.GoalID == 0 {
		return domain.GoalContribution{}, errors.New("storage: contribution requires user_id and goal_id")
	}
	if c.ContributionID == "" {
		return domain.GoalContribution{}, errors.New("storage: contribution_id required")
	}
	const q = `
		INSERT INTO goal_contributions (user_id, goal_id, contribution_id, amount, contribution_date, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := p.DB.QueryRowContext(ctx, q,
		c.UserID, c.GoalID, c.ContributionID, c.Amount.Decimal(), c.ContributionDate, c.Comment,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return domain.GoalContribution{}, translatePgError(err, "goal_contributions_user_id_goal_id_contribution_id_key", ErrContributionExists)
	}
	return c, nil
}

// GetContributionByClientID returns a contribution row by the client-generated
// contribution_id. Used to fulfil the idempotent response on ErrContributionExists
// (analogous to operations.GetOperationByCalcID in ф.1).
func (p *Pool) GetContributionByClientID(ctx context.Context, userID, goalID int64, contributionID string) (domain.GoalContribution, error) {
	const q = `
		SELECT id, user_id, goal_id, contribution_id, amount, contribution_date, comment, created_at
		FROM goal_contributions
		WHERE user_id = $1 AND goal_id = $2 AND contribution_id = $3
	`
	row := p.DB.QueryRowContext(ctx, q, userID, goalID, contributionID)
	c, err := scanContribution(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.GoalContribution{}, ErrContributionNotFound
		}
		return domain.GoalContribution{}, fmt.Errorf("storage: get contribution: %w", err)
	}
	return c, nil
}

// ListContributions returns the contribution journal for a goal, newest first.
func (p *Pool) ListContributions(ctx context.Context, userID, goalID int64) ([]domain.GoalContribution, error) {
	const q = `
		SELECT id, user_id, goal_id, contribution_id, amount, contribution_date, comment, created_at
		FROM goal_contributions
		WHERE user_id = $1 AND goal_id = $2
		ORDER BY contribution_date DESC, id DESC
	`
	rows, err := p.DB.QueryContext(ctx, q, userID, goalID)
	if err != nil {
		return nil, fmt.Errorf("storage: list contributions: %w", err)
	}
	defer rows.Close()

	var out []domain.GoalContribution
	for rows.Next() {
		c, err := scanContribution(rows)
		if err != nil {
			return nil, fmt.Errorf("storage: scan contribution: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteContribution hard-deletes a contribution row by (user, goal, row id).
// Hard delete (not soft) because the contribution journal is event-log-shaped:
// removing an event is the only way to recompute Σ correctly without a tombstone
// filter in every aggregation. The service layer recomputes projection after.
func (p *Pool) DeleteContribution(ctx context.Context, userID, goalID, id int64) error {
	const q = `DELETE FROM goal_contributions WHERE id = $1 AND user_id = $2 AND goal_id = $3`
	res, err := p.DB.ExecContext(ctx, q, id, userID, goalID)
	if err != nil {
		return fmt.Errorf("storage: delete contribution: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: delete contribution rows affected: %w", err)
	}
	if n == 0 {
		return ErrContributionNotFound
	}
	return nil
}

// --- helpers ---

// goalNullables packs the optional pointer fields of a goal into the Nullable
// shapes database/sql expects for binding (INSERT/UPDATE args).
func goalNullables(monthly *domain.Money, targetDate *time.Time) (sql.NullString, sql.NullTime) {
	var monthlyNull sql.NullString
	if monthly != nil {
		monthlyNull = sql.NullString{String: monthly.String(), Valid: true}
	}
	var dateNull sql.NullTime
	if targetDate != nil {
		dateNull = sql.NullTime{Time: *targetDate, Valid: true}
	}
	return monthlyNull, dateNull
}

// scanner is the subset of *sql.Row / *sql.Rows both implement — just Scan.
// Lets scanGoal serve Get/Update (Row) and List (Rows) with one code path.
// NB: тип scanner уже объявлен в operations.go и shared across the package; we
// переиспользуем его, чтобы не дублировать.

func scanGoal(s scanner) (domain.Goal, error) {
	var (
		g           domain.Goal
		targetRaw   = new(decimalScanner)
		currentRaw  = new(decimalScanner)
		yieldRaw    = new(decimalScanner)
		monthlyNull sql.NullString
		dateNull    sql.NullTime
	)
	if err := s.Scan(
		&g.ID, &g.UserID, &g.Name, targetRaw, currentRaw, &monthlyNull, &dateNull, yieldRaw,
		&g.CreatedAt, &g.UpdatedAt,
	); err != nil {
		return domain.Goal{}, err
	}
	g.TargetAmount = domain.FromDecimal(targetRaw.d)
	g.CurrentAmount = domain.FromDecimal(currentRaw.d)
	g.ExpectedYield = yieldRaw.d
	if monthlyNull.Valid {
		if m, err := domain.ParseMoney(monthlyNull.String); err == nil {
			g.MonthlyContribution = &m
		}
	}
	if dateNull.Valid {
		t := dateNull.Time
		g.TargetDate = &t
	}
	return g, nil
}

func scanContribution(s scanner) (domain.GoalContribution, error) {
	var (
		c         domain.GoalContribution
		amountRaw = new(decimalScanner)
		dateRaw   time.Time
	)
	if err := s.Scan(
		&c.ID, &c.UserID, &c.GoalID, &c.ContributionID, amountRaw, &dateRaw, &c.Comment, &c.CreatedAt,
	); err != nil {
		return domain.GoalContribution{}, err
	}
	c.Amount = domain.FromDecimal(amountRaw.d)
	c.ContributionDate = dateRaw
	return c, nil
}
