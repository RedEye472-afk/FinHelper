// Package operations implements BUSINESS_LOGIC.md ф.1 — manual entry of
// financial operations and their effect on account balances.
//
// The service is the single place where business rules live:
//   - Operation invariants (domain.Operation.Validate).
//   - PII masking of counterparty / description before persistence.
//   - calc_id generation for idempotency.
//   - Recomputation of cached account balances from operation history.
//
// Storage is abstracted behind OperationRepo so the service is unit-testable
// without a running database (same pattern as auth.JWTVerifier).
package operations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/pii"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
)

// OperationRepo is the storage contract the service depends on. The concrete
// implementation is *storage.Pool; tests substitute a fake.
type OperationRepo interface {
	CreateOperation(ctx context.Context, op storage.Operation) (storage.Operation, error)
	GetOperation(ctx context.Context, userID, id int64) (storage.Operation, error)
	GetOperationByCalcID(ctx context.Context, userID int64, calcID string) (storage.Operation, error)
	ListOperations(ctx context.Context, userID int64, f storage.OperationFilter, page storage.Page) ([]storage.Operation, error)
	DeleteOperation(ctx context.Context, userID, id int64) error
	UpdateOperationCategory(ctx context.Context, userID, id int64, categoryID *int64, confidence *decimal.Decimal) error

	GetAccount(ctx context.Context, userID, id int64) (storage.Account, error)
	SetAccountBalance(ctx context.Context, userID, id int64, balance domain.Money) error

	// SumByAccountSince returns Σ signed_amount over non-deleted operations
	// of the account. Used to recompute the cached balance deterministically.
	SumByAccountSince(ctx context.Context, accountID int64) (domain.Money, error)
}

// AccountRepo is split out for callers that only need account operations.
// Satisfied by *storage.Pool.
type AccountRepo interface {
	CreateAccount(ctx context.Context, userID int64, name string, accType domain.AccountType, currency string) (storage.Account, error)
	GetAccount(ctx context.Context, userID, id int64) (storage.Account, error)
	ListAccounts(ctx context.Context, userID int64) ([]storage.Account, error)
	SetAccountBalance(ctx context.Context, userID, id int64, balance domain.Money) error
}

// Categorizer is an optional dependency that suggests a category for a new
// operation when the caller didn't supply one. Defined here (rather than
// importing service/categorization) to avoid an import cycle and to keep the
// operations package decoupled from the categorization implementation.
//
// nil categorizer = auto-categorization disabled (the create path still works,
// just leaves CategoryID nil). This keeps graceful degradation working: a
// smoke/CI boot without a categorizer does not panic.
type Categorizer interface {
	// CategorizeForCreate is a trimmed view of categorization.Service.Categorize:
	// it returns a suggested (categoryID, confidence) for the masked
	// counterparty + description, or (0, _) when no rule matches.
	CategorizeForCreate(ctx context.Context, userID int64, counterparty, description string) (categoryID int64, confidence *decimal.Decimal, err error)
}

// Service is the operations business layer. Construct once at boot, share
// across requests — it holds no per-request state.
type Service struct {
	repo       OperationRepo
	categorizer Categorizer // nil = auto-categorization disabled
	now        func() time.Time
}

// NewService returns a Service. repo must be non-nil. The categorizer defaults
// to nil (disabled); attach one with SetCategorizer at wiring time.
func NewService(repo OperationRepo) *Service {
	return &Service{repo: repo, now: time.Now}
}

// SetCategorizer attaches an auto-categorizer. Optional: calling it is only
// required when /operations should auto-assign categories on create. Passing
// nil disables the feature again.
func (s *Service) SetCategorizer(c Categorizer) { s.categorizer = c }

// Sentinel errors surfaced to handlers. Storage errors are mapped to these.
var (
	// ErrInvalidArgument — request failed validation; details in the wrapped err.
	ErrInvalidArgument = errors.New("operations: invalid argument")
	// ErrNotFound — operation or account not found (or belongs to another user).
	ErrNotFound = errors.New("operations: not found")
	// ErrAccountMissing — operation references an account that doesn't exist.
	ErrAccountMissing = errors.New("operations: account not found")
)

// CreateInput is what a caller (HTTP handler) supplies. The service fills in
// derived fields: UserID (from auth), CalcID (if absent), masked PII, dates.
type CreateInput struct {
	Type            domain.OperationType
	Amount          domain.Money
	AmountDst       *domain.Money // currency_exchange only
	Currency        string
	AccountID       int64
	AccountDstID    *int64
	CategoryID      *int64
	IncomeSubtype   *domain.IncomeSubtype
	Counterparty    string
	Description     string
	OperationDate   time.Time // zero = today
	CalcID          string    // optional; generated if empty
}

// Create validates, masks PII, persists the operation, and recomputes the
// affected account balance(s). Idempotent on calc_id: a duplicate returns the
// existing operation unchanged.
//
// BUSINESS_LOGIC ф.1: transfers / currency exchanges / refunds are excluded
// from cashflow (handled by domain.AffectsCashflow), but they DO move money
// between accounts, so both legs' balances are recomputed.
func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (storage.Operation, error) {
	if userID == 0 {
		return storage.Operation{}, fmt.Errorf("%w: user_id required", ErrInvalidArgument)
	}
	now := s.now()
	date := in.OperationDate
	if date.IsZero() {
		date = now
	}
	isPlanned := date.After(now)

	op := storage.Operation{
		UserID:        userID,
		CalcID:        in.CalcID,
		Type:          in.Type,
		Amount:        in.Amount,
		AmountDst:     in.AmountDst,
		Currency:      in.Currency,
		AccountID:     in.AccountID,
		AccountDstID:  in.AccountDstID,
		CategoryID:    in.CategoryID,
		IncomeSubtype: in.IncomeSubtype,
		Counterparty:  pii.Mask(in.Counterparty),
		Description:   pii.Mask(in.Description),
		OperationDate: date,
		IsPlanned:     isPlanned,
	}
	if op.Currency == "" {
		op.Currency = "RUB"
	}
	if op.CalcID == "" {
		op.CalcID = newCalcID(userID, now)
	}
	if err := op.Validate(); err != nil {
		return storage.Operation{}, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	if op.AccountID <= 0 {
		return storage.Operation{}, fmt.Errorf("%w: account_id must be positive", ErrInvalidArgument)
	}

	// Verify account ownership up-front so we return a clean 400 instead of a
	// cryptic FK violation from the INSERT.
	if _, err := s.repo.GetAccount(ctx, userID, op.AccountID); err != nil {
		return storage.Operation{}, mapStorageErr(err, "source account")
	}
	if op.AccountDstID != nil {
		if _, err := s.repo.GetAccount(ctx, userID, *op.AccountDstID); err != nil {
			return storage.Operation{}, mapStorageErr(err, "destination account")
		}
	}

	// Auto-categorize when no category was supplied (BUSINESS_LOGIC ф.2). The
	// matcher sees the already-masked counterparty/description, never raw PII.
	// A categorizer failure must NOT block the write — log it and proceed with
	// whatever category we have (nil). Persistence of the money movement is
	// more important than a perfect category.
	if op.CategoryID == nil && s.categorizer != nil {
		catID, conf, err := s.categorizer.CategorizeForCreate(ctx, userID, op.Counterparty, op.Description)
		if err == nil && catID > 0 {
			op.CategoryID = &catID
			op.CategoryConfidence = conf
		}
	}

	created, err := s.repo.CreateOperation(ctx, op)
	if err != nil {
		if errors.Is(err, storage.ErrOperationExists) {
			// Idempotency: return the original.
			return s.repo.GetOperationByCalcID(ctx, userID, op.CalcID)
		}
		return storage.Operation{}, fmt.Errorf("operations: create: %w", err)
	}

	// Recompute affected balances from history (deterministic, no drift).
	if err := s.recomputeBalance(ctx, userID, created.AccountID); err != nil {
		return created, err
	}
	if created.AccountDstID != nil {
		if err := s.recomputeBalance(ctx, userID, *created.AccountDstID); err != nil {
			return created, err
		}
	}
	return created, nil
}

// Get returns one operation, scoped to the requesting user.
func (s *Service) Get(ctx context.Context, userID, id int64) (storage.Operation, error) {
	op, err := s.repo.GetOperation(ctx, userID, id)
	if err != nil {
		return storage.Operation{}, mapStorageErr(err, "operation")
	}
	return op, nil
}

// List returns a page of operations matching the filter. It also reports
// whether more rows exist beyond this page (the client uses this to decide
// whether to show "load more").
func (s *Service) List(ctx context.Context, userID int64, f storage.OperationFilter, page storage.Page) (items []storage.Operation, more bool, err error) {
	if page.Limit <= 0 {
		page.Limit = storage.DefaultPageLimit
	}
	// Fetch one extra to detect the next page without a separate COUNT query.
	page.Limit++
	items, err = s.repo.ListOperations(ctx, userID, f, page)
	if err != nil {
		return nil, false, fmt.Errorf("operations: list: %w", err)
	}
	if len(items) > 0 && len(items) == page.Limit {
		// Drop the overflow row; signal there is more.
		items = items[:len(items)-1]
		more = true
	}
	return items, more, nil
}

// Delete soft-deletes an operation and recomputes the affected balance(s).
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	op, err := s.repo.GetOperation(ctx, userID, id)
	if err != nil {
		return mapStorageErr(err, "operation")
	}
	if err := s.repo.DeleteOperation(ctx, userID, id); err != nil {
		return mapStorageErr(err, "operation")
	}
	if err := s.recomputeBalance(ctx, userID, op.AccountID); err != nil {
		return err
	}
	if op.AccountDstID != nil {
		if err := s.recomputeBalance(ctx, userID, *op.AccountDstID); err != nil {
			return err
		}
	}
	return nil
}

// SetCategory is the path used by auto-categorization confirmations and manual
// overrides. confidence=nil marks a user-set (authoritative) assignment.
func (s *Service) SetCategory(ctx context.Context, userID, id int64, categoryID *int64, confidence *decimal.Decimal) error {
	if err := s.repo.UpdateOperationCategory(ctx, userID, id, categoryID, confidence); err != nil {
		return mapStorageErr(err, "operation")
	}
	return nil
}

// recomputeBalance sets the cached balance of one account to the sum of all
// its operations' signed amounts. Doing a full recompute (vs delta-update)
// keeps the cache self-healing: any drift from a partial failure is corrected
// on the next operation touching that account.
func (s *Service) recomputeBalance(ctx context.Context, userID, accountID int64) error {
	if _, err := s.repo.GetAccount(ctx, userID, accountID); err != nil {
		return mapStorageErr(err, "account")
	}
	sum, err := s.repo.SumByAccountSince(ctx, accountID)
	if err != nil {
		return fmt.Errorf("operations: recompute balance: %w", err)
	}
	return s.repo.SetAccountBalance(ctx, userID, accountID, sum)
}

func mapStorageErr(err error, what string) error {
	switch {
	case errors.Is(err, storage.ErrOperationExists):
		return err // pass through to Create's idempotency path
	case errors.Is(err, storage.ErrOperationNotFound), errors.Is(err, storage.ErrAccountNotFound), errors.Is(err, storage.ErrCategoryNotFound):
		return fmt.Errorf("%w: %s", ErrNotFound, what)
	default:
		return fmt.Errorf("operations: %s: %w", what, err)
	}
}

// newCalcID generates a client-style idempotency key server-side when the
// caller did not provide one. We use a unix-nano timestamp + userID so that
// collisions require two requests in the same nanosecond for the same user.
func newCalcID(userID int64, now time.Time) string {
	return fmt.Sprintf("srv:%d:%d", userID, now.UnixNano())
}
