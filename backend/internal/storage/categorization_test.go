package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
)

// ----------------------------------------------------------------------------
// CreateRule
// ----------------------------------------------------------------------------

func TestCreateRule_Success_UserPriority(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`INSERT INTO categorization_rules`)).
		WithArgs(int64(7), "ozon", int64(10), string(domain.RuleUser), 100, true).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	r, err := pool.CreateRule(ctx, CategorizationRule{
		UserID: 7, Keyword: "ozon", CategoryID: 10,
		Source: domain.RuleUser, IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("CreateRule: %v", err)
	}
	if r.ID != 1 || r.Priority != 100 {
		t.Errorf("rule = %+v, want id=1 priority=100", r)
	}
}

func TestCreateRule_Duplicate(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`INSERT INTO categorization_rules`)).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "categorization_rules_user_id_keyword_key", Message: "dup"})

	_, err := pool.CreateRule(ctx, CategorizationRule{UserID: 7, Keyword: "ozon", CategoryID: 10})
	if !errors.Is(err, ErrRuleExists) {
		t.Fatalf("expected ErrRuleExists, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// ListRules
// ----------------------------------------------------------------------------

func TestListRules_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "user_id", "keyword", "category_id", "source", "priority", "is_enabled"}).
		AddRow(int64(1), int64(7), "ozon", int64(60), "user", 100, true).
		AddRow(int64(2), int64(7), "магнит", int64(10), "system", 0, true)
	mock.ExpectQuery(q(`SELECT .* FROM categorization_rules WHERE user_id`)).
		WithArgs(int64(7)).
		WillReturnRows(rows)

	rules, err := pool.ListRules(ctx, 7)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Source != domain.RuleUser || rules[0].Priority != 100 {
		t.Errorf("first rule = %+v, want user/100", rules[0])
	}
}

// ----------------------------------------------------------------------------
// DeleteRule
// ----------------------------------------------------------------------------

func TestDeleteRule_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectExec(q(`UPDATE categorization_rules SET deleted_at`)).
		WithArgs(int64(99), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := pool.DeleteRule(ctx, 7, 99)
	if !errors.Is(err, ErrRuleNotFound) {
		t.Fatalf("expected ErrRuleNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// GetOverride / UpsertOverrideConfirmation
// ----------------------------------------------------------------------------

func TestGetOverride_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT .* FROM counterparty_overrides WHERE user_id`)).
		WithArgs(int64(7), "ozon").
		WillReturnError(sql.ErrNoRows)

	_, err := pool.GetOverride(ctx, 7, "ozon")
	if !errors.Is(err, ErrOverrideNotFound) {
		t.Fatalf("expected ErrOverrideNotFound, got %v", err)
	}
}

func TestUpsertOverrideConfirmation_NewRow(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`INSERT INTO counterparty_overrides`)).
		WithArgs(int64(7), "ozon", int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "counterparty", "category_id", "confirmations"}).
			AddRow(int64(1), int64(7), "ozon", int64(10), 1))

	o, err := pool.UpsertOverrideConfirmation(ctx, 7, "ozon", 10)
	if err != nil {
		t.Fatalf("UpsertOverrideConfirmation: %v", err)
	}
	if o.Confirmations != 1 {
		t.Errorf("confirmations = %d, want 1", o.Confirmations)
	}
}

func TestSetOverrideCategory_ResetsConfirmations(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`UPDATE counterparty_overrides SET category_id`)).
		WithArgs(int64(20), int64(1), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "counterparty", "category_id", "confirmations"}).
			AddRow(int64(1), int64(7), "ozon", int64(20), 1))

	o, err := pool.SetOverrideCategory(ctx, 7, 1, 20)
	if err != nil {
		t.Fatalf("SetOverrideCategory: %v", err)
	}
	if o.Confirmations != 1 {
		t.Errorf("confirmations = %d, want reset to 1", o.Confirmations)
	}
}

func TestSetOverrideCategory_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`UPDATE counterparty_overrides SET category_id`)).
		WillReturnError(sql.ErrNoRows)

	_, err := pool.SetOverrideCategory(ctx, 7, 999, 20)
	if !errors.Is(err, ErrOverrideNotFound) {
		t.Fatalf("expected ErrOverrideNotFound, got %v", err)
	}
}
