// Package tvm implements Time-Value-of-Money formulas (simple interest,
// compound interest, effective rate, Fisher real rate).
//
// Design principle (CLAUDE.md §1 "Детерминизм"): all calculations use
// decimal.Decimal — never float64. Source: Копнова Г.П. "Финансовая
// математика" Гл. 1; MATH_FORMULAS.md §1.
package tvm

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// Sentinel errors.
var (
	ErrNegativeTime          = errors.New("tvm: time must be >= 0")
	ErrInvalidCompounding    = errors.New("tvm: compounding periods per year must be > 0")
	ErrDeflation100Percent   = errors.New("tvm: inflation = -1 would divide by zero")
	ErrNegativePrincipalRate = errors.New("tvm: inputs must be non-negative where applicable")
)

// SimpleInterest computes S = P × (1 + i × t).
//
// Formula:   S = P × (1 + i × t)
// Variables: P principal, i annual rate (fraction), t time in years.
// Source:    Копнова Г.П. Гл. 1.1; MATH_FORMULAS.md §1.1.
//
// Edge cases:
//   - t = 0 → S = P
//   - i = 0 → S = P
//   - t < 0 → ErrNegativeTime
func SimpleInterest(principal, annualRate, years decimal.Decimal) (decimal.Decimal, error) {
	if years.IsNegative() {
		return decimal.Zero, ErrNegativeTime
	}
	// 1 + i*t
	factor := decimal.NewFromInt(1).Add(annualRate.Mul(years))
	return principal.Mul(factor), nil
}

// CompoundInterest computes S = P × (1 + i/m)^(m×t).
//
// Formula:   S = P × (1 + i/m)^(m×t)
// Variables: P principal, i annual rate, m compounding periods per year,
//            t time in years.
// Source:    Копнова Г.П. Гл. 1.2; MATH_FORMULAS.md §1.2.
//
// Implementation note: decimal.Pow supports fractional exponents (via
// Ln+Exp internally), so fractional years (e.g. t=1.5) work. On undefined
// inputs (e.g. negative base with fractional exponent) decimal.Pow returns
// its zero value — we guard the common cases here.
//
// Edge cases:
//   - m <= 0 → ErrInvalidCompounding
//   - t = 0 → S = P (exponent 0, base^0 = 1)
//   - i = 0 → S = P
//   - t < 0 → ErrNegativeTime
func CompoundInterest(principal, annualRate decimal.Decimal, compoundingPerYear int, years decimal.Decimal) (decimal.Decimal, error) {
	if compoundingPerYear <= 0 {
		return decimal.Zero, ErrInvalidCompounding
	}
	if years.IsNegative() {
		return decimal.Zero, ErrNegativeTime
	}
	m := decimal.NewFromInt(int64(compoundingPerYear))
	// (1 + i/m)
	base := decimal.NewFromInt(1).Add(annualRate.Div(m))
	// exponent = m × t
	exp := m.Mul(years)
	pow := base.Pow(exp)
	// decimal.Pow returns Decimal{} on undefined input. Treat that as an error
	// so callers don't silently get a zero result.
	if pow.IsZero() && !base.IsZero() {
		return decimal.Zero, fmt.Errorf("tvm: compound interest undefined for base=%s exp=%s", base, exp)
	}
	return principal.Mul(pow), nil
}

// EffectiveRate computes i_eff = (1 + i_nom/m)^m − 1.
//
// Formula:   i_eff = (1 + i_nom/m)^m − 1
// Source:    Копнова Г.П. Гл. 1.3; MATH_FORMULAS.md §1.3.
//
// Use to compare deposits with different compounding frequencies: a 10%
// nominal rate compounded monthly yields ~10.47% effective annually.
//
// Edge cases:
//   - m <= 0 → ErrInvalidCompounding
//   - i_nom = 0 → 0
func EffectiveRate(nominalRate decimal.Decimal, compoundingPerYear int) (decimal.Decimal, error) {
	if compoundingPerYear <= 0 {
		return decimal.Zero, ErrInvalidCompounding
	}
	m := decimal.NewFromInt(int64(compoundingPerYear))
	base := decimal.NewFromInt(1).Add(nominalRate.Div(m))
	pow := base.Pow(m)
	return pow.Sub(decimal.NewFromInt(1)), nil
}

// FisherRealRate computes r_real = (1 + r_nom) / (1 + π) − 1.
//
// Formula:   r_real = (1 + r_nom) / (1 + π) − 1
// Variables: r_nom nominal yield, π inflation (fraction).
// Source:    Копнова Г.П. Гл. 1.4; MATH_FORMULAS.md §1.4 (Fisher equation).
//
// Edge cases:
//   - π = −1 → ErrDeflation100Percent (division by zero; 100% deflation is
//     economically impossible but mathematically needs guarding)
//   - π > r_nom → negative real yield (money losing value) — valid, not an error
func FisherRealRate(nominalRate, inflation decimal.Decimal) (decimal.Decimal, error) {
	one := decimal.NewFromInt(1)
	denom := one.Add(inflation)
	if denom.IsZero() {
		return decimal.Zero, ErrDeflation100Percent
	}
	return one.Add(nominalRate).Div(denom).Sub(one), nil
}
