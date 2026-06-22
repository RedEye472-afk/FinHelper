// Package investment implements investment-metric formulas: NPV, XIRR, MIRR,
// DPP (discounted payback period), and PI (profitability index).
//
// Design principle (CLAUDE.md §1 "Детерминизм"): all monetary math uses
// decimal.Decimal. The SINGLE documented exception is XIRR — it reuses the
// BrentQ float64 solver from the credit package (the second and last
// float64 bridge allowed by "Plane from GLM.md" §"Самый рискованный"; PSK
// is the first). The XIRR rate is converted back to decimal at the API
// boundary. Unlike PSK (which is a NOMINAL annual rate per CBR
// methodology), XIRR is an EFFECTIVE annual yield — the exponent is
// days/365, not base-periods.
//
// Sources:
//   - Копнова Г.П. "Финансовая математика" Гл. 6
//   - Ширяев А.Н. "Финансовая математика" (XIRR, MIRR)
//   - MATH_FORMULAS.md §3
package investment

import "errors"

// Sentinel errors. Callers branch on errors.Is.
var (
	// ErrInsufficientCashflows — need at least 2 cashflows for IRR-family.
	ErrInsufficientCashflows = errors.New("investment: need at least 2 cashflows")
	// ErrNoSignChange — XIRR undefined: cashflows do not change sign.
	ErrNoSignChange = errors.New("investment: no sign change in cashflows — XIRR undefined")
	// ErrSolverFailed — BrentQ did not converge within iteration budget.
	ErrSolverFailed = errors.New("investment: solver did not converge")
	// ErrNoPositiveCF — MIRR/PI require at least one positive flow.
	ErrNoPositiveCF = errors.New("investment: no positive cashflow")
	// ErrNoNegativeCF — MIRR requires at least one negative flow (the investment).
	ErrNoNegativeCF = errors.New("investment: no negative cashflow (investment)")
	// ErrInvalidRate — discount/finance rate must be >= -1.
	ErrInvalidRate = errors.New("investment: rate must be > -1")
	// ErrNeverPaidBack — DPP: cumulative discounted flow never reaches zero.
	ErrNeverPaidBack = errors.New("investment: project never pays back at this rate")
	// ErrZeroInitialInvestment — PI undefined: initial investment is zero.
	ErrZeroInitialInvestment = errors.New("investment: initial investment is zero — PI undefined")
)
