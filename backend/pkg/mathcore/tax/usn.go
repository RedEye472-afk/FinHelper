package tax

// УСН — Упрощённая система налогообложения (НК РФ Гл. 26.2).
//   - USNIncome: 6% of revenue
//   - USNIncomeMinusExpenses: 15% of (revenue − expenses), ≥ 0
//
// Source: MATH_FORMULAS.md §5.2.

import (
	"github.com/shopspring/decimal"
)

// USN computes the small-business tax under the chosen regime.
//
// Edge cases:
//   - revenue < 0 → ErrNegativeIncome
//   - expenses < 0 → ErrNegativeExpenses
//   - expenses > revenue (USNIncomeMinusExpenses) → tax = 0 (no refund)
//
// Source: MATH_FORMULAS.md §5.2.
func USN(rules Rules, regime USNRegime, revenue, expenses decimal.Decimal) (decimal.Decimal, error) {
	if revenue.IsNegative() {
		return decimal.Zero, ErrNegativeIncome
	}
	if expenses.IsNegative() {
		return decimal.Zero, ErrNegativeExpenses
	}
	switch regime {
	case USNIncome:
		return revenue.Mul(rules.USN.RateIncome.Value()).Round(2), nil
	case USNIncomeMinusExpenses:
		profit := revenue.Sub(expenses)
		if !profit.IsPositive() {
			return decimal.Zero, nil
		}
		return profit.Mul(rules.USN.RateIncomeMinusExpenses.Value()).Round(2), nil
	default:
		return decimal.Zero, ErrUnknownUSNRegime
	}
}
