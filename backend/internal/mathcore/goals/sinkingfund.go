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
