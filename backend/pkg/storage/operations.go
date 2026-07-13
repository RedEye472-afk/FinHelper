package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
)

// Operation is the persisted operation record. Mirrors domain.Operation but
// stays in storage to match the pattern of User/Account (until a richer
// domain behaviour emerges; domain.Operation already covers the invariants).
type Operation = domain.Operation

// Sentinel errors for operations.
var (
	// ErrOperationExists — duplicate (user_id, calc_id). Idempotency violation.
	ErrOperationExists = errors.New("storage: operation already exists (calc_id)")
	// ErrOperationNotFound — no row matches (user_id, id).
	ErrOperationNotFound = errors.New("storage: operation not found")
)

// OperationFilter narrows ListOperations. Zero-value filter = all operations
// for the user (subject to pagination).
type OperationFilter struct {
	From     *time.Time // operation_date >= From (inclusive)
	To       *time.Time // operation_date <= To (inclusive)
	Types    []domain.OperationType
	AccID    *int64
	CatID    *int64
	Planned  *bool // nil = any, true = only planned, false = only actual
}

// Page is cursor-style pagination. Limit<=0 falls back to a default of 50,
// capped at 200 to bound query cost. BeforeID returns rows with id < BeforeID
// in descending order — the client uses the last seen id as the next cursor.
type Page struct {
	Limit    int
	BeforeID *int64
}

const (
	// DefaultPageLimit is applied when a Page.Limit is unset. Exported so
	// callers can mirror the default without importing a magic number.
	DefaultPageLimit = 50
	maxPageLimit     = 200
)

// CreateOperation inserts an operation. Returns ErrOperationExists on a
// duplicate calc_id for the same user — this is the idempotency signal: the
// caller treats it as "already created, fetch and return existing".
//
// Ownership of account_id / account_dst_id / category_id is validated by FK;
// we additionally scope the INSERT through a CTE that checks user ownership
// of the source account so a malicious client can't target another user's
// account by guessing its id.
func (p *Pool) CreateOperation(ctx context.Context, op Operation) (Operation, error) {
	if op.UserID == 0 {
		return Operation{}, errors.New("storage: operation requires user_id")
	}
	const q = `
		INSERT INTO operations (
			user_id, calc_id, operation_type, amount, amount_dst, currency,
			account_id, account_dst_id, category_id, income_subtype,
			counterparty, description, operation_date, is_planned, category_confidence
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, created_at, updated_at
	`
	var (
		amountDst   any
		accDst      any
		catID       any
		incomeSub   any
		confidence  any
	)
	if op.AmountDst != nil {
		amountDst = op.AmountDst.Decimal()
	}
	if op.AccountDstID != nil {
		accDst = *op.AccountDstID
	}
	if op.CategoryID != nil {
		catID = *op.CategoryID
	}
	if op.IncomeSubtype != nil {
		incomeSub = string(*op.IncomeSubtype)
	}
	if op.CategoryConfidence != nil {
		confidence = op.CategoryConfidence
	}
	err := p.DB.QueryRowContext(ctx, q,
		op.UserID, op.CalcID, string(op.Type), op.Amount.Decimal(), amountDst, op.Currency,
		op.AccountID, accDst, catID, incomeSub,
		op.Counterparty, op.Description, op.OperationDate, op.IsPlanned, confidence,
	).Scan(&op.ID, &op.CreatedAt, &op.UpdatedAt)
	if err != nil {
		return Operation{}, translatePgError(err, "operations_user_id_calc_id_key", ErrOperationExists)
	}
	return op, nil
}

// GetOperation returns one operation scoped to the owner.
func (p *Pool) GetOperation(ctx context.Context, userID, id int64) (Operation, error) {
	const q = `
		SELECT id, user_id, calc_id, operation_type, amount, amount_dst, currency,
		       account_id, account_dst_id, category_id, income_subtype,
		       counterparty, description, operation_date, is_planned,
		       category_confidence, created_at, updated_at
		FROM operations
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	return scanOperation(p.DB.QueryRowContext(ctx, q, id, userID))
}

// GetOperationByCalcID returns an operation by its idempotency key. Used by
// the service layer when CreateOperation returns ErrOperationExists — the
// client gets back the original result instead of an error.
func (p *Pool) GetOperationByCalcID(ctx context.Context, userID int64, calcID string) (Operation, error) {
	const q = `
		SELECT id, user_id, calc_id, operation_type, amount, amount_dst, currency,
		       account_id, account_dst_id, category_id, income_subtype,
		       counterparty, description, operation_date, is_planned,
		       category_confidence, created_at, updated_at
		FROM operations
		WHERE user_id = $1 AND calc_id = $2 AND deleted_at IS NULL
	`
	return scanOperation(p.DB.QueryRowContext(ctx, q, userID, calcID))
}

// ListOperations returns a page of operations for the user, newest first.
// Filters and pagination are applied at SQL level to keep the result set bounded.
func (p *Pool) ListOperations(ctx context.Context, userID int64, f OperationFilter, page Page) ([]Operation, error) {
	var (
		clauses = []string{"user_id = $1", "deleted_at IS NULL"}
		args    = []any{userID}
		nextArg = func(v any) string {
			args = append(args, v)
			return fmt.Sprintf("$%d", len(args))
		}
	)
	if f.From != nil {
		clauses = append(clauses, "operation_date >= "+nextArg(*f.From))
	}
	if f.To != nil {
		clauses = append(clauses, "operation_date <= "+nextArg(*f.To))
	}
	if f.AccID != nil {
		clauses = append(clauses, "account_id = "+nextArg(*f.AccID))
	}
	if f.CatID != nil {
		clauses = append(clauses, "category_id = "+nextArg(*f.CatID))
	}
	if f.Planned != nil {
		clauses = append(clauses, "is_planned = "+nextArg(*f.Planned))
	}
	if len(f.Types) == 1 {
		clauses = append(clauses, "operation_type = "+nextArg(string(f.Types[0])))
	} else if len(f.Types) > 1 {
		ph := make([]string, len(f.Types))
		for i, t := range f.Types {
			ph[i] = nextArg(string(t))
		}
		clauses = append(clauses, "operation_type IN ("+strings.Join(ph, ",")+")")
	}
	if page.BeforeID != nil {
		clauses = append(clauses, "id < "+nextArg(*page.BeforeID))
	}

	limit := page.Limit
	if limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	// fetch limit+1 so the caller can tell whether more rows exist
	args = append(args, limit+1)

	q := fmt.Sprintf(`
		SELECT id, user_id, calc_id, operation_type, amount, amount_dst, currency,
		       account_id, account_dst_id, category_id, income_subtype,
		       counterparty, description, operation_date, is_planned,
		       category_confidence, created_at, updated_at
		FROM operations
		WHERE %s
		ORDER BY id DESC
		LIMIT $%d
	`, strings.Join(clauses, " AND "), len(args))

	rows, err := p.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: list operations: %w", err)
	}
	defer rows.Close()

	out := make([]Operation, 0, limit)
	for rows.Next() {
		op, err := scanOperation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	return out, rows.Err()
}

// DeleteOperation soft-deletes an operation (sets deleted_at = NOW()). The
// service layer is responsible for recomputing affected account balances.
func (p *Pool) DeleteOperation(ctx context.Context, userID, id int64) error {
	const q = `
		UPDATE operations SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	res, err := p.DB.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("storage: delete operation: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: delete operation rows affected: %w", err)
	}
	if n == 0 {
		return ErrOperationNotFound
	}
	return nil
}

// UpdateOperationCategory changes only the category assignment + confidence.
// This is the path the auto-categorizer and manual overrides use; full-row
// updates go through Delete+Create to preserve idempotency of the audit trail.
func (p *Pool) UpdateOperationCategory(ctx context.Context, userID, id int64, categoryID *int64, confidence *decimal.Decimal) error {
	const q = `
		UPDATE operations
		SET category_id = $1, category_confidence = $2
		WHERE id = $3 AND user_id = $4 AND deleted_at IS NULL
	`
	var cat any
	if categoryID != nil {
		cat = *categoryID
	}
	res, err := p.DB.ExecContext(ctx, q, cat, confidence, id, userID)
	if err != nil {
		return fmt.Errorf("storage: update operation category: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: update operation category rows affected: %w", err)
	}
	if n == 0 {
		return ErrOperationNotFound
	}
	return nil
}

// scanner abstracts *sql.Row and *sql.Rows for scanOperation.
type scanner interface {
	Scan(dest ...any) error
}

func scanOperation(s scanner) (Operation, error) {
	var (
		op          Operation
		opType      string
		amountRaw   decimal.Decimal
		amountDstS  sql.NullString
		accDst      sql.NullInt64
		catID       sql.NullInt64
		incomeSub   sql.NullString
		confidenceS sql.NullString
	)
	err := s.Scan(
		&op.ID, &op.UserID, &op.CalcID, &opType, &amountRaw, &amountDstS, &op.Currency,
		&op.AccountID, &accDst, &catID, &incomeSub,
		&op.Counterparty, &op.Description, &op.OperationDate, &op.IsPlanned,
		&confidenceS, &op.CreatedAt, &op.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Operation{}, ErrOperationNotFound
		}
		return Operation{}, fmt.Errorf("storage: scan operation: %w", err)
	}
	op.Type = domain.OperationType(opType)
	op.Amount = domain.FromDecimal(amountRaw)
	if amountDstS.Valid && amountDstS.String != "" {
		d, perr := decimal.NewFromString(amountDstS.String)
		if perr != nil {
			return Operation{}, fmt.Errorf("storage: parse amount_dst %q: %w", amountDstS.String, perr)
		}
		m := domain.FromDecimal(d)
		op.AmountDst = &m
	}
	if accDst.Valid {
		v := accDst.Int64
		op.AccountDstID = &v
	}
	if catID.Valid {
		v := catID.Int64
		op.CategoryID = &v
	}
	if incomeSub.Valid {
		s := domain.IncomeSubtype(incomeSub.String)
		op.IncomeSubtype = &s
	}
	if confidenceS.Valid && confidenceS.String != "" {
		d, perr := decimal.NewFromString(confidenceS.String)
		if perr == nil {
			op.CategoryConfidence = &d
		}
		// On parse failure we leave confidence nil rather than fail the row —
		// a malformed confidence must not break reading the whole operation.
	}
	return op, nil
}

// SumByAccountSince returns the recomputed balance for the account: the sum
// of every non-deleted operation's effect on it. Used by the operations
// service to keep the cached accounts.balance column self-healing.
//
// Effect per operation type (BUSINESS_LOGIC.md ф.1):
//   - income / refund:   +amount on the source account
//   - expense:           −amount on the source account
//   - transfer / currency_exchange: −amount on source, +amount_dst
//     (or +amount when amount_dst IS NULL) on the destination account
//
// The query matches the account as either leg of the operation, so a single
// pass correctly handles transfers. Returns Zero on no rows.
// DeleteOperationsByAccount deletes all operations for a given account (hard delete).
// Optionally filters by date range (from <= operation_date <= to).
// Returns count of deleted rows. Used for full reimport or cleanup.
func (p *Pool) DeleteOperationsByAccount(ctx context.Context, accountID int64, from, to *time.Time) (int64, error) {
	q := `DELETE FROM operations WHERE account_id = $1 AND deleted_at IS NULL`
	args := []any{accountID}
	argNum := 2
	if from != nil {
		q += fmt.Sprintf(" AND operation_date >= $%d", argNum)
		args = append(args, *from)
		argNum++
	}
	if to != nil {
		q += fmt.Sprintf(" AND operation_date <= $%d", argNum)
		args = append(args, *to)
		argNum++
	}
	res, err := p.DB.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("storage: delete operations by account: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("storage: delete operations by account rows affected: %w", err)
	}
	return n, nil
}

// SumByAccountSince returns the recomputed balance for the account: the sum
// of every non-deleted operation's effect on it. Used by the operations
// service to keep the cached accounts.balance column self-healing.
func (p *Pool) SumByAccountSince(ctx context.Context, accountID int64) (domain.Money, error) {
	const q = `
		SELECT COALESCE(SUM(leg), 0) FROM (
			SELECT
				CASE
					WHEN operation_type IN ('income', 'refund') THEN amount
					WHEN operation_type = 'expense'             THEN -amount
					WHEN operation_type IN ('transfer', 'currency_exchange') THEN -amount
					ELSE 0
				END AS leg
			FROM operations
			WHERE account_id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT
				CASE
					WHEN operation_type IN ('transfer', 'currency_exchange')
						THEN COALESCE(amount_dst, amount)
					ELSE 0
				END
			FROM operations
			WHERE account_dst_id = $1 AND deleted_at IS NULL
		) AS legs
	`
	var sum decimal.Decimal
	if err := p.DB.QueryRowContext(ctx, q, accountID).Scan(&sum); err != nil {
		return domain.Zero, fmt.Errorf("storage: sum by account: %w", err)
	}
	return domain.FromDecimal(sum), nil
}
