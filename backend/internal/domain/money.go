// Package domain defines the core business types of FinHelper.
//
// Design principle (CLAUDE.md §1 "Детерминизм"): all monetary calculations
// MUST use decimal.Decimal — never float64. Money wraps decimal.Decimal with
// a fixed scale of 2 (kopecks), so rounding is explicit and auditable.
package domain

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Money represents a monetary amount in minor units (kopecks for RUB).
// Scale is fixed at 2 decimal places to match database NUMERIC(28,2).
//
// Use the constructor NewMoney to enforce scale. Never create Money from
// float64 — that loses precision before we even start.
type Money struct {
	v decimal.Decimal
}

// MoneyScale is the number of decimal places for all monetary values.
const MoneyScale = 2

// Zero is the zero-value Money.
var Zero = Money{v: decimal.Zero}

// NewMoney creates a Money value from a decimal.Decimal, rounding to
// MoneyScale using bankers' rounding (HALF_EVEN) — the financial standard.
func NewMoney(d decimal.Decimal) Money {
	return Money{v: d.Round(MoneyScale)}
}

// MustParseMoney parses a string like "47073.46" into Money.
// Panics on malformed input — use for compile-time constants only.
// For runtime parsing use ParseMoney.
func MustParseMoney(s string) Money {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(fmt.Sprintf("domain: invalid money literal %q: %v", s, err))
	}
	return NewMoney(d)
}

// ParseMoney parses a string into Money, returning an error on failure.
// Accepts formats: "47073.46", "47073,46" (ru), " 47 073.46 " (with spaces).
func ParseMoney(s string) (Money, error) {
	s = strings.ReplaceAll(s, ",", ".")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u00a0", "") // non-breaking space
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return Zero, fmt.Errorf("domain: invalid money %q: %w", s, err)
	}
	return NewMoney(d), nil
}

// FromInt creates Money from an integer value (assumed already in major units).
func FromInt(n int64) Money {
	return Money{v: decimal.NewFromInt(n).Round(MoneyScale)}
}

// String formats Money as "47073.46".
func (m Money) String() string {
	return m.v.StringFixed(MoneyScale)
}

// Decimal returns the underlying decimal.Decimal (already at scale 2).
// Exposed for the mathcore package; outside mathcore prefer staying in Money.
func (m Money) Decimal() decimal.Decimal { return m.v }

// Add returns a + b.
func (m Money) Add(b Money) Money { return Money{v: m.v.Add(b.v)} }

// Sub returns a - b (can be negative; use IsNegative to check).
func (m Money) Sub(b Money) Money { return Money{v: m.v.Sub(b.v)} }

// Mul returns a * factor (factor is a pure number, not Money).
// Result is rounded to MoneyScale.
func (m Money) Mul(factor decimal.Decimal) Money {
	return Money{v: m.v.Mul(factor).Round(MoneyScale)}
}

// Cmp returns -1, 0, +1 for less, equal, greater.
func (m Money) Cmp(b Money) int { return m.v.Cmp(b.v) }

// IsNegative reports whether the amount is negative.
func (m Money) IsNegative() bool { return m.v.IsNegative() }

// IsZero reports whether the amount equals zero.
func (m Money) IsZero() bool { return m.v.IsZero() }

// IsPositive reports whether the amount is greater than zero.
func (m Money) IsPositive() bool { return m.v.IsPositive() }
