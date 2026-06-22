package credit

// PSK (Полная стоимость кредита) — the annualised cost of credit per Bank of
// Russia Указание 5750-У (ex-5832-У). This is the ONLY place besides the
// solver itself where a float64 bridge is permitted (see package doc).
//
// Methodology (CBR Указание 5750-У, Приложение 1):
//
//	1. Pick a base period. For loans with monthly payments ЧБП (число базовых
//	   периодов в году) = 12. The base-period length is 365/ЧБП ≈ 30.42 days.
//	2. Express every cashflow date as q_k — the number of base periods from
//	   the first date: q_k = (d_k − d_0) × ЧБП / 365.
//	3. Solve for the per-base-period rate i in:
//	       Σ_{k=0}^{n}  CF_k / (1 + i)^q_k = 0
//	4. Annualise as ПСК = i × ЧБП. The result is the NOMINAL annual rate
//	   compounded ЧБП times per year — the convention used in Russian loan
//	   disclosures (NOT the effective annual rate).
//
// Why nominal not effective: the CBR-defined PSK lets consumers compare
// loans the same way they compare nominal interest rates. A 12%-nominal
// monthly-compounded loan with no fees has ПСК = 12% exactly, which matches
// the headline rate the bank advertises.
//
// Source: MATH_FORMULAS.md §2.3; Bank of Russia Указание 5750-У.

import (
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// Cashflow is one dated monetary flow for IRR/PSK computation. Positive =
// money received by the borrower (disbursed principal), negative = money
// paid out (interest, principal, fees). The date is used for base-period
// discounting; amounts are NOT scaled here — pass raw decimal values.
type Cashflow struct {
	Date   time.Time
	Amount decimal.Decimal
}

// PeriodsPerYear is the CBR ЧБП — number of base periods in a calendar year.
// For monthly loans this is 12; we expose it so callers can override for
// quarterly (4) or annual (1) schedules.
type PeriodsPerYear int

const (
	// MonthlyPeriods is the ЧБП for monthly-payment loans (default).
	MonthlyPeriods PeriodsPerYear = 12
)

// PSK computes the annual Полная стоимость кредита for the given cashflows
// using CBR Указание 5750-У methodology with ЧБП = 12 (monthly base period).
// Returns the rate as a decimal fraction (0.1796 ≈ 17.96 %).
//
// The cashflows MUST contain at least one positive and one negative amount
// (a sign change is necessary for an IRR to exist). Dates may be unsorted;
// they are normalised to UTC midnight and ordered. The first chronological
// flow's date is the discount base (d_0).
//
// Edge cases:
//   - < 2 cashflows → ErrInsufficientCashflows
//   - no sign change → ErrNoSignChange
//   - bracket search fails → ErrSolverFailed
//
// Source: MATH_FORMULAS.md §2.3; CBR Указание 5750-У.
func PSK(cashflows []Cashflow) (decimal.Decimal, error) {
	return PSKWithPeriods(cashflows, MonthlyPeriods)
}

// PSKWithPeriods is like PSK but lets the caller choose ЧБП (e.g. 4 for
// quarterly schedules). Exposed so the investment package can reuse the
// solver for XIRR-style computations with a different base period.
func PSKWithPeriods(cashflows []Cashflow, ppw PeriodsPerYear) (decimal.Decimal, error) {
	if len(cashflows) < 2 {
		return decimal.Zero, ErrInsufficientCashflows
	}
	if ppw <= 0 {
		return decimal.Zero, ErrInvalidTerm
	}

	// Normalise to UTC midnight and sort by date ascending.
	normalised := make([]Cashflow, len(cashflows))
	for i, cf := range cashflows {
		normalised[i] = Cashflow{
			Date:   time.Date(cf.Date.Year(), cf.Date.Month(), cf.Date.Day(), 0, 0, 0, 0, time.UTC),
			Amount: cf.Amount,
		}
	}
	sort.SliceStable(normalised, func(i, j int) bool {
		return normalised[i].Date.Before(normalised[j].Date)
	})

	// Convert to the float64 representation the solver consumes.
	// q_k = (d_k − d_0) × ЧБП / 365 — fractional base periods from the base date.
	chbp := float64(ppw)
	epoch := normalised[0].Date
	cfs := make([]cfF64, len(normalised))
	hasPos, hasNeg := false, false
	for i, cf := range normalised {
		days := cf.Date.Sub(epoch).Hours() / 24
		q := days * chbp / 365.0
		amt := cf.Amount.InexactFloat64()
		cfs[i] = cfF64{q: q, amount: amt}
		if amt > 0 {
			hasPos = true
		}
		if amt < 0 {
			hasNeg = true
		}
	}
	if !(hasPos && hasNeg) {
		return decimal.Zero, ErrNoSignChange
	}

	ratePerPeriod, err := solveIRR(cfs, npvPerPeriod)
	if err != nil {
		return decimal.Zero, err
	}
	// Annualise: ПСК = i × ЧБП (nominal annual rate per CBR convention).
	annual := ratePerPeriod * chbp
	return decimal.NewFromFloat(annual), nil
}

// cfF64 is the float64 projection of a Cashflow used by the solver.
type cfF64 struct {
	q      float64 // base-periods from base date
	amount float64
}

// npvPerPeriod evaluates Σ CF_k / (1+i)^q_k at the given per-period rate.
// Runs on float64 deliberately (see package doc on the bridge).
func npvPerPeriod(cfs []cfF64, ratePerPeriod float64) float64 {
	var npv float64
	for _, cf := range cfs {
		disc := math.Pow(1+ratePerPeriod, cf.q)
		npv += cf.amount / disc
	}
	return npv
}

// solveIRR finds a bracket for the NPV function and runs BrentQ.
//
// IMPORTANT monotonicity note. For a borrower's loan cashflow the
// disbursed principal (positive) comes first and the repayments (negative)
// come later. NPV(i) = Σ CF_k / (1+i)^q_k is then MONOTONICALLY INCREASING
// in i: a higher rate discounts the later negative payments more, pulling
// NPV up. Therefore:
//
//   - if NPV(0) < 0  → root is POSITIVE (search UP from 0)
//   - if NPV(0) > 0  → root is NEGATIVE (search DOWN from 0)
//
// (For a typical investment cashflow the direction is opposite, but this
// function is shared with XIRR; the scan below handles both because it
// simply walks in whichever direction the sign changes.)
//
// Economic bounds: i ∈ (-0.99, +∞) per base period — 1+i must stay > 0.
func solveIRR(cfs []cfF64, npv func([]cfF64, float64) float64) (float64, error) {
	npvAt := func(r float64) float64 { return npv(cfs, r) }

	fZero := npvAt(0)

	// fZero == 0 means cashflows sum to zero — IRR is exactly 0.
	if fZero == 0 {
		return 0, nil
	}

	// Walk away from 0 in the direction where NPV sign changes:
	//   - if NPV(0) < 0, we want a positive r with NPV(r) > 0 → walk UP
	//   - if NPV(0) > 0, we want a negative r with NPV(r) < 0 → walk DOWN
	direction := 1.0
	if fZero > 0 {
		direction = -1.0
	}

	anchor, fAnchor := 0.0, fZero
	step := 0.01
	for i := 0; i < 200; i++ {
		candidate := direction * step
		// Guard the lower bound: never cross -0.99 (1+i > 0 must hold).
		if candidate <= -0.99 {
			candidate = -0.99 + 1e-9
		}
		fCand := npvAt(candidate)
		// Sign change found between anchor and candidate → Brent on that bracket.
		if math.Signbit(fAnchor) != math.Signbit(fCand) {
			lo, fLo := anchor, fAnchor
			hi, fHi := candidate, fCand
			if lo > hi {
				lo, hi = hi, lo
				fLo, fHi = fHi, fLo
			}
			return BrentQ(npvAt, lo, hi, fLo, fHi, 1e-12, 200)
		}
		// No sign change yet: advance the anchor and grow the step.
		anchor, fAnchor = candidate, fCand
		step *= 2
		// Upper bound: stop at absurd rates; lower bound handled above.
		if direction > 0 && step > 1e6 {
			break
		}
		if direction < 0 && anchor <= -0.99+1e-9 {
			break
		}
	}
	return 0, ErrSolverFailed
}

// AnnuityCashflows builds the cashflow series for an annuity loan with
// optional one-off compulsory fees disbursed on the same day as the
// principal. This is the typical input for PSK:
//
//	CF_0 = +principal − ΣupfrontFees     (money the borrower actually receives)
//	CF_k = −monthlyPayment              (each month, k = 1..n)
//
// Dates are spaced 1 calendar month from firstDate via AddDate(0, k, 0);
// the day-of-month is kept (so firstDate's day stays the payment day).
// AddDate normalises overflow (e.g. Jan 31 + 1 month → Feb 28/29), which is
// handled correctly by the date-based q_k computation in PSK.
//
// Source: MATH_FORMULAS.md §2.3 example.
func AnnuityCashflows(
	principal decimal.Decimal,
	monthlyPayment decimal.Decimal,
	termMonths int,
	upfrontFees []decimal.Decimal,
	firstDate time.Time,
) ([]Cashflow, error) {
	if termMonths <= 0 {
		return nil, ErrInvalidTerm
	}
	if principal.IsNegative() {
		return nil, ErrInvalidPrincipal
	}

	totalFees := decimal.Zero
	for _, f := range upfrontFees {
		totalFees = totalFees.Add(f)
	}
	netDisbursed := principal.Sub(totalFees)

	out := make([]Cashflow, 0, termMonths+1)
	out = append(out, Cashflow{Date: firstDate, Amount: netDisbursed})
	for k := 1; k <= termMonths; k++ {
		d := firstDate.AddDate(0, k, 0)
		out = append(out, Cashflow{Date: d, Amount: monthlyPayment.Neg()})
	}
	return out, nil
}
