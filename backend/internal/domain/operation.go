package domain

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// OperationType is the kind of financial movement.
// See BUSINESS_LOGIC.md ф.1 — transfers/exchanges/refunds MUST NOT
// appear in cashflow or tax calculations, only in account balances.
type OperationType string

const (
	OpIncome           OperationType = "income"
	OpExpense          OperationType = "expense"
	OpTransfer         OperationType = "transfer"
	OpCurrencyExchange OperationType = "currency_exchange"
	OpRefund           OperationType = "refund"
)

// IncomeSubtype classifies income for analytics and tax (BUSINESS_LOGIC ф.1).
type IncomeSubtype string

const (
	IncomeSalary        IncomeSubtype = "salary"
	IncomeFee           IncomeSubtype = "fee"
	IncomeGift          IncomeSubtype = "gift"
	IncomeInvestment    IncomeSubtype = "investment"
	IncomeLoanRepayment IncomeSubtype = "loan_repayment"
)

// negOne is a reusable factor to flip the sign of a Money amount.
var negOne = decimal.NewFromInt(-1)

// Operation is a financial transaction.
//
// INVARIANTS:
//   - Amount is always positive; sign is derived from OperationType.
//   - Transfers/exchanges require AccountDstID != nil.
//   - CalcID is unique per user → idempotency key.
//   - Counterparty is PII-masked before persistence (PRIVACY_RULES.md).
type Operation struct {
	ID                 int64
	UserID             int64
	CalcID             string
	Type               OperationType
	Amount             Money
	AmountDst          *Money           // non-nil only for currency_exchange
	Currency           string
	AccountID          int64
	AccountDstID       *int64
	CategoryID         *int64
	IncomeSubtype      *IncomeSubtype
	Counterparty       string
	Description        string
	OperationDate      time.Time
	IsPlanned          bool
	CategoryConfidence *decimal.Decimal // 0..1, nil if user-set manually
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// AffectsCashflow reports whether this operation contributes to net cashflow.
// Transfers, currency exchanges, and refunds are internal movements —
// they MUST be excluded from income/expense totals (BUSINESS_LOGIC.md ф.1).
func (o Operation) AffectsCashflow() bool {
	switch o.Type {
	case OpTransfer, OpCurrencyExchange, OpRefund:
		return false
	default:
		return true
	}
}

// SignedAmount returns the amount with the sign appropriate to its type:
// income/refund → positive, expense → negative, transfer/exchange → zero.
// Use only for cashflow aggregation; balances are updated by the service layer.
func (o Operation) SignedAmount() Money {
	switch o.Type {
	case OpExpense:
		return o.Amount.Mul(negOne)
	case OpIncome, OpRefund:
		return o.Amount
	default:
		return Zero
	}
}

// Validate enforces the operation invariants.
func (o Operation) Validate() error {
	if o.UserID == 0 {
		return errors.New("operation: user_id is required")
	}
	if o.CalcID == "" {
		return errors.New("operation: calc_id is required")
	}
	if !o.Amount.IsPositive() {
		return errors.New("operation: amount must be positive")
	}
	switch o.Type {
	case OpIncome, OpExpense, OpTransfer, OpCurrencyExchange, OpRefund:
	default:
		return errors.New("operation: invalid operation_type")
	}
	if o.Type == OpTransfer || o.Type == OpCurrencyExchange {
		if o.AccountDstID == nil || *o.AccountDstID == 0 {
			return errors.New("operation: transfer/exchange requires destination account")
		}
		if *o.AccountDstID == o.AccountID {
			return errors.New("operation: source and destination accounts must differ")
		}
	}
	if o.OperationDate.IsZero() {
		return errors.New("operation: operation_date is required")
	}
	if o.CategoryConfidence != nil {
		if o.CategoryConfidence.LessThan(decimal.Zero) || o.CategoryConfidence.GreaterThan(decimal.NewFromInt(1)) {
			return errors.New("operation: category_confidence must be in [0, 1]")
		}
	}
	return nil
}
