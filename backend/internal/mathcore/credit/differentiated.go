package credit

import (
	"github.com/shopspring/decimal"
)

// DifferentiatedPayment computes the k-th payment of a differentiated loan
// (declining payments: principal portion is constant P/n, interest declines).
//
//	Платёж_k = P/n + (P − Σ_{j=1}^{k-1} P/n) × i
//
// Variables:
//   - P principal
//   - monthlyRate i = annual_rate / 12
//   - n term in months
//   - k month number, 1-based
//
// Source: Копнова Г.П. Гл. 4.2.2; MATH_FORMULAS.md §2.2.
//
// Edge cases:
//   - n <= 0 → ErrInvalidTerm
//   - k < 1 || k > n → ErrInvalidMonth
//   - i = 0 → all payments equal P/n
//   - P = 0 → 0
//
// For a full schedule prefer DifferentiatedSchedule, which keeps the running
// balance and rounds consistently with MoneyScale.
func DifferentiatedPayment(principal, monthlyRate decimal.Decimal, termMonths, month int) (decimal.Decimal, error) {
	if termMonths <= 0 {
		return decimal.Zero, ErrInvalidTerm
	}
	if month < 1 || month > termMonths {
		return decimal.Zero, ErrInvalidMonth
	}
	if principal.IsNegative() {
		return decimal.Zero, ErrInvalidPrincipal
	}
	n := decimal.NewFromInt(int64(termMonths))
	principalPart := principal.Div(n).Round(2)
	// Remaining principal before month k: P − (k−1) × P/n.
	paidPrincipal := principalPart.Mul(decimal.NewFromInt(int64(month - 1)))
	remaining := principal.Sub(paidPrincipal)
	interest := remaining.Mul(monthlyRate).Round(2)
	return principalPart.Add(interest), nil
}

// DifferentiatedSchedule builds the full schedule of a differentiated loan.
// Each row carries principal repaid (constant), interest (declining), and
// the outstanding balance after the payment.
//
// Source: Копнова Г.П. Гл. 4.2.2; rounding to scale 2 (kopecks).
func DifferentiatedSchedule(principal, monthlyRate decimal.Decimal, termMonths int) ([]ScheduleRow, error) {
	if termMonths <= 0 {
		return nil, ErrInvalidTerm
	}
	n := decimal.NewFromInt(int64(termMonths))
	principalPart := principal.Div(n).Round(2)
	balance := principal
	rows := make([]ScheduleRow, 0, termMonths)
	for k := 1; k <= termMonths; k++ {
		interest := balance.Mul(monthlyRate).Round(2)
		payPrincipal := principalPart
		// Final-row residual absorb (rounding may leave ±1 kopeck).
		if k == termMonths {
			payPrincipal = balance.Round(2)
		}
		payment := payPrincipal.Add(interest).Round(2)
		balance = balance.Sub(payPrincipal).Round(2)
		if balance.IsNegative() {
			balance = decimal.Zero
		}
		rows = append(rows, ScheduleRow{
			Month:      k,
			Payment:    payment,
			Principal:  payPrincipal,
			Interest:   interest,
			BalanceEnd: balance,
		})
	}
	return rows, nil
}
