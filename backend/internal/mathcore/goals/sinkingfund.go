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

// SolveTerm computes the number of periods needed to reach target S with
// present capital P, periodic contribution A, and per-period rate i:
//
//	n = ln((S·i + A) / (A + P·i)) / ln(1+i)
//
// Returns a decimal (caller rounds up to whole periods).
// Source: Копнова Г.П. Гл. 3.3.3 (inverse of SolveFutureValue for n).
//
// Edge cases:
//   - i <= 0  → ErrInvalidRate (Ln requires 1+i > 0 AND != 1)
//   - P >= S  → returns 0 (already reached)
//   - A <= P·i → ErrUnreachable (contribution doesn't outpace interest growth
//     of present capital; the target recedes instead of approaching)
func SolveTerm(P, S, A, i decimal.Decimal) (decimal.Decimal, error) {
	if !i.IsPositive() {
		return decimal.Zero, ErrInvalidRate
	}
	if P.GreaterThanOrEqual(S) {
		return decimal.Zero, nil
	}
	// Unreachable: вклады A не превышают рост P·i → S убегает.
	if A.LessThanOrEqual(P.Mul(i)) {
		return decimal.Zero, ErrUnreachable
	}
	one := decimal.NewFromInt(1)
	num := S.Mul(i).Add(A)   // S·i + A
	denom := A.Add(P.Mul(i)) // A + P·i
	ratio := num.Div(denom)  // должно быть > 1 (проверено выше)
	// ratio = (1+i)^n → n = ln(ratio)/ln(1+i)
	// shopspring/decimal v1.4.0: Ln(precision int32) → (Decimal, error)
	// (в отличие от Pow, который возвращает Decimal без error). Precision 16
	// = decimal.DivisionPrecision по умолчанию.
	const lnPrecision = 16
	lnRatio, err := ratio.Ln(lnPrecision)
	if err != nil {
		return decimal.Zero, err
	}
	lnBase, err := one.Add(i).Ln(lnPrecision)
	if err != nil {
		return decimal.Zero, err
	}
	return lnRatio.Div(lnBase), nil
}
