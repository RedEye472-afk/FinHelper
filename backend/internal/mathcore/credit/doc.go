// Package credit implements loan calculations: annuity and differentiated
// payments, the Полная стоимость кредита (PSK / total cost of credit) per
// Bank of Russia methodology, and early-repayment scenarios.
//
// Design principle (CLAUDE.md §1 "Детерминизм"): monetary math uses
// decimal.Decimal — never float64. The SINGLE documented exception is the
// numerical root-finder (BrentQ) inside solver.go and the IRR/PSK/XIRR
// routines that depend on it: the solver runs on float64 and its result is
// immediately converted back to decimal.Decimal at the API boundary. This
// matches the plan in "Plane from GLM.md" §"Самый рискованный" — only PSK
// and XIRR may use a documented float64 bridge, everything else stays
// strictly decimal.
//
// Sources:
//   - Копнова Г.П. "Финансовая математика" Гл. 4 (annuity, differentiated)
//   - Bank of Russia Указание 5750-У (ex-5832-У) — PSK methodology
//   - MATH_FORMULAS.md §2
package credit

import "errors"

// Sentinel errors. Callers branch on errors.Is.
var (
	// ErrInvalidTerm — term in months must be > 0.
	ErrInvalidTerm = errors.New("credit: term months must be > 0")
	// ErrInvalidMonth — month number out of [1, term] range.
	ErrInvalidMonth = errors.New("credit: month number out of range")
	// ErrInvalidPrincipal — principal must be >= 0.
	ErrInvalidPrincipal = errors.New("credit: principal must be >= 0")
	// ErrNoSignChange — IRR/PSK undefined: cashflows do not change sign.
	ErrNoSignChange = errors.New("credit: no sign change in cashflows — IRR undefined")
	// ErrInsufficientCashflows — need at least 2 cashflows to compute IRR.
	ErrInsufficientCashflows = errors.New("credit: need at least 2 cashflows")
	// ErrSolverFailed — BrentQ did not converge within iteration budget.
	ErrSolverFailed = errors.New("credit: solver did not converge")
	// ErrEarlyExceedsBalance — early repayment amount exceeds outstanding balance.
	ErrEarlyExceedsBalance = errors.New("credit: early repayment exceeds outstanding balance")
	// ErrInvalidEarlyAmount — early repayment amount must be >= 0.
	ErrInvalidEarlyAmount = errors.New("credit: early repayment amount must be >= 0")
)
