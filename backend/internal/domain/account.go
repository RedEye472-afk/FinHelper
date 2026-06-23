package domain

import (
	"errors"
	"time"
)

// AccountType is the kind of account a user holds (BUSINESS_LOGIC.md ф.1).
type AccountType string

const (
	AccountCash        AccountType = "cash"
	AccountBank        AccountType = "bank"
	AccountSavings     AccountType = "savings"
	AccountInvestment  AccountType = "investment"
	AccountCrypto      AccountType = "crypto"
	AccountDebt         AccountType = "debt"
)

// Account is a user's financial account. Balances are cached on the row for
// dashboard performance and recomputed on every operation insert/update/delete
// by the service layer (see service/operations).
//
// INVARIANT: Currency is always RUB in MVP (CLAUDE.md scope-lock). The field
// remains so multi-currency can be added without a schema migration.
type Account struct {
	ID          int64
	UserID      int64
	Name        string
	Type        AccountType
	Currency    string
	Balance     Money
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate enforces the account invariants.
func (a Account) Validate() error {
	if a.UserID == 0 {
		return errors.New("account: user_id is required")
	}
	if a.Name == "" {
		return errors.New("account: name is required")
	}
	switch a.Type {
	case AccountCash, AccountBank, AccountSavings, AccountInvestment, AccountCrypto, AccountDebt:
	default:
		return errors.New("account: invalid account_type")
	}
	if a.Currency == "" {
		return errors.New("account: currency is required")
	}
	return nil
}
