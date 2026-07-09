// Package deposit implements BUSINESS_LOGIC.md ф.6 — deposit calculator.
//
// This is the stateless orchestration layer over mathcore/tvm:
// it validates inputs, computes compound/simple interest with various
// capitalisation frequencies, calculates effective rate, real return
// (Fisher equation), tax on deposit interest, and a month-by-month
// projection.
//
// Design mirrors service/credit (BUSINESS_LOGIC ф.7):
//   - Pure service: holds no per-request state, no database dependency.
//     The HTTP handler for POST /calc/deposit constructs a Service at boot
//     and calls Calculate on each request.
//   - Money is decimal end-to-end; the only float64 surface lives inside
//     mathcore/credit's PSK solver (not used here — this package is pure
//     decimal).
//   - Inputs are validated here, not in mathcore/tvm; mathcore takes
//     already-sane primitives. Sentinel errors map to HTTP 400 in the
//     handler.
//
// Sources: BUSINESS_LOGIC.md §6; MATH_FORMULAS.md §1 (Копнова Гл. 1);
// НК РФ ст. 214.2; ФЗ-382 от 26.12.2023.
package deposit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/mathcore/tax"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/mathcore/tvm"
)

// Sentinel errors. The HTTP handler branches on errors.Is.
var (
	// ErrInvalidArgument — request failed validation.
	ErrInvalidArgument = errors.New("deposit: invalid argument")
)

// CapFreq selects how often interest is capitalised (added to principal).
type CapFreq string

const (
	// CapMonthly — interest capitalised every month (most common).
	CapMonthly CapFreq = "monthly"
	// CapQuarterly — interest capitalised every quarter (every 3 months).
	CapQuarterly CapFreq = "quarterly"
	// CapAnnually — interest capitalised once per year.
	CapAnnually CapFreq = "annually"
	// CapMaturity — no capitalisation; simple interest paid at maturity.
	CapMaturity CapFreq = "maturity"
)

// Input is the body of POST /calc/deposit. All monetary fields are raw
// decimal strings parsed by the handler into decimal.Decimal. Optional
// fields have zero/empty defaults documented inline.
type Input struct {
	Principal      decimal.Decimal // > 0
	AnnualRate     decimal.Decimal // >= 0 (0 → interest-free)
	TermMonths     int             // > 0
	Capitalization CapFreq         // "monthly" (default) | "quarterly" | "annually" | "maturity"
	InflationRate  decimal.Decimal // optional, >= 0 (0 → skip real return)
	TaxYear        int             // optional, 0 → skip tax calc
}

// Result is the response of POST /calc/deposit (BUSINESS_LOGIC §6).
type Result struct {
	MaturityAmount decimal.Decimal `json:"maturity_amount"`
	TotalInterest  decimal.Decimal `json:"total_interest"`
	EffectiveRate  decimal.Decimal `json:"effective_rate"`
	RealReturn     decimal.Decimal `json:"real_return,omitempty"`
	TaxAmount      decimal.Decimal `json:"tax_amount,omitempty"`
	Projection     []PeriodRow     `json:"projection,omitempty"`
	Disclaimer     string          `json:"disclaimer"`
}

// PeriodRow is one month of the deposit projection.
type PeriodRow struct {
	Month              int             `json:"month"`
	Balance            decimal.Decimal `json:"balance"`
	Interest           decimal.Decimal `json:"interest"`
	CumulativeInterest decimal.Decimal `json:"cumulative_interest"`
}

// Service is the deposit calculator. Construct once at boot; stateless.
type Service struct {
	now func() time.Time
}

// NewService returns a Service. now defaults to time.Now; tests inject.
func NewService() *Service { return &Service{now: time.Now} }

// Calculate runs the deposit calculator statelessly. Returns ErrInvalidArgument
// on bad input; mathcore errors propagate wrapped.
func (s *Service) Calculate(ctx context.Context, in Input) (Result, error) {
	// ---- validation ----
	if !in.Principal.IsPositive() {
		return Result{}, fmt.Errorf("%w: principal must be > 0", ErrInvalidArgument)
	}
	if in.TermMonths <= 0 {
		return Result{}, fmt.Errorf("%w: term_months must be > 0", ErrInvalidArgument)
	}
	if in.AnnualRate.IsNegative() {
		return Result{}, fmt.Errorf("%w: annual_rate must be >= 0", ErrInvalidArgument)
	}
	if in.InflationRate.IsNegative() {
		return Result{}, fmt.Errorf("%w: inflation_rate must be >= 0", ErrInvalidArgument)
	}

	// Default capitalisation to monthly when empty/unknown.
	if in.Capitalization == "" {
		in.Capitalization = CapMonthly
	}
	validCap := map[CapFreq]bool{
		CapMonthly: true, CapQuarterly: true, CapAnnually: true, CapMaturity: true,
	}
	if !validCap[in.Capitalization] {
		return Result{}, fmt.Errorf("%w: capitalization must be one of: monthly, quarterly, annually, maturity", ErrInvalidArgument)
	}

	// ---- compute maturity amount and total interest ----
	years := decimal.NewFromInt(int64(in.TermMonths)).Div(decimal.NewFromInt(12))

	var maturityAmount decimal.Decimal
	var err error

	switch in.Capitalization {
	case CapMonthly:
		maturityAmount, err = tvm.CompoundInterest(in.Principal, in.AnnualRate, 12, years)
	case CapQuarterly:
		maturityAmount, err = tvm.CompoundInterest(in.Principal, in.AnnualRate, 4, years)
	case CapAnnually:
		maturityAmount, err = tvm.CompoundInterest(in.Principal, in.AnnualRate, 1, years)
	case CapMaturity:
		maturityAmount, err = tvm.SimpleInterest(in.Principal, in.AnnualRate, years)
	}
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}

	maturityAmount = maturityAmount.Round(2)
	totalInterest := maturityAmount.Sub(in.Principal)

	// ---- effective rate ----
	// For maturity (simple interest) the effective rate equals the nominal rate.
	// For compound caps we compute (1 + i/m)^m − 1.
	var effectiveRate decimal.Decimal
	if in.AnnualRate.IsPositive() {
		compPerYear := compoundingPerYear(in.Capitalization)
		if compPerYear > 0 {
			er, effErr := tvm.EffectiveRate(in.AnnualRate, compPerYear)
			if effErr == nil {
				effectiveRate = er
			}
		}
	}

	// ---- real return (Fisher equation) ----
	// r_real = (1 + r_eff) / (1 + π) − 1
	var realReturn decimal.Decimal
	if in.InflationRate.IsPositive() {
		rr, fisherErr := tvm.FisherRealRate(effectiveRate, in.InflationRate)
		if fisherErr == nil {
			realReturn = rr
		}
	}

	// ---- tax calculation (ФЗ-382) ----
	// Load rules for the given year and compute tax on interest exceeding
	// the non-taxable threshold (1M × key_rate_jan1). If the year is
	// unsupported or the loader fails, we silently skip tax (consistent
	// with the "optional, 0 = skip" convention).
	var taxAmount decimal.Decimal
	if in.TaxYear > 0 {
		rules, loadErr := tax.LoadRules(in.TaxYear)
		if loadErr == nil {
			if t, calcErr := tax.DepositTax(rules, totalInterest); calcErr == nil {
				taxAmount = t
			}
		}
	}

	// ---- month-by-month projection ----
	projection := s.buildProjection(in)

	const disclaimer = "Расчёт носит справочный характер. Реальная доходность зависит от условий банка и может отличаться."

	return Result{
		MaturityAmount: maturityAmount,
		TotalInterest:  totalInterest,
		EffectiveRate:  effectiveRate,
		RealReturn:     realReturn,
		TaxAmount:      taxAmount,
		Projection:     projection,
		Disclaimer:     disclaimer,
	}, nil
}

// buildProjection generates a month-by-month breakdown of the deposit balance,
// per-month interest, and cumulative interest.
//
// Capitalisation logic per frequency:
//   - monthly: balance is updated each month (compound inside the period).
//   - quarterly: balance is updated every 3 months.
//   - annually: balance is updated every 12 months.
//   - maturity: balance stays at principal throughout (interest paid at end).
func (s *Service) buildProjection(in Input) []PeriodRow {
	rows := make([]PeriodRow, 0, in.TermMonths)
	balance := in.Principal
	monthlyRate := in.AnnualRate.Div(decimal.NewFromInt(12))
	cumulative := decimal.Zero

	capInterval := capitalizationInterval(in.Capitalization)

	for month := 1; month <= in.TermMonths; month++ {
		// Interest accrued this month on the current balance.
		interest := balance.Mul(monthlyRate)
		cumulative = cumulative.Add(interest)

		if capInterval > 0 && month%capInterval == 0 {
			// At a capitalisation event the accumulated interest for the
			// interval is added to the principal. Since we record per-month
			// interest separately, multiply by the interval length so the
			// balance jump matches the cumulative interest shown.
			balance = balance.Add(interest.Mul(decimal.NewFromInt(int64(capInterval))))
		}

		rows = append(rows, PeriodRow{
			Month:              month,
			Balance:            balance.Round(2),
			Interest:           interest.Round(2),
			CumulativeInterest: cumulative.Round(2),
		})
	}
	return rows
}

// compoundingPerYear maps a CapFreq to the number of compounding periods
// per year for effective-rate and compound-interest calculations.
func compoundingPerYear(c CapFreq) int {
	switch c {
	case CapMonthly:
		return 12
	case CapQuarterly:
		return 4
	case CapAnnually, CapMaturity:
		return 1
	default:
		return 0
	}
}

// capitalizationInterval returns the month interval at which interest is
// capitalised (added to principal). 0 means "no capitalisation during term"
// (maturity).
func capitalizationInterval(c CapFreq) int {
	switch c {
	case CapMonthly:
		return 1
	case CapQuarterly:
		return 3
	case CapAnnually:
		return 12
	case CapMaturity:
		return 0
	default:
		return 1
	}
}
