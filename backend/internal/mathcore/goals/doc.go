// Package goals implements sinking-fund formulas for savings-goal planning
// (BUSINESS_LOGIC.md ф.5). Given a present capital P, a target amount S, a
// periodic contribution A, a per-period rate i, and a number of periods n,
// the package solves for any one unknown analytically.
//
// Design principle (CLAUDE.md §1 "Детерминизм"): all calculations use
// decimal.Decimal — never float64. No numerical solver is required: each
// unknown has a closed form (Копнова Гл. 3.3.3), so this package adds NO
// new float64 bridge. The project's documented float64 bridges remain at 2
// (credit/BrentQ for PSK + XIRR).
//
// Conventions:
//   - i is the per-period rate (annual_yield / 12 for monthly contributions)
//   - n is the number of periods (months)
//   - amounts are passed as decimal.Decimal; money-scale rounding is the
//     caller's responsibility (this package is pure math)
//
// Source: Копнова Г.П. "Финансовая математика" Гл. 3.3.3 "Фонд возмещения";
// MATH_FORMULAS.md §2.1 (annuity family).
package goals

import "errors"

// Sentinel errors. Callers branch on errors.Is.
var (
	// ErrNonPositiveTarget — target amount S must be > 0.
	ErrNonPositiveTarget = errors.New("goals: target amount must be > 0")
	// ErrNonPositiveContribution — periodic contribution A must be > 0 where required.
	ErrNonPositiveContribution = errors.New("goals: contribution must be > 0")
	// ErrInvalidPeriods — number of periods n must be > 0 where required.
	ErrInvalidPeriods = errors.New("goals: periods must be > 0")
	// ErrUnreachable — the contribution is too small to ever reach the target
	// (it does not cover the growth of interest on the present capital, so the
	// gap widens instead of closing). Returned by SolveTerm.
	ErrUnreachable = errors.New("goals: contribution too small to reach target")
	// ErrInvalidRate — periodic rate i must be > 0 for SolveTerm (Ln requires it).
	ErrInvalidRate = errors.New("goals: periodic rate must be > 0 for SolveTerm")
	// ErrDeflation100Percent — inflation = -1 would divide by zero in InflateTarget.
	ErrDeflation100Percent = errors.New("goals: inflation = -1 would divide by zero")
)
