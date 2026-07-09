// Package deposit (test): deposit_test.go unit-tests the stateless deposit
// calculator service against golden values from mathcore/tvm and the
// business logic spec.
// Tests run without a database — the service has no storage dependency.
package deposit

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

// d is a test helper that panics on bad decimal fixtures (fail fast).
func d(s string) decimal.Decimal {
	v, err := decimal.NewFromString(s)
	if err != nil {
		panic("bad decimal fixture: " + s)
	}
	return v
}

// tol is the acceptable delta for effective-rate comparisons (irrational
// results from tvm.EffectiveRate).
var tol = decimal.NewFromFloat(1e-6)

// closeTo checks that got is within tol of want.
func closeTo(t *testing.T, got, want decimal.Decimal, msg string) {
	t.Helper()
	if diff := got.Sub(want).Abs(); diff.GreaterThan(tol) {
		t.Errorf("%s: got %s, want %s, diff %s", msg, got, want, diff)
	}
}

// -------- Compound golden --------

// TestCalculate_Compound_Golden mirrors the mathcore/tvm golden:
// P=100000, 10%, 12m, monthly cap.
//   - Maturity amount: 110471.31
//   - Effective rate: ≈ 0.104713 (within tol)
func TestCalculate_Compound_Golden(t *testing.T) {
	s := NewService()
	res, err := s.Calculate(context.Background(), Input{
		Principal:      d("100000"),
		AnnualRate:     d("0.10"),
		TermMonths:     12,
		Capitalization: CapMonthly,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}

	wantMaturity := d("110471.31")
	if !res.MaturityAmount.Equal(wantMaturity) {
		t.Errorf("maturity_amount = %s, want %s", res.MaturityAmount, wantMaturity)
	}

	wantInterest := d("10471.31")
	if !res.TotalInterest.Equal(wantInterest) {
		t.Errorf("total_interest = %s, want %s", res.TotalInterest, wantInterest)
	}

	closeTo(t, res.EffectiveRate, d("0.104713"), "effective_rate")

	if len(res.Projection) != 12 {
		t.Fatalf("projection len = %d, want 12", len(res.Projection))
	}
	// Last row balance must match maturity amount.
	last := res.Projection[len(res.Projection)-1]
	if !last.Balance.Equal(wantMaturity) {
		t.Errorf("last projection balance = %s, want %s", last.Balance, wantMaturity)
	}
	// Last cumulative interest equals total interest.
	if !last.CumulativeInterest.Equal(wantInterest) {
		t.Errorf("last cumulative_interest = %s, want %s", last.CumulativeInterest, wantInterest)
	}
}

// -------- Simple (maturity cap) --------

// TestCalculate_Simple verifies maturity (no capitalisation):
// P=100000, 10%, 6m → 105000.00 using simple interest.
func TestCalculate_Simple(t *testing.T) {
	s := NewService()
	res, err := s.Calculate(context.Background(), Input{
		Principal:      d("100000"),
		AnnualRate:     d("0.10"),
		TermMonths:     6,
		Capitalization: CapMaturity,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}

	wantMaturity := d("105000.00")
	if !res.MaturityAmount.Equal(wantMaturity) {
		t.Errorf("maturity_amount = %s, want %s", res.MaturityAmount, wantMaturity)
	}

	wantInterest := d("5000.00")
	if !res.TotalInterest.Equal(wantInterest) {
		t.Errorf("total_interest = %s, want %s", res.TotalInterest, wantInterest)
	}

	// For simple (maturity) interest, effective == nominal.
	if !res.EffectiveRate.Equal(d("0.10")) {
		t.Errorf("effective_rate = %s, want 0.10", res.EffectiveRate)
	}

	// Maturity projection: balance stays at principal throughout.
	if len(res.Projection) != 6 {
		t.Fatalf("projection len = %d, want 6", len(res.Projection))
	}
	for i, row := range res.Projection {
		if !row.Balance.Equal(d("100000")) {
			t.Errorf("row %d balance = %s, want 100000 (maturity: no cap)", i, row.Balance)
		}
	}
	// Final cumulative interest = total simple interest.
	if !res.Projection[5].CumulativeInterest.Equal(d("5000.00")) {
		t.Errorf("last cumulative_interest = %s, want 5000.00", res.Projection[5].CumulativeInterest)
	}
}

// -------- Zero rate --------

// TestCalculate_ZeroRate: zero annual rate → maturity equals principal.
func TestCalculate_ZeroRate(t *testing.T) {
	s := NewService()
	res, err := s.Calculate(context.Background(), Input{
		Principal:      d("100000"),
		AnnualRate:     d("0"),
		TermMonths:     12,
		Capitalization: CapMonthly,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if !res.MaturityAmount.Equal(d("100000")) {
		t.Errorf("maturity_amount = %s, want 100000", res.MaturityAmount)
	}
	if !res.TotalInterest.IsZero() {
		t.Errorf("total_interest = %s, want 0", res.TotalInterest)
	}
	if !res.EffectiveRate.IsZero() {
		t.Errorf("effective_rate = %s, want 0", res.EffectiveRate)
	}
}

// -------- Zero principal --------

// TestCalculate_ZeroPrincipal: zero principal → all outputs zero.
func TestCalculate_ZeroPrincipal(t *testing.T) {
	s := NewService()
	_, err := s.Calculate(context.Background(), Input{
		Principal:      d("0"),
		AnnualRate:     d("0.10"),
		TermMonths:     12,
		Capitalization: CapMonthly,
	})
	if err == nil {
		t.Fatal("expected ErrInvalidArgument for zero principal, got nil")
	}
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

// -------- Inflation (Fisher real return) --------

// TestCalculate_WithInflation: inflation > 0 produces a real return
// strictly lower than the effective rate.
func TestCalculate_WithInflation(t *testing.T) {
	s := NewService()
	res, err := s.Calculate(context.Background(), Input{
		Principal:      d("100000"),
		AnnualRate:     d("0.10"),
		TermMonths:     12,
		Capitalization: CapMonthly,
		InflationRate:  d("0.08"),
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if res.RealReturn.IsZero() {
		t.Fatal("real_return must be non-zero when inflation > 0")
	}
	if !res.RealReturn.LessThan(res.EffectiveRate) {
		t.Errorf("real_return %s must be < effective_rate %s (Fisher)", res.RealReturn, res.EffectiveRate)
	}
	// Expected: (1+0.104713)/(1+0.08)-1 ≈ 0.02288...
	closeTo(t, res.RealReturn, d("0.022882"), "real_return (Fisher)")
}

// -------- Validation errors --------

// TestCalculate_Validation verifies that ErrInvalidArgument is returned
// for each bad-input case.
func TestCalculate_Validation(t *testing.T) {
	s := NewService()
	cases := []struct {
		name string
		in   Input
	}{
		{"zero principal", Input{Principal: d("0"), AnnualRate: d("0.10"), TermMonths: 12, Capitalization: CapMonthly}},
		{"negative principal", Input{Principal: d("-1"), AnnualRate: d("0.10"), TermMonths: 12, Capitalization: CapMonthly}},
		{"zero term", Input{Principal: d("100"), AnnualRate: d("0.10"), TermMonths: 0, Capitalization: CapMonthly}},
		{"negative rate", Input{Principal: d("100"), AnnualRate: d("-0.01"), TermMonths: 12, Capitalization: CapMonthly}},
		{"bad capitalization", Input{Principal: d("100"), AnnualRate: d("0.10"), TermMonths: 12, Capitalization: CapFreq("fortnightly")}},
		{"negative inflation", Input{Principal: d("100"), AnnualRate: d("0.10"), TermMonths: 12, Capitalization: CapMonthly, InflationRate: d("-0.01")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := s.Calculate(context.Background(), c.in)
			if err == nil {
				t.Fatal("expected ErrInvalidArgument, got nil")
			}
			if !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("expected ErrInvalidArgument, got %v", err)
			}
		})
	}
}

// -------- Default cap is monthly --------

// TestCalculate_DefaultCapIsMonthly: empty Capitalization defaults to
// monthly, producing the same result as the explicit monthly golden test.
func TestCalculate_DefaultCapIsMonthly(t *testing.T) {
	s := NewService()
	res, err := s.Calculate(context.Background(), Input{
		Principal:  d("100000"),
		AnnualRate: d("0.10"),
		TermMonths: 12,
		// Capitalization intentionally empty → defaults to monthly.
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	want := d("110471.31")
	if !res.MaturityAmount.Equal(want) {
		t.Errorf("maturity_amount = %s, want %s (expected monthly default)", res.MaturityAmount, want)
	}
}

// -------- Quarterly capitalisation --------

// TestCalculate_QuarterlyCap verifies quarterly compounding:
// P=100000, 10%, 12m → P × (1 + 0.10/4)^4 ≈ 110381.29
func TestCalculate_QuarterlyCap(t *testing.T) {
	s := NewService()
	res, err := s.Calculate(context.Background(), Input{
		Principal:      d("100000"),
		AnnualRate:     d("0.10"),
		TermMonths:     12,
		Capitalization: CapQuarterly,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	wantMaturity := d("110381.29")
	if !res.MaturityAmount.Equal(wantMaturity) {
		t.Errorf("maturity_amount = %s, want %s", res.MaturityAmount, wantMaturity)
	}
	// Quarterly effective: (1+0.10/4)^4 - 1 ≈ 0.103813
	closeTo(t, res.EffectiveRate, d("0.103813"), "quarterly effective_rate")
}

// -------- Annual capitalisation --------

// TestCalculate_AnnualCap verifies annual compounding:
// P=100000, 10%, 24m → P × (1 + 0.10)^2 = 121000.00
func TestCalculate_AnnualCap(t *testing.T) {
	s := NewService()
	res, err := s.Calculate(context.Background(), Input{
		Principal:      d("100000"),
		AnnualRate:     d("0.10"),
		TermMonths:     24,
		Capitalization: CapAnnually,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	wantMaturity := d("121000.00")
	if !res.MaturityAmount.Equal(wantMaturity) {
		t.Errorf("maturity_amount = %s, want %s", res.MaturityAmount, wantMaturity)
	}
	// Annual effective = nominal (m=1).
	if !res.EffectiveRate.Equal(d("0.10")) {
		t.Errorf("effective_rate = %s, want 0.10", res.EffectiveRate)
	}
}

// -------- Disclaimer --------

// TestCalculate_Disclaimer ensures the disclaimer is always present.
func TestCalculate_Disclaimer(t *testing.T) {
	s := NewService()
	res, err := s.Calculate(context.Background(), Input{
		Principal:      d("100000"),
		AnnualRate:     d("0.10"),
		TermMonths:     12,
		Capitalization: CapMonthly,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if res.Disclaimer == "" {
		t.Error("disclaimer must not be empty")
	}
}
