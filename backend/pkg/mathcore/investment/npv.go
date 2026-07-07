package investment

// NPV, DPP, PI — deterministic, fully on decimal.Decimal (no float bridge).
//
// Source: Копнова Г.П. Гл. 6.1; MATH_FORMULAS.md §3.1, §3.4, §3.5.

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// Cashflow is one dated monetary flow. Negative = investment outflow,
// positive = return inflow. Date is used for XIRR day-count discounting.
type Cashflow struct {
	Date   time.Time
	Amount decimal.Decimal
}

// NPV computes the Net Present Value of a periodic cashflow series at the
// given per-period discount rate:
//
//	NPV = Σ_{t=0}^{n}  CF_t / (1 + r)^t
//
// Periods are assumed equal-length and 0-indexed from the first flow.
// Use XIRR-family for irregular dates.
//
// Edge cases:
//   - r <= -1 → ErrInvalidRate ((1+r)^t must stay positive)
//   - empty cashflows → 0
//
// Source: MATH_FORMULAS.md §3.1.
func NPV(cashflows []decimal.Decimal, rate decimal.Decimal) (decimal.Decimal, error) {
	if len(cashflows) == 0 {
		return decimal.Zero, nil
	}
	if !rate.GreaterThan(decimal.NewFromInt(-1)) {
		return decimal.Zero, ErrInvalidRate
	}
	one := decimal.NewFromInt(1)
	base := one.Add(rate)
	total := decimal.Zero
	for t, cf := range cashflows {
		disc := base.Pow(decimal.NewFromInt(int64(t)))
		total = total.Add(cf.Div(disc))
	}
	return total, nil
}

// sortByDate orders cashflows chronologically and snaps to UTC midnight.
// Returns a new slice; does not mutate the input.
func sortByDate(cfs []Cashflow) []Cashflow {
	out := make([]Cashflow, len(cfs))
	for i, cf := range cfs {
		out[i] = Cashflow{
			Date:   time.Date(cf.Date.Year(), cf.Date.Month(), cf.Date.Day(), 0, 0, 0, 0, time.UTC),
			Amount: cf.Amount,
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Date.Before(out[j].Date)
	})
	return out
}
