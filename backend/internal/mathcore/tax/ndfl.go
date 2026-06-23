package tax

// НДФЛ — Налог на доходы физических лиц (НК РФ Гл. 23).
// Progressive scale: 13% up to 5M ₽/year, 15% on the portion above 5M.
//
// Source: MATH_FORMULAS.md §5.3.

import (
	"github.com/shopspring/decimal"
)

// NDFL computes personal income tax for a year, accounting for deductions
// (child deductions, IIS, property — passed as a single aggregate).
//
// Method: taxable = max(0, income − deductions). The first 5M of taxable is
// at BaseRate (13%); the excess above HighIncomeThreshold is at HighRate
// (15%).
//
// Edge cases:
//   - income < 0 → ErrNegativeIncome
//   - deductions < 0 → ErrNegativeDeductions
//   - deductions ≥ income → tax = 0
//
// Source: MATH_FORMULAS.md §5.3; НК РФ ст. 210, 224.
func NDFL(rules Rules, income, deductions decimal.Decimal) (decimal.Decimal, error) {
	if income.IsNegative() {
		return decimal.Zero, ErrNegativeIncome
	}
	if deductions.IsNegative() {
		return decimal.Zero, ErrNegativeDeductions
	}
	taxable := income.Sub(deductions)
	if !taxable.IsPositive() {
		return decimal.Zero, nil
	}

	threshold := rules.NDFL.HighIncomeThreshold.Value()
	if !taxable.GreaterThan(threshold) {
		// Entire taxable at the base rate.
		return taxable.Mul(rules.NDFL.BaseRate.Value()).Round(2), nil
	}
	// Split: threshold at base, excess at high.
	base := threshold.Mul(rules.NDFL.BaseRate.Value())
	excess := taxable.Sub(threshold).Mul(rules.NDFL.HighRate.Value())
	return base.Add(excess).Round(2), nil
}

// ChildDeduction returns the monthly child deduction given the number of
// children (НК РФ ст. 218): 1400 for each of the first two, 3000 for the
// third and each subsequent.
func ChildDeduction(rules Rules, children int) decimal.Decimal {
	if children <= 0 {
		return decimal.Zero
	}
	first := children
	if first > 2 {
		first = 2
	}
	rest := children - 2
	if rest < 0 {
		rest = 0
	}
	d1 := rules.NDFL.ChildDeduction1.Value().Mul(decimal.NewFromInt(int64(first)))
	d3 := rules.NDFL.ChildDeduction3.Value().Mul(decimal.NewFromInt(int64(rest)))
	return d1.Add(d3)
}
