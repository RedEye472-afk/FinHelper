package tax

// НДФЛ — Налог на доходы физических лиц (НК РФ Гл. 23).
// Progressive scale, encoded as ordered brackets in Rules.NDFL.Brackets:
//   - 2024: 2 brackets (13% up to 5M, 15% above)
//   - 2025+: 5 brackets (13% / 15% / 18% / 20% / 22%) per ФЗ-257 от 12.12.2024.
// Marginal: each bracket's rate applies only to the slice of taxable income
// that falls inside (prev up_to, this up_to].
//
// Source: MATH_FORMULAS.md §5.3; справочные_материалы/05_nalogovye_vychety_rf_2025_2026.md §0.

import (
	"github.com/shopspring/decimal"
)

// NDFL computes personal income tax for a year, accounting for deductions
// (child deductions, IIS, property — passed as a single aggregate).
//
// Method: taxable = max(0, income − deductions). The progressive scale is
// applied marginally across rules.NDFL.Brackets (falling back to the legacy
// 2-step HighIncomeThreshold/HighRate fields when Brackets is empty).
//
// Edge cases:
//   - income < 0 → ErrNegativeIncome
//   - deductions < 0 → ErrNegativeDeductions
//   - deductions ≥ income → tax = 0
//   - empty brackets → ErrUnknownNDFLScale (misconfigured year)
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

	brackets := rules.NDFL.NDFLBrackets()
	if len(brackets) == 0 {
		return decimal.Zero, ErrUnknownNDFLScale
	}

	tax := decimal.Zero
	remaining := taxable
	var prevUpTo decimal.Decimal // нижняя граница текущей ступени (0 для первой)
	for _, b := range brackets {
		if !remaining.IsPositive() {
			break
		}
		upTo := b.UpTo.Value()
		var slice decimal.Decimal
		if upTo.IsZero() {
			// Верхняя (бесконечная) ступень — забираем весь остаток.
			slice = remaining
		} else {
			// Ширина ступени = up_to − prevUpTo. Берём min(remaining, ширина).
			width := upTo.Sub(prevUpTo)
			if remaining.LessThanOrEqual(width) {
				slice = remaining
			} else {
				slice = width
			}
			prevUpTo = upTo
		}
		tax = tax.Add(slice.Mul(b.Rate.Value()))
		remaining = remaining.Sub(slice)
	}
	return tax.Round(2), nil
}

// Sentinel for NDFL scale misconfiguration.
var ErrUnknownNDFLScale = errUnknownNDFLScale{}

type errUnknownNDFLScale struct{}

func (errUnknownNDFLScale) Error() string { return "tax: ndfl progressive scale not configured" }
func (e errUnknownNDFLScale) Is(target error) bool {
	_, ok := target.(errUnknownNDFLScale)
	return ok
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
