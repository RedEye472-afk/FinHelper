package credit

// This file contains the SINGLE documented float64 bridge in the mathcore
// package. Brent's method (BrentQ) needs iterative function evaluation and
// is impractical to express efficiently on decimal.Decimal.
//
// Per "Plane from GLM.md" §"Самый рискованный" only PSK and XIRR may use a
// float64 numerical solver. BrentQ finds a root; the caller (PSK/XIRR)
// converts the result back to decimal.Decimal at the API boundary. No float
// ever leaks into money math.
//
// Algorithm: Brent–Dekker root finding (bisection + secant + inverse
// quadratic interpolation). Reference: Brent, R. P. (1973), "Algorithms for
// Minimization without Derivatives", Ch. 4. Sign-faithful and bracket-
// preserving: the root is always inside the input bracket.

import (
	"math"
)

// BrentQ finds a root of f on the bracket [a, b] using Brent's method.
//
// Precondition: f(a) and f(b) must have opposite signs (or be zero). The
// caller is responsible for finding a valid bracket; BrentQ does not search.
// This precondition is enforced: equal-sign non-zero fa/fb → ErrSolverFailed.
//
// Parameters:
//   - f: function whose root we want.
//   - a, b: bracket endpoints (a < b not required, but typical).
//   - fa, fb: precomputed f(a), f(b) — saves recomputation and lets callers
//     assert the bracket validity once.
//   - tol: absolute tolerance on the root location.
//   - maxIter: iteration budget (typical: 100).
//
// Returns ErrSolverFailed if maxIter is exhausted without convergence.
func BrentQ(f func(float64) float64, a, b, fa, fb, tol float64, maxIter int) (float64, error) {
	// Validate bracket: opposite signs (zeros allowed).
	if fa != 0 && fb != 0 && math.Signbit(fa) == math.Signbit(fb) {
		return 0, ErrSolverFailed
	}
	if fa == 0 {
		return a, nil
	}
	if fb == 0 {
		return b, nil
	}

	// Invariant: f(a) and f(b) have opposite signs; b holds the best guess.
	// Arrange so |f(a)| >= |f(b)|.
	if math.Abs(fa) < math.Abs(fb) {
		a, b = b, a
		fa, fb = fb, fa
	}

	c, fc := a, fa // c is the previous b.
	d := b         // d is the previous c (init to b is harmless; only used with !mflag after first iter).
	mflag := true

	for i := 0; i < maxIter; i++ {
		if fb == 0 || math.Abs(b-a) < tol {
			return b, nil
		}
		var s float64
		useBisection := false

		if fa != fc && fb != fc {
			// Inverse quadratic interpolation across (a, b, c).
			s = a*fb*fc/((fa-fb)*(fa-fc)) +
				b*fa*fc/((fb-fa)*(fb-fc)) +
				c*fa*fb/((fc-fa)*(fc-fb))
		} else {
			// Secant across (a, b).
			s = b - fb*(b-a)/(fb-fa)
		}

		// Conditions from Brent (1973) that force bisection instead of IQI/secant.
		sOutOfBounds := s < (3*a+b)/4 || s > b
		if mflag {
			sOutOfBounds = sOutOfBounds || math.Abs(s-b) >= math.Abs(b-c)/2
		} else {
			sOutOfBounds = sOutOfBounds || math.Abs(s-b) >= math.Abs(c-d)/2
		}
		if mflag {
			sOutOfBounds = sOutOfBounds || math.Abs(b-c) < tol
		} else {
			sOutOfBounds = sOutOfBounds || math.Abs(c-d) < tol
		}
		if sOutOfBounds {
			s = (a + b) / 2
			useBisection = true
		}
		fs := f(s)

		// Advance history: d ← c, c ← b.
		d, c, fc = c, b, fb

		if math.Signbit(fa) != math.Signbit(fs) {
			// Root is in [a, s]; b becomes s.
			b, fb = s, fs
		} else {
			// Root is in [s, b]; a becomes s, f(a) inherits sign of f(s).
			a, fa = s, fs
		}

		// Re-establish |f(a)| >= |f(b)| so b stays the best guess.
		if math.Abs(fa) < math.Abs(fb) {
			a, b = b, a
			fa, fb = fb, fa
		}
		mflag = useBisection
	}
	return b, ErrSolverFailed
}
