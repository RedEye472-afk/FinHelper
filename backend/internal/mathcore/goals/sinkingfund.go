package goals

import "github.com/shopspring/decimal"

// SolveFutureValue computes the accumulated amount after n periods:
//
//	S = P·(1+i)^n + A·((1+i)^n − 1)/i
//
// Variables: P present capital, A periodic contribution, i per-period rate,
// n number of periods.
// Source: Копнова Г.П. Гл. 3.3.3; MATH_FORMULAS.md §2.1.
//
// Edge cases:
//   - i = 0  → linear fallback S = P + A·n
//   - n = 0  → S = P
//   - n < 0  → ErrInvalidPeriods
func SolveFutureValue(P, A, i decimal.Decimal, n int) (decimal.Decimal, error) {
	if n < 0 {
		return decimal.Zero, ErrInvalidPeriods
	}
	if n == 0 {
		return P, nil
	}
	if i.IsZero() {
		// Без процентов: просто сумма взносов плюс исходный капитал.
		return P.Add(A.Mul(decimal.NewFromInt(int64(n)))), nil
	}
	one := decimal.NewFromInt(1)
	factor := one.Add(i).Pow(decimal.NewFromInt(int64(n))) // (1+i)^n
	principalGrown := P.Mul(factor)
	annuityPart := A.Mul(factor.Sub(one)).Div(i)
	return principalGrown.Add(annuityPart), nil
}

// SolveContribution computes the periodic contribution needed to reach target
// S after n periods, given present capital P and per-period rate i:
//
//	A = (S − P·(1+i)^n) · i / ((1+i)^n − 1)
//
// Source: Копнова Г.П. Гл. 3.3.3 (inverse of SolveFutureValue).
//
// Edge cases:
//   - i = 0  → linear fallback A = (S − P)/n
//   - n <= 0 → ErrInvalidPeriods
//   - S <= P·(1+i)^n → returns 0 (target already reached by capital growth)
func SolveContribution(P, S, i decimal.Decimal, n int) (decimal.Decimal, error) {
	if n <= 0 {
		return decimal.Zero, ErrInvalidPeriods
	}
	if i.IsZero() {
		return S.Sub(P).Div(decimal.NewFromInt(int64(n))), nil
	}
	one := decimal.NewFromInt(1)
	factor := one.Add(i).Pow(decimal.NewFromInt(int64(n)))
	grown := P.Mul(factor)
	if !S.GreaterThan(grown) {
		// Цель уже достигнута ростом капитала — взнос не требуется.
		return decimal.Zero, nil
	}
	need := S.Sub(grown)
	return need.Mul(i).Div(factor.Sub(one)), nil
}
