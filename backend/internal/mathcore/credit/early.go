package credit

// Early repayment scenarios: given an outstanding balance on an annuity
// loan, an extra principal payment either shortens the remaining term
// (same payment) or lowers the payment (same term). Recomputes the
// remaining schedule on the reduced balance.
//
// Source: MATH_FORMULAS.md §2.4; Копнова Г.П. Гл. 4.3.

import (
	"github.com/shopspring/decimal"
)

// EarlyRepaymentMode selects how an early payment is applied.
type EarlyRepaymentMode int

const (
	// EarlyShortenTerm keeps the monthly payment, reduces remaining months.
	EarlyShortenTerm EarlyRepaymentMode = iota
	// EarlyLowerPayment keeps the term, reduces the monthly payment.
	EarlyLowerPayment
)

// EarlyRepaymentResult summarises a single early-repayment scenario.
type EarlyRepaymentResult struct {
	NewPayment       decimal.Decimal // monthly payment after the change
	NewRemainingTerm int             // months left after the early payment
	InterestSaved    decimal.Decimal // interest saved vs. the original schedule
	NewSchedule      []ScheduleRow   // amortisation from the next month onward
}

// EarlyRepayment recomputes a loan after a lump-sum principal payment at the
// end of `paidMonths`. The outstanding balance at that point is reduced by
// `earlyAmount`; the loan is then re-amortised per `mode`.
//
// Inputs:
//   - principal, annualRate: original loan terms.
//   - termMonths: original term.
//   - paidMonths: number of months already paid (0 ≤ paidMonths ≤ termMonths).
//   - earlyAmount: extra principal paid at month `paidMonths` (≥ 0).
//   - mode: shorten-term or lower-payment.
//
// Edge cases:
//   - earlyAmount == 0 → returns the remaining original schedule (no change).
//   - earlyAmount ≥ outstanding balance → loan closed; NewRemainingTerm = 0.
//   - earlyAmount < 0 → ErrInvalidEarlyAmount.
//   - earlyAmount > balance → ErrEarlyExceedsBalance.
//
// Source: MATH_FORMULAS.md §2.4.
func EarlyRepayment(
	principal, annualRate decimal.Decimal,
	termMonths, paidMonths int,
	earlyAmount decimal.Decimal,
	mode EarlyRepaymentMode,
) (EarlyRepaymentResult, error) {
	if termMonths <= 0 {
		return EarlyRepaymentResult{}, ErrInvalidTerm
	}
	if paidMonths < 0 || paidMonths > termMonths {
		return EarlyRepaymentResult{}, ErrInvalidMonth
	}
	if earlyAmount.IsNegative() {
		return EarlyRepaymentResult{}, ErrInvalidEarlyAmount
	}

	monthlyRate := annualRate.Div(decimal.NewFromInt(12))
	origPayment, err := AnnuityPayment(principal, monthlyRate, termMonths)
	if err != nil {
		return EarlyRepaymentResult{}, err
	}
	origSchedule, err := AnnuitySchedule(principal, monthlyRate, origPayment, termMonths)
	if err != nil {
		return EarlyRepaymentResult{}, err
	}

	// Outstanding balance at the end of paidMonths (0 if none paid yet).
	balance := principal
	if paidMonths > 0 {
		balance = origSchedule[paidMonths-1].BalanceEnd
	}

	if earlyAmount.GreaterThan(balance) {
		return EarlyRepaymentResult{}, ErrEarlyExceedsBalance
	}
	newBalance := balance.Sub(earlyAmount).Round(2)

	res := EarlyRepaymentResult{}

	// Early payoff closes the loan.
	if newBalance.IsZero() {
		// Interest saved = original interest on remaining months.
		origRemainingInterest := sumInterest(origSchedule[paidMonths:])
		res.NewPayment = decimal.Zero
		res.NewRemainingTerm = 0
		res.InterestSaved = origRemainingInterest
		res.NewSchedule = nil
		return res, nil
	}

	switch mode {
	case EarlyShortenTerm:
		// Same payment; find the new term that pays off newBalance.
		newSchedule := reamortise(newBalance, monthlyRate, origPayment)
		newRemainingInterest := sumInterest(newSchedule)
		origRemainingInterest := sumInterest(origSchedule[paidMonths:])
		res.NewPayment = origPayment
		res.NewRemainingTerm = len(newSchedule)
		res.InterestSaved = origRemainingInterest.Sub(newRemainingInterest).Round(2)
		res.NewSchedule = newSchedule
	case EarlyLowerPayment:
		// Same remaining term; recompute payment.
		remainingMonths := termMonths - paidMonths
		newPayment, perr := AnnuityPayment(newBalance, monthlyRate, remainingMonths)
		if perr != nil {
			return EarlyRepaymentResult{}, perr
		}
		newSchedule, serr := AnnuitySchedule(newBalance, monthlyRate, newPayment, remainingMonths)
		if serr != nil {
			return EarlyRepaymentResult{}, serr
		}
		newRemainingInterest := sumInterest(newSchedule)
		origRemainingInterest := sumInterest(origSchedule[paidMonths:])
		res.NewPayment = newPayment
		res.NewRemainingTerm = remainingMonths
		res.InterestSaved = origRemainingInterest.Sub(newRemainingInterest).Round(2)
		res.NewSchedule = newSchedule
	}
	return res, nil
}

// reamortise pays off balance at `payment` per month until balance closes.
// Used by shorten-term mode where the payment is fixed but the new term is
// unknown. The last payment absorbs rounding residual.
func reamortise(balance, monthlyRate, payment decimal.Decimal) []ScheduleRow {
	scale := int32(2)
	rows := make([]ScheduleRow, 0, 32)
	for !balance.IsZero() && balance.IsPositive() {
		k := len(rows) + 1
		interest := balance.Mul(monthlyRate).Round(scale)
		principalPart := payment.Sub(interest).Round(scale)
		// If the principal part would exceed balance (final payment), clamp.
		if principalPart.GreaterThanOrEqual(balance) {
			principalPart = balance.Round(scale)
		}
		paymentRow := principalPart.Add(interest).Round(scale)
		balance = balance.Sub(principalPart).Round(scale)
		if balance.IsNegative() {
			balance = decimal.Zero
		}
		rows = append(rows, ScheduleRow{
			Month:      k,
			Payment:    paymentRow,
			Principal:  principalPart,
			Interest:   interest,
			BalanceEnd: balance,
		})
		// Safety against infinite loops on pathological inputs.
		if len(rows) > 10000 {
			break
		}
	}
	return rows
}

func sumInterest(rows []ScheduleRow) decimal.Decimal {
	total := decimal.Zero
	for _, r := range rows {
		total = total.Add(r.Interest)
	}
	return total
}
