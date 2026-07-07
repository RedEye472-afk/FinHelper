package credit

import (
	"github.com/shopspring/decimal"
)

// AnnuityPayment computes the level monthly payment
//
//	A = P × [i × (1+i)^n] / [(1+i)^n − 1]
//
// Variables:
//   - P principal
//   - monthlyRate i = annual_rate / 12
//   - n term in months
//
// Source: Копнова Г.П. Гл. 4.2.1; MATH_FORMULAS.md §2.1.
//
// Edge cases:
//   - n <= 0 → ErrInvalidTerm
//   - i = 0 → fallback A = P / n (interest-free loan)
//   - P = 0 → 0
func AnnuityPayment(principal, monthlyRate decimal.Decimal, termMonths int) (decimal.Decimal, error) {
	if termMonths <= 0 {
		return decimal.Zero, ErrInvalidTerm
	}
	if principal.IsNegative() {
		return decimal.Zero, ErrInvalidPrincipal
	}
	if principal.IsZero() {
		return decimal.Zero, nil
	}
	if monthlyRate.IsZero() {
		// Interest-free: equal principal fractions.
		return principal.Div(decimal.NewFromInt(int64(termMonths))), nil
	}
	one := decimal.NewFromInt(1)
	growth := one.Add(monthlyRate)          // (1+i)
	factor := growth.Pow(decimal.NewFromInt(int64(termMonths))) // (1+i)^n
	num := monthlyRate.Mul(factor)
	denom := factor.Sub(one)
	return principal.Mul(num).Div(denom), nil
}

// AnnuitySchedule builds the full payment schedule for an annuity loan.
// Each row contains principal repaid, interest paid, and outstanding
// balance AFTER the payment. The last row absorbs the rounding residual so
// the balance closes exactly to zero — this matters for PSK validation.
//
// monthlyPayment must equal AnnuityPayment(principal, monthlyRate, term);
// pass it explicitly so callers can round once and reuse.
//
// Source: Копнова Г.П. Гл. 4.2.1; rounding to scale 2 (kopecks).
func AnnuitySchedule(principal, monthlyRate, monthlyPayment decimal.Decimal, termMonths int) ([]ScheduleRow, error) {
	if termMonths <= 0 {
		return nil, ErrInvalidTerm
	}
	rows := make([]ScheduleRow, 0, termMonths)
	balance := principal
	scale := int32(2)
	for k := 1; k <= termMonths; k++ {
		interest := balance.Mul(monthlyRate).Round(scale)
		principalPart := monthlyPayment.Sub(interest).Round(scale)
		// On the final payment, absorb any residual (±1 kopeck from rounding
		// each row) so the balance closes exactly to zero.
		if k == termMonths {
			principalPart = balance.Round(scale)
			monthlyPayment = principalPart.Add(interest).Round(scale)
		}
		balance = balance.Sub(principalPart).Round(scale)
		if balance.IsNegative() {
			balance = decimal.Zero
		}
		rows = append(rows, ScheduleRow{
			Month:      k,
			Payment:    monthlyPayment,
			Principal:  principalPart,
			Interest:   interest,
			BalanceEnd: balance,
		})
	}
	return rows, nil
}

// ScheduleRow is one row of a loan amortisation schedule.
// All amounts are rounded to MoneyScale (2 = kopecks).
type ScheduleRow struct {
	Month      int
	Payment    decimal.Decimal // total payment this month
	Principal  decimal.Decimal // principal portion
	Interest   decimal.Decimal // interest portion
	BalanceEnd decimal.Decimal // outstanding balance after this payment
}
