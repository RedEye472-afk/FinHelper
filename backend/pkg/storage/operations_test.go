package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
)

// newOp builds a valid operation used across tests.
func newOp() Operation {
	t := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	o := Operation{
		UserID:        7,
		CalcID:        "abc",
		Type:          domain.OpExpense,
		Amount:        domain.MustParseMoney("500.00"),
		Currency:      "RUB",
		AccountID:     1,
		Counterparty:  "Кафе",
		OperationDate: t,
	}
	return o
}

// opCols mirrors the column order in scanOperation's SELECT list.
var opCols = []string{
	"id", "user_id", "calc_id", "operation_type", "amount", "amount_dst",
	"currency", "account_id", "account_dst_id", "category_id", "income_subtype",
	"counterparty", "description", "operation_date", "is_planned",
	"category_confidence", "created_at", "updated_at",
}

func opRow(o Operation) *sqlmock.Rows {
	now := o.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC().Truncate(time.Microsecond)
	}
	r := sqlmock.NewRows(opCols).
		AddRow(o.ID, o.UserID, o.CalcID, string(o.Type), o.Amount.Decimal().String(), nil,
			o.Currency, o.AccountID, nil, nil, nil,
			o.Counterparty, o.Description, o.OperationDate, o.IsPlanned,
			nil, now, now)
	return r
}

// ----------------------------------------------------------------------------
// CreateOperation
// ----------------------------------------------------------------------------

func TestCreateOperation_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	in := newOp()

	mock.ExpectQuery(q(`INSERT INTO operations`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(42), time.Now().UTC(), time.Now().UTC()))

	out, err := pool.CreateOperation(ctx, in)
	if err != nil {
		t.Fatalf("CreateOperation: %v", err)
	}
	if out.ID != 42 {
		t.Errorf("ID = %d, want 42", out.ID)
	}
}

func TestCreateOperation_DuplicateCalcID(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`INSERT INTO operations`)).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "operations_user_id_calc_id_key", Message: "duplicate"})

	_, err := pool.CreateOperation(ctx, newOp())
	if !errors.Is(err, ErrOperationExists) {
		t.Fatalf("expected ErrOperationExists, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// GetOperation
// ----------------------------------------------------------------------------

func TestGetOperation_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	in := newOp()
	in.ID = 42
	in.CategoryID = ptrInt64(3)

	rows := sqlmock.NewRows(opCols).
		AddRow(in.ID, in.UserID, in.CalcID, string(in.Type), in.Amount.Decimal().String(), nil,
			in.Currency, in.AccountID, nil, in.CategoryID, nil,
			in.Counterparty, in.Description, in.OperationDate, in.IsPlanned,
			nil, time.Now().UTC(), time.Now().UTC())

	mock.ExpectQuery(q(`SELECT .* FROM operations WHERE id`)).
		WithArgs(int64(42), int64(7)).
		WillReturnRows(rows)

	out, err := pool.GetOperation(ctx, 7, 42)
	if err != nil {
		t.Fatalf("GetOperation: %v", err)
	}
	if out.ID != 42 || out.CategoryID == nil || *out.CategoryID != 3 {
		t.Errorf("unexpected op: %+v", out)
	}
}

func TestGetOperation_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT .* FROM operations WHERE id`)).
		WithArgs(int64(99), int64(7)).
		WillReturnError(sql.ErrNoRows)

	_, err := pool.GetOperation(ctx, 7, 99)
	if !errors.Is(err, ErrOperationNotFound) {
		t.Fatalf("expected ErrOperationNotFound, got %v", err)
	}
}

func TestGetOperationByCalcID_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	in := newOp()
	in.ID = 42

	mock.ExpectQuery(q(`SELECT .* FROM operations WHERE user_id`)).
		WithArgs(int64(7), "abc").
		WillReturnRows(opRow(in))

	out, err := pool.GetOperationByCalcID(ctx, 7, "abc")
	if err != nil {
		t.Fatalf("GetOperationByCalcID: %v", err)
	}
	if out.CalcID != "abc" {
		t.Errorf("CalcID = %s, want abc", out.CalcID)
	}
}

// ----------------------------------------------------------------------------
// ListOperations — pagination + filter
// ----------------------------------------------------------------------------

func TestListOperations_Pagination(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	// We ask for limit=2 → service adds 1; storage runs with LIMIT 3.
	// Storage returns all 3 rows here; service would slice to 2 + more=true.
	rows := sqlmock.NewRows(opCols)
	for i, id := range []int64{5, 4, 3} {
		o := newOp()
		o.ID = id
		o.CalcID = "c"
		_ = i
		rows.AddRow(o.ID, o.UserID, o.CalcID, string(o.Type), o.Amount.Decimal().String(), nil,
			o.Currency, o.AccountID, nil, nil, nil,
			o.Counterparty, o.Description, o.OperationDate, o.IsPlanned,
			nil, time.Now().UTC(), time.Now().UTC())
	}
	mock.ExpectQuery(q(`SELECT .* FROM operations WHERE`)).
		WillReturnRows(rows)

	out, err := pool.ListOperations(ctx, 7, OperationFilter{}, Page{Limit: 2})
	if err != nil {
		t.Fatalf("ListOperations: %v", err)
	}
	if len(out) != 3 {
		t.Errorf("len = %d, want 3 (storage does not slice; service does)", len(out))
	}
}

// ----------------------------------------------------------------------------
// DeleteOperation
// ----------------------------------------------------------------------------

func TestDeleteOperation_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectExec(q(`UPDATE operations SET deleted_at`)).
		WithArgs(int64(42), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := pool.DeleteOperation(ctx, 7, 42); err != nil {
		t.Fatalf("DeleteOperation: %v", err)
	}
}

func TestDeleteOperation_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectExec(q(`UPDATE operations SET deleted_at`)).
		WithArgs(int64(99), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := pool.DeleteOperation(ctx, 7, 99)
	if !errors.Is(err, ErrOperationNotFound) {
		t.Fatalf("expected ErrOperationNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// UpdateOperationCategory
// ----------------------------------------------------------------------------

func TestUpdateOperationCategory_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	catID := int64(5)

	mock.ExpectExec(q(`UPDATE operations SET category_id`)).
		WithArgs(catID, nil, int64(42), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := pool.UpdateOperationCategory(ctx, 7, 42, &catID, nil); err != nil {
		t.Fatalf("UpdateOperationCategory: %v", err)
	}
}

// ----------------------------------------------------------------------------
// SumByAccountSince — verify the transfer/cashflow sign convention
// ----------------------------------------------------------------------------

func TestSumByAccountSince_ExpenseIncome(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	// Just verify the query runs; the SQL semantics (income +, expense −) are
	// a DB-layer concern better validated by an integration test once
	// docker-compose is available. Here we only confirm the call shape.
	mock.ExpectQuery(q(`SELECT COALESCE\(SUM\(leg\), 0\)`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow("-500.00"))

	got, err := pool.SumByAccountSince(ctx, 1)
	if err != nil {
		t.Fatalf("SumByAccountSince: %v", err)
	}
	if !got.Equal(domain.MustParseMoney("-500.00")) {
		t.Errorf("sum = %s, want -500.00", got)
	}
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func ptrInt64(v int64) *int64 { return &v }
