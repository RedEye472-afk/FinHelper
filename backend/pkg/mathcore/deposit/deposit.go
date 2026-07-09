package deposit

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// CapFreq represents the capitalization (compounding) frequency per year.
type CapFreq int

const (
	// CapMaturity — simple interest, no capitalization during the term.
	// Formula: S = P * (1 + rate * months/12).
	CapMaturity CapFreq = 0

	// CapAnnually — capitalization once per year (m = 1).
	CapAnnually CapFreq = 1

	// CapQuarterly — capitalization four times per year (m = 4).
	CapQuarterly CapFreq = 4

	// CapMonthly — capitalization twelve times per year (m = 12).
	CapMonthly CapFreq = 12
)

// Result holds the output of a deposit calculation.
type Result struct {
	MaturityAmount decimal.Decimal // итоговая сумма на момент окончания
	TotalInterest  decimal.Decimal // начисленные проценты (MaturityAmount − Principal)
	EffectiveRate  decimal.Decimal // эффективная годовая ставка
	RealReturn     decimal.Decimal // реальная доходность после инфляции (Fisher), 0 если inflationRate = 0
	Projection     []PeriodRow     // помесячная разбивка для графика
}

// PeriodRow is one month in the deposit projection.
type PeriodRow struct {
	Month              int             // номер месяца (1-based)
	Balance            decimal.Decimal // остаток после начисления (capitalisation events applied)
	Interest           decimal.Decimal // проценты, начисленные за этот месяц
	CumulativeInterest decimal.Decimal // проценты накопленные с начала срока
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	// ErrInvalidTerm — term in months must be > 0.
	ErrInvalidTerm = errors.New("deposit: term months must be > 0")
	// ErrInvalidCapFreq — unrecognised capitalization frequency.
	ErrInvalidCapFreq = errors.New("deposit: invalid capitalization frequency")
	// ErrInvalidRate — annual rate must be >= 0.
	ErrInvalidRate = errors.New("deposit: annual rate must be >= 0")
)

// ---------------------------------------------------------------------------
// Calculate (public API)
// ---------------------------------------------------------------------------

// Calculate computes the full deposit result.
//
// Parameters:
//   - P:          principal (начальная сумма)
//   - rate:       annual nominal rate as a fraction (0.10 = 10 %)
//   - months:     term in months (must be > 0)
//   - capFreq:    capitalization frequency (CapMaturity / CapAnnually / CapQuarterly / CapMonthly)
//   - inflation:  annual inflation rate as a fraction (0.08 = 8 %); pass zero to skip Fisher calc
//
// Edge cases:
//   - rate = 0      → maturity = principal, all interest = 0
//   - months <= 0   → ErrInvalidTerm
//   - P = 0         → all zero (no error)
//   - inflation = 0 → RealReturn = 0 (Fisher skipped)
//   - capFreq = CapMaturity → simple interest
//
// Rounding: monetary values (MaturityAmount, TotalInterest, projection
// Balance/Interest) are rounded to scale 2 (kopecks) with RoundHalfUp.
// EffectiveRate and RealReturn are left at full precision.
func Calculate(P decimal.Decimal, rate decimal.Decimal, months int, capFreq CapFreq, inflation decimal.Decimal) (Result, error) {
	if months <= 0 {
		return Result{}, ErrInvalidTerm
	}
	if rate.IsNegative() {
		return Result{}, ErrInvalidRate
	}

	// P = 0 → everything zero.
	if P.IsZero() {
		return zeroResult(months), nil
	}

	var (
		maturity decimal.Decimal
		proj     []PeriodRow
	)

	switch capFreq {
	case CapMaturity:
		maturity, proj = simpleCalc(P, rate, months)
	case CapAnnually, CapQuarterly, CapMonthly:
		maturity, proj = compoundCalc(P, rate, months, int(capFreq))
	default:
		return Result{}, fmt.Errorf("%w: %d", ErrInvalidCapFreq, capFreq)
	}

	totalInterest := maturity.Sub(P)

	// ---- Effective annual rate ----
	effRate := decimal.Zero
	if rate.IsPositive() && capFreq != CapMaturity {
		m := decimal.NewFromInt(int64(capFreq))
		one := decimal.NewFromInt(1)
		base := one.Add(rate.Div(m))
		effRate = base.Pow(m).Sub(one)
	} else if rate.IsPositive() && capFreq == CapMaturity {
		// For simple interest the nominal rate IS the effective rate
		// (no compounding within the year).
		effRate = rate
	}
	// rate == 0 → effRate stays 0

	// ---- Fisher real return ----
	realReturn := decimal.Zero
	if inflation.IsPositive() {
		one := decimal.NewFromInt(1)
		denom := one.Add(inflation)
		if !denom.IsZero() { // guard against pi = -1 (economically impossible but defensive)
			realReturn = one.Add(rate).Div(denom).Sub(one)
		}
	}

	r := Result{
		MaturityAmount: maturity.Round(2),
		TotalInterest:  totalInterest.Round(2),
		EffectiveRate:  effRate,
		RealReturn:     realReturn,
		Projection:     proj,
	}

	return r, nil
}

// ---------------------------------------------------------------------------
// Simple interest (CapMaturity)
// ---------------------------------------------------------------------------

// simpleCalc computes S = P × (1 + rate × months/12) and builds a projection.
// The projection keeps the balance at P for months 1..months-1 and shows the
// full interest accrual in the final month.
func simpleCalc(P, rate decimal.Decimal, months int) (decimal.Decimal, []PeriodRow) {
	one := decimal.NewFromInt(1)
	t := decimal.NewFromInt(int64(months)).Div(decimal.NewFromInt(12)) // months / 12
	factor := one.Add(rate.Mul(t))
	maturity := P.Mul(factor)

	proj := make([]PeriodRow, months)
	var cumInt decimal.Decimal

	for m := 1; m <= months; m++ {
		if m == months {
			interest := maturity.Sub(P)
			cumInt = cumInt.Add(interest)
			proj[m-1] = PeriodRow{
				Month:              m,
				Balance:            maturity.Round(2),
				Interest:           interest.Round(2),
				CumulativeInterest: cumInt.Round(2),
			}
		} else {
			proj[m-1] = PeriodRow{
				Month:              m,
				Balance:            P.Round(2),
				Interest:           decimal.Zero,
				CumulativeInterest: decimal.Zero,
			}
		}
	}

	return maturity, proj
}

// ---------------------------------------------------------------------------
// Compound interest
// ---------------------------------------------------------------------------

// compoundCalc computes S = P × (1 + rate/m)^(m × months/12) and builds a
// month-by-month projection. Capitalisation events happen at regular intervals
// (every 12/m months). Interest accrues and is credited to the balance only
// on capitalisation months; between events the balance stays flat.
//
// The final maturity is the balance after the last capitalisation event
// (which may be before the final month if the term is not aligned to a
// capitalisation boundary — realistic for early withdrawal).
func compoundCalc(P, rate decimal.Decimal, months int, periodsPerYear int) (decimal.Decimal, []PeriodRow) {
	periodRate := rate.Div(decimal.NewFromInt(int64(periodsPerYear)))

	// Capitalisation interval in months: 12/periodsPerYear
	//   monthly   (12) → 1 month
	//   quarterly (4)  → 3 months
	//   annually  (1)  → 12 months
	capInterval := 12 / periodsPerYear

	balance := P
	var cumInt decimal.Decimal
	proj := make([]PeriodRow, months)

	for m := 1; m <= months; m++ {
		var interest decimal.Decimal

		if m%capInterval == 0 {
			interest = balance.Mul(periodRate)
			balance = balance.Add(interest)
			cumInt = cumInt.Add(interest)
		}
		// else: no capitalisation this month → interest = 0, balance unchanged

		proj[m-1] = PeriodRow{
			Month:              m,
			Balance:            balance.Round(2),
			Interest:           interest.Round(2),
			CumulativeInterest: cumInt.Round(2),
		}
	}

	return balance, proj
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// zeroResult returns a Result where everything is zero.
func zeroResult(months int) Result {
	proj := make([]PeriodRow, months)
	for m := 1; m <= months; m++ {
		proj[m-1] = PeriodRow{
			Month:              m,
			Balance:            decimal.Zero,
			Interest:           decimal.Zero,
			CumulativeInterest: decimal.Zero,
		}
	}
	return Result{
		MaturityAmount: decimal.Zero,
		TotalInterest:  decimal.Zero,
		EffectiveRate:  decimal.Zero,
		RealReturn:     decimal.Zero,
		Projection:     proj,
	}
}
