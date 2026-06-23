package investment

// XIRR — Extended IRR for cashflows at IRREGULAR dates. Reuses the BrentQ
// solver from the credit package (the second and final documented float64
// bridge; see package doc and "Plane from GLM.md"). The discount exponent
// is days_t / 365, so XIRR is an EFFECTIVE annual yield — distinct from
// PSK's nominal annual rate.
//
// Equation to solve for r:
//
//	Σ_{t}  CF_t / (1 + r)^(days_t / 365) = 0
//
// Source: Ширяев А.Н. "Финансовая математика"; MATH_FORMULAS.md §3.2.

import (
	"math"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/internal/mathcore/credit"
)

// cfF64 is the float64 projection of a Cashflow used inside the solver.
type cfF64 struct {
	days   float64 // calendar days from the first flow date
	amount float64
}

// XIRR computes the effective annual internal rate of return of the given
// dated cashflows. Returns the rate as a decimal fraction (0.1234 ≈ 12.34 %).
//
// The cashflows MUST contain at least one positive and one negative amount.
// Dates may be unsorted; they are normalised to UTC midnight and ordered.
// The first chronological flow's date is the discount base (d_0).
//
// Edge cases:
//   - < 2 cashflows → ErrInsufficientCashflows
//   - no sign change → ErrNoSignChange
//   - bracket search fails → ErrSolverFailed
//
// Source: MATH_FORMULAS.md §3.2.
func XIRR(cashflows []Cashflow) (decimal.Decimal, error) {
	if len(cashflows) < 2 {
		return decimal.Zero, ErrInsufficientCashflows
	}
	ordered := sortByDate(cashflows)

	epoch := ordered[0].Date
	cfs := make([]cfF64, len(ordered))
	hasPos, hasNeg := false, false
	for i, cf := range ordered {
		days := cf.Date.Sub(epoch).Hours() / 24
		amt := cf.Amount.InexactFloat64()
		cfs[i] = cfF64{days: days, amount: amt}
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

	rate, err := solveXIRR(cfs)
	if err != nil {
		return decimal.Zero, err
	}
	return decimal.NewFromFloat(rate), nil
}

// npvXIRR evaluates Σ CF_t / (1+r)^(days_t/365) at the given annual rate.
// Runs on float64 deliberately — see package doc on the bridge.
func npvXIRR(cfs []cfF64, r float64) float64 {
	var npv float64
	for _, cf := range cfs {
		years := cf.days / 365.0
		disc := math.Pow(1+r, years)
		npv += cf.amount / disc
	}
	return npv
}

// solveXIRR finds a bracket and runs BrentQ.
//
// Monotonicity: NPV(r) = Σ CF_t / (1+r)^(days_t/365) is monotonically
// DECREASING in r for a typical investment cashflow (negative outflow at
// start, positive returns later — discounting shrinks returns as r grows).
// But for arbitrary cashflow patterns the sign of the slope can flip, so we
// don't assume a direction: probe a fixed grid of rates on [-0.99, +large]
// and pick the first adjacent pair with a sign change.
//
// Economic bounds: r ∈ (-0.99, +∞) — 1+r must stay positive.
func solveXIRR(cfs []cfF64) (float64, error) {
	npvAt := func(r float64) float64 { return npvXIRR(cfs, r) }

	fZero := npvAt(0)
	if fZero == 0 {
		return 0, nil
	}

	// Bracket search: probe upward first (typical investment root > 0),
	// then downward toward -0.99. Geometric step doubling.
	type probe struct{ r, f float64 }
	history := []probe{{0, fZero}}

	tryAt := func(r float64) (float64, bool) {
		f := npvAt(r)
		// Compare against the most recent anchor with opposite sign.
		for _, p := range history {
			if math.Signbit(p.f) != math.Signbit(f) && p.f != 0 {
				lo, fLo := p.r, p.f
				hi, fHi := r, f
				if lo > hi {
					lo, hi = hi, lo
					fLo, fHi = fHi, fLo
				}
				root, err := credit.BrentQ(npvAt, lo, hi, fLo, fHi, 1e-12, 200)
				if err != nil {
					return 0, false
				}
				return root, true
			}
		}
		history = append(history, probe{r, f})
		return 0, false
	}

	// Probe upward: 0.01, 0.02, 0.04, ... up to 1e6.
	step := 0.01
	for i := 0; i < 200; i++ {
		r := step
		if root, ok := tryAt(r); ok {
			return root, nil
		}
		step *= 2
		if step > 1e6 {
			break
		}
	}
	// Probe downward toward -0.99: -0.01, then bisect toward -0.99.
	lower := -0.01
	for i := 0; i < 200; i++ {
		if root, ok := tryAt(lower); ok {
			return root, nil
		}
		next := lower + (-0.99-lower)/2
		if next <= -0.99 || next >= lower {
			break
		}
		lower = next
	}
	return 0, ErrSolverFailed
}
