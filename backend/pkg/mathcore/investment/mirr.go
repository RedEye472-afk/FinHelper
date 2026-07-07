package investment

// MIRR — Modified Internal Rate of Return. Unlike IRR, MIRR assumes
// positive cashflows are reinvested at `reinvestRate` and negative flows
// are financed at `financeRate`, avoiding the multi-root problem of IRR.
//
//	MIRR = (FV_positive_CF / PV_negative_CF)^(1/n) − 1
//
// where n is the number of periods (len(cashflows) − 1), FV compounds
// positive flows forward to the terminal period at reinvestRate, and PV
// discounts negative flows back to period 0 at financeRate.
//
// Fully on decimal.Decimal (no float bridge — only IRR/XIRR need the solver).
//
// Source: Ширяев А.Н. "Финансовая математика"; MATH_FORMULAS.md §3.3.

import (
	"github.com/shopspring/decimal"
)

// MIRR computes the Modified IRR for a periodic (equal-length) cashflow
// series.
//
// Edge cases:
//   - < 2 cashflows → ErrInsufficientCashflows
//   - no positive CF → ErrNoPositiveCF
//   - no negative CF → ErrNoNegativeCF
//   - reinvestRate or financeRate <= -1 → ErrInvalidRate
//
// Source: MATH_FORMULAS.md §3.3.
func MIRR(cashflows []decimal.Decimal, financeRate, reinvestRate decimal.Decimal) (decimal.Decimal, error) {
	if len(cashflows) < 2 {
		return decimal.Zero, ErrInsufficientCashflows
	}
	if !financeRate.GreaterThan(decimal.NewFromInt(-1)) || !reinvestRate.GreaterThan(decimal.NewFromInt(-1)) {
		return decimal.Zero, ErrInvalidRate
	}
	n := len(cashflows) - 1 // number of periods

	hasPos, hasNeg := false, false
	for _, cf := range cashflows {
		if cf.IsPositive() {
			hasPos = true
		}
		if cf.IsNegative() {
			hasNeg = true
		}
	}
	if !hasPos {
		return decimal.Zero, ErrNoPositiveCF
	}
	if !hasNeg {
		return decimal.Zero, ErrNoNegativeCF
	}

	one := decimal.NewFromInt(1)

	// FV of positive flows: compound each positive CF forward from its
	// period t to the terminal period n at reinvestRate.
	fvPos := decimal.Zero
	for t, cf := range cashflows {
		if cf.IsPositive() {
			periodsLeft := int64(n - t)
			fvPos = fvPos.Add(cf.Mul(one.Add(reinvestRate).Pow(decimal.NewFromInt(periodsLeft))))
		}
	}

	// PV of negative flows: discount each negative CF back to period 0 at
	// financeRate. Use the absolute value for the denominator ratio.
	pvNeg := decimal.Zero
	for t, cf := range cashflows {
		if cf.IsNegative() {
			pvNeg = pvNeg.Add(cf.Mul(one.Add(financeRate).Pow(decimal.NewFromInt(int64(t)))))
		}
	}
	pvNeg = pvNeg.Abs()

	// MIRR = (fvPos / pvNeg)^(1/n) − 1.
	ratio := fvPos.Div(pvNeg)
	root := ratio.Pow(decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(n))))
	return root.Sub(one), nil
}
