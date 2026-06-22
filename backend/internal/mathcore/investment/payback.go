package investment

// DPP (Discounted Payback Period) and PI (Profitability Index).
// Both fully on decimal.Decimal — no float bridge.

import (
	"github.com/shopspring/decimal"
)

// DPP computes the Discounted Payback Period — the fractional period at
// which the cumulative discounted cashflow first reaches zero.
//
// Method: walk the cashflows accumulating CF_t / (1+r)^t. When the running
// sum crosses from negative to positive between t-1 and t, linearly
// interpolate the fraction of period t needed to recover the remaining
// shortfall.
//
// Edge cases:
//   - r <= -1 → ErrInvalidRate
//   - empty cashflows → ErrNeverPaidBack
//   - cumulative sum never reaches 0 → ErrNeverPaidBack
//   - first flow already positive → DPP = 0 (paid back immediately)
//
// Source: MATH_FORMULAS.md §3.4.
func DPP(cashflows []decimal.Decimal, rate decimal.Decimal) (decimal.Decimal, error) {
	if len(cashflows) == 0 {
		return decimal.Zero, ErrNeverPaidBack
	}
	if !rate.GreaterThan(decimal.NewFromInt(-1)) {
		return decimal.Zero, ErrInvalidRate
	}
	one := decimal.NewFromInt(1)
	base := one.Add(rate)

	cumulative := decimal.Zero
	for t, cf := range cashflows {
		disc := cf.Div(base.Pow(decimal.NewFromInt(int64(t))))
		prev := cumulative
		cumulative = cumulative.Add(disc)

		if !prev.IsPositive() && !cumulative.IsNegative() {
			// Crossed zero at this period.
			if t == 0 {
				// The first flow alone brings cumulative to >= 0: paid back
				// immediately (no time elapses).
				return decimal.Zero, nil
			}
			// Interpolate the fraction of period t needed to close the
			// remaining shortfall from the prior cumulative.
			if disc.IsZero() {
				return decimal.NewFromInt(int64(t)), nil
			}
			// Fraction = |prev| / disc (the share of period t's discounted
			// flow needed to close the prior shortfall).
			frac := prev.Abs().Div(disc.Abs())
			return decimal.NewFromInt(int64(t - 1)).Add(frac), nil
		}
	}
	return decimal.Zero, ErrNeverPaidBack
}

// PI computes the Profitability Index:
//
//	PI = PV(future positive CF) / |initial investment|
//
// The initial investment is cashflows[0] (typically negative). PV is taken
// at the given per-period discount rate.
//
// Decision rule: PI > 1 → profitable, PI < 1 → unprofitable.
//
// Edge cases:
//   - empty cashflows → ErrInsufficientCashflows
//   - cashflows[0] == 0 → ErrZeroInitialInvestment
//   - r <= -1 → ErrInvalidRate
//
// Source: MATH_FORMULAS.md §3.5.
func PI(cashflows []decimal.Decimal, rate decimal.Decimal) (decimal.Decimal, error) {
	if len(cashflows) == 0 {
		return decimal.Zero, ErrInsufficientCashflows
	}
	if !rate.GreaterThan(decimal.NewFromInt(-1)) {
		return decimal.Zero, ErrInvalidRate
	}
	initial := cashflows[0]
	if initial.IsZero() {
		return decimal.Zero, ErrZeroInitialInvestment
	}
	one := decimal.NewFromInt(1)
	base := one.Add(rate)

	pvFuture := decimal.Zero
	for t := 1; t < len(cashflows); t++ {
		cf := cashflows[t]
		if cf.IsPositive() {
			disc := cf.Div(base.Pow(decimal.NewFromInt(int64(t))))
			pvFuture = pvFuture.Add(disc)
		}
	}
	return pvFuture.Div(initial.Abs()), nil
}
