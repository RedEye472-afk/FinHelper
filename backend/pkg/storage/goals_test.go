package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
)

// goalCols — column list returned by every SELECT / UPDATE...RETURNING on goals.
// Keep in lockstep with scanGoal.
var goalCols = []string{
	"id", "user_id", "name", "target_amount", "current_amount",
	"monthly_contribution", "target_date", "expected_yield",
	"created_at", "updated_at",
}

// contributionCols — column list for SELECTs on goal_contributions.
var contributionCols = []string{
	"id", "user_id", "goal_id", "contribution_id", "amount",
	"contribution_date", "comment", "created_at",
}

// ----------------------------------------------------------------------------
// CreateGoal
// ----------------------------------------------------------------------------

func TestCreateGoal_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	yield := decimal.NewFromFloat(0.08)

	mock.ExpectQuery(q(`INSERT INTO goals`)).
		WithArgs(int64(7), "Машина", sqlmock.AnyArg(), sqlmock.AnyArg(),
			sql.NullString{}, sql.NullTime{}, yield).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(1), now, now))

	g, err := pool.CreateGoal(ctx, domain.Goal{
		UserID: 7, Name: "Машина",
		TargetAmount:  domain.MustParseMoney("1000000.00"),
		CurrentAmount: domain.Zero,
		ExpectedYield: yield,
	})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g.ID != 1 {
		t.Errorf("id = %d, want 1", g.ID)
	}
	if !g.CreatedAt.Equal(now) {
		t.Errorf("created_at = %s, want %s", g.CreatedAt, now)
	}
}

func TestCreateGoal_WithOptionals(t *testing.T) {
	// monthly_contribution + target_date заданы → NullString/NullTime валидны.
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	deadline := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	monthly := domain.MustParseMoney("50000.00")
	yield := decimal.NewFromFloat(0.08)

	mock.ExpectQuery(q(`INSERT INTO goals`)).
		WithArgs(int64(7), "Квартира", sqlmock.AnyArg(), sqlmock.AnyArg(),
			sql.NullString{String: monthly.String(), Valid: true},
			sql.NullTime{Time: deadline, Valid: true},
			yield).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(2), now, now))

	g, err := pool.CreateGoal(ctx, domain.Goal{
		UserID: 7, Name: "Квартира",
		TargetAmount:         domain.MustParseMoney("5000000.00"),
		CurrentAmount:        domain.MustParseMoney("500000.00"),
		MonthlyContribution:  &monthly,
		TargetDate:           &deadline,
		ExpectedYield:        yield,
	})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if g.ID != 2 {
		t.Errorf("id = %d, want 2", g.ID)
	}
}

func TestCreateGoal_RequiresUserID(t *testing.T) {
	pool, _ := newMockPool(t)
	_, err := pool.CreateGoal(context.Background(), domain.Goal{Name: "X", TargetAmount: domain.MustParseMoney("1.00")})
	if err == nil {
		t.Errorf("expected error for missing user_id")
	}
}

// ----------------------------------------------------------------------------
// GetGoal
// ----------------------------------------------------------------------------

func TestGetGoal_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(q(`SELECT .* FROM goals WHERE id`)).
		WithArgs(int64(1), int64(7)).
		WillReturnRows(sqlmock.NewRows(goalCols).
			AddRow(int64(1), int64(7), "Машина", "1000000.00", "0", sql.NullString{}, sql.NullTime{}, "0.08", now, now))

	g, err := pool.GetGoal(ctx, 7, 1)
	if err != nil {
		t.Fatalf("GetGoal: %v", err)
	}
	if g.Name != "Машина" {
		t.Errorf("name = %s", g.Name)
	}
	if !g.TargetAmount.Equal(domain.MustParseMoney("1000000.00")) {
		t.Errorf("target = %s", g.TargetAmount)
	}
	if g.MonthlyContribution != nil {
		t.Errorf("expected nil monthly, got %s", g.MonthlyContribution)
	}
}

func TestGetGoal_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`SELECT .* FROM goals WHERE id`)).
		WithArgs(int64(99), int64(7)).
		WillReturnError(sql.ErrNoRows)
	_, err := pool.GetGoal(context.Background(), 7, 99)
	if !errors.Is(err, ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// ListGoals
// ----------------------------------------------------------------------------

func TestListGoals_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows(goalCols).
		AddRow(int64(1), int64(7), "Машина", "1000000.00", "0", sql.NullString{}, sql.NullTime{}, "0.08", now, now).
		AddRow(int64(2), int64(7), "Отпуск", "300000.00", "50000.00", "10000.00", sql.NullTime{}, "0.05", now, now)
	mock.ExpectQuery(q(`SELECT .* FROM goals WHERE user_id`)).
		WithArgs(int64(7)).
		WillReturnRows(rows)

	gs, err := pool.ListGoals(ctx, 7)
	if err != nil {
		t.Fatalf("ListGoals: %v", err)
	}
	if len(gs) != 2 {
		t.Fatalf("expected 2 goals, got %d", len(gs))
	}
	if gs[1].MonthlyContribution == nil {
		t.Errorf("expected non-nil monthly for second goal")
	}
}

func TestListGoals_Empty(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`SELECT .* FROM goals WHERE user_id`)).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows(goalCols))
	gs, err := pool.ListGoals(context.Background(), 7)
	if err != nil {
		t.Fatalf("ListGoals empty: %v", err)
	}
	if gs != nil && len(gs) != 0 {
		t.Errorf("expected nil/empty, got %d", len(gs))
	}
}

// ----------------------------------------------------------------------------
// UpdateGoal
// ----------------------------------------------------------------------------

func TestUpdateGoal_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(q(`UPDATE goals SET`)).
		WithArgs("Отпуск", sqlmock.AnyArg(), sqlmock.AnyArg(),
			sql.NullString{}, sql.NullTime{}, decimal.NewFromFloat(0.05),
			int64(2), int64(7)).
		WillReturnRows(sqlmock.NewRows(goalCols).
			AddRow(int64(2), int64(7), "Отпуск", "350000.00", "50000.00", sql.NullString{}, sql.NullTime{}, "0.05", now, now))

	g, err := pool.UpdateGoal(ctx, domain.Goal{
		ID: 2, UserID: 7, Name: "Отпуск",
		TargetAmount:  domain.MustParseMoney("350000.00"),
		CurrentAmount: domain.MustParseMoney("50000.00"),
		ExpectedYield: decimal.NewFromFloat(0.05),
	})
	if err != nil {
		t.Fatalf("UpdateGoal: %v", err)
	}
	if !g.TargetAmount.Equal(domain.MustParseMoney("350000.00")) {
		t.Errorf("target after update = %s", g.TargetAmount)
	}
}

func TestUpdateGoal_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`UPDATE goals SET`)).
		WillReturnError(sql.ErrNoRows)
	_, err := pool.UpdateGoal(context.Background(), domain.Goal{
		ID: 999, UserID: 7, Name: "X", TargetAmount: domain.MustParseMoney("1.00"),
	})
	if !errors.Is(err, ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// DeleteGoal
// ----------------------------------------------------------------------------

func TestDeleteGoal_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectExec(q(`UPDATE goals SET deleted_at`)).
		WithArgs(int64(1), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := pool.DeleteGoal(context.Background(), 7, 1); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}
}

func TestDeleteGoal_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectExec(q(`UPDATE goals SET deleted_at`)).
		WithArgs(int64(99), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err := pool.DeleteGoal(context.Background(), 7, 99)
	if !errors.Is(err, ErrGoalNotFound) {
		t.Fatalf("expected ErrGoalNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// SumContributions
// ----------------------------------------------------------------------------

func TestSumContributions_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`SELECT COALESCE\(SUM\(amount\), 0\) FROM goal_contributions`)).
		WithArgs(int64(7), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow("75000.00"))
	got, err := pool.SumContributions(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("SumContributions: %v", err)
	}
	if !got.Equal(domain.MustParseMoney("75000.00")) {
		t.Errorf("sum = %s, want 75000.00", got)
	}
}

func TestSumContributions_None_Zero(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`SELECT COALESCE\(SUM\(amount\), 0\) FROM goal_contributions`)).
		WithArgs(int64(7), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow("0"))
	got, err := pool.SumContributions(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("SumContributions: %v", err)
	}
	if !got.Equal(domain.Zero) {
		t.Errorf("sum = %s, want 0", got)
	}
}

// ----------------------------------------------------------------------------
// CreateContribution
// ----------------------------------------------------------------------------

func TestCreateContribution_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	date := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(q(`INSERT INTO goal_contributions`)).
		WithArgs(int64(7), int64(1), "abc", sqlmock.AnyArg(), date, "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(10), now))

	c, err := pool.CreateContribution(ctx, domain.GoalContribution{
		UserID: 7, GoalID: 1, ContributionID: "abc",
		Amount:           domain.MustParseMoney("5000.00"),
		ContributionDate: date,
	})
	if err != nil {
		t.Fatalf("CreateContribution: %v", err)
	}
	if c.ID != 10 {
		t.Errorf("id = %d, want 10", c.ID)
	}
}

func TestCreateContribution_Duplicate(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`INSERT INTO goal_contributions`)).
		WillReturnError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "goal_contributions_user_id_goal_id_contribution_id_key",
			Message:        "duplicate",
		})
	_, err := pool.CreateContribution(context.Background(), domain.GoalContribution{
		UserID: 7, GoalID: 1, ContributionID: "abc",
		Amount:           domain.MustParseMoney("5000.00"),
		ContributionDate: time.Now(),
	})
	if !errors.Is(err, ErrContributionExists) {
		t.Fatalf("expected ErrContributionExists, got %v", err)
	}
}

func TestCreateContribution_RequiresIDs(t *testing.T) {
	pool, _ := newMockPool(t)
	_, err := pool.CreateContribution(context.Background(), domain.GoalContribution{
		GoalID: 1, ContributionID: "abc", Amount: domain.MustParseMoney("1.00"),
	})
	if err == nil {
		t.Errorf("expected error for missing user_id")
	}
}

// ----------------------------------------------------------------------------
// GetContributionByClientID
// ----------------------------------------------------------------------------

func TestGetContributionByClientID_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	date := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(q(`SELECT .* FROM goal_contributions WHERE user_id`)).
		WithArgs(int64(7), int64(1), "abc").
		WillReturnRows(sqlmock.NewRows(contributionCols).
			AddRow(int64(10), int64(7), int64(1), "abc", "5000.00", date, "премия", now))

	c, err := pool.GetContributionByClientID(ctx, 7, 1, "abc")
	if err != nil {
		t.Fatalf("GetContributionByClientID: %v", err)
	}
	if c.ID != 10 || c.ContributionID != "abc" {
		t.Errorf("got = %+v", c)
	}
	if !c.Amount.Equal(domain.MustParseMoney("5000.00")) {
		t.Errorf("amount = %s", c.Amount)
	}
}

func TestGetContributionByClientID_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectQuery(q(`SELECT .* FROM goal_contributions WHERE user_id`)).
		WithArgs(int64(7), int64(1), "missing").
		WillReturnError(sql.ErrNoRows)
	_, err := pool.GetContributionByClientID(context.Background(), 7, 1, "missing")
	if !errors.Is(err, ErrContributionNotFound) {
		t.Fatalf("expected ErrContributionNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// ListContributions
// ----------------------------------------------------------------------------

func TestListContributions_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	d1 := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows(contributionCols).
		AddRow(int64(11), int64(7), int64(1), "b", "3000.00", d1, "", now).
		AddRow(int64(10), int64(7), int64(1), "a", "5000.00", d2, "премия", now)
	mock.ExpectQuery(q(`SELECT .* FROM goal_contributions WHERE user_id`)).
		WithArgs(int64(7), int64(1)).
		WillReturnRows(rows)

	cs, err := pool.ListContributions(ctx, 7, 1)
	if err != nil {
		t.Fatalf("ListContributions: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("expected 2, got %d", len(cs))
	}
	if cs[0].ContributionID != "b" {
		t.Errorf("expected newest first (b), got %s", cs[0].ContributionID)
	}
}

// ----------------------------------------------------------------------------
// DeleteContribution
// ----------------------------------------------------------------------------

func TestDeleteContribution_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectExec(q(`DELETE FROM goal_contributions WHERE id`)).
		WithArgs(int64(10), int64(7), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := pool.DeleteContribution(context.Background(), 7, 1, 10); err != nil {
		t.Fatalf("DeleteContribution: %v", err)
	}
}

func TestDeleteContribution_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	mock.ExpectExec(q(`DELETE FROM goal_contributions WHERE id`)).
		WithArgs(int64(99), int64(7), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err := pool.DeleteContribution(context.Background(), 7, 1, 99)
	if !errors.Is(err, ErrContributionNotFound) {
		t.Fatalf("expected ErrContributionNotFound, got %v", err)
	}
}
