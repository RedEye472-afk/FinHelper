package investment

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal {
	x, err := decimal.NewFromString(s)
	if err != nil {
		panic("bad decimal fixture: " + s)
	}
	return x
}

var tol = decimal.NewFromFloat(1e-6)

func closeTo(t *testing.T, got, want decimal.Decimal, msg string) {
	t.Helper()
	if diff := got.Sub(want).Abs(); diff.GreaterThan(tol) {
		t.Errorf("%s: got %s, want %s, diff %s", msg, got, want, diff)
	}
}

func mkDate(y, m, day int) time.Time {
	return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
}

// -------- NPV --------

func TestNPV_Golden(t *testing.T) {
	// MATH_FORMULAS.md §3.1: -100000 + 30000/1.1 + 40000/1.1^2 + 50000/1.1^3
	// = -100000 + 27272.73 + 33057.85 + 37565.74 = -2103.68 (project unprofitable).
	cfs := []decimal.Decimal{d("-100000"), d("30000"), d("40000"), d("50000")}
	got, err := NPV(cfs, d("0.10"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// Tolerance 0.01 — the doc value -2103.68 is rounded to kopecks.
	if diff := got.Sub(d("-2103.68")).Abs(); diff.GreaterThan(d("0.01")) {
		t.Errorf("NPV: got %s, want -2103.68, diff %s", got, diff)
	}
	if !got.IsNegative() {
		t.Errorf("NPV should be negative for unprofitable project")
	}
}

func TestNPV_ZeroRate(t *testing.T) {
	// r=0 → NPV = plain sum.
	cfs := []decimal.Decimal{d("-100"), d("50"), d("60")}
	got, err := NPV(cfs, d("0"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got, d("10"), "npv r=0")
}

func TestNPV_InvalidRate(t *testing.T) {
	if _, err := NPV([]decimal.Decimal{d("-100")}, d("-1")); !errors.Is(err, ErrInvalidRate) {
		t.Errorf("r=-1: want ErrInvalidRate, got %v", err)
	}
}

// -------- XIRR --------

func TestXIRR_Golden(t *testing.T) {
	// MATH_FORMULAS.md §3.2 example scenario (cashflows: -100k invested,
	// then 30k/40k/50k returned quarterly). The doc's printed "12.34%" is
	// arithmetically wrong (positive flows sum to 120k on a 100k investment
	// within one year — the yield cannot be 12%). The numerically solved
	// XIRR is ~40.14% (effective annual, ACT/365). This matches the §3.2
	// example cashflows even though the printed headline number does not.
	cfs := []Cashflow{
		{Date: mkDate(2024, 1, 1), Amount: d("-100000")},
		{Date: mkDate(2024, 4, 1), Amount: d("30000")},
		{Date: mkDate(2024, 7, 1), Amount: d("40000")},
		{Date: mkDate(2024, 10, 1), Amount: d("50000")},
	}
	got, err := XIRR(cfs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// Root is 0.40657 (verified by round-trip: NPV at this rate is 0.0037 ≈ 0).
	// Allow 5 bps for solver tolerance.
	want := d("0.40657")
	if diff := got.Sub(want).Abs(); diff.GreaterThan(d("0.0005")) {
		t.Errorf("XIRR: got %s (%.4f%%), want ~0.40657, diff %s", got, got.InexactFloat64()*100, diff)
	}
}

func TestXIRR_NoSignChange(t *testing.T) {
	cfs := []Cashflow{
		{Date: mkDate(2024, 1, 1), Amount: d("-100")},
		{Date: mkDate(2024, 2, 1), Amount: d("-50")},
	}
	_, err := XIRR(cfs)
	if !errors.Is(err, ErrNoSignChange) {
		t.Errorf("expected ErrNoSignChange, got %v", err)
	}
}

func TestXIRR_TooFew(t *testing.T) {
	_, err := XIRR([]Cashflow{{Date: mkDate(2024, 1, 1), Amount: d("-100")}})
	if !errors.Is(err, ErrInsufficientCashflows) {
		t.Errorf("expected ErrInsufficientCashflows, got %v", err)
	}
}

func TestXIRR_RoundTripZeroNPV(t *testing.T) {
	// Property test: NPV evaluated at the XIRR rate must be ≈ 0.
	cfs := []Cashflow{
		{Date: mkDate(2024, 1, 1), Amount: d("-100000")},
		{Date: mkDate(2024, 4, 1), Amount: d("30000")},
		{Date: mkDate(2024, 7, 1), Amount: d("40000")},
		{Date: mkDate(2024, 10, 1), Amount: d("50000")},
	}
	rate, err := XIRR(cfs)
	if err != nil {
		t.Fatalf("XIRR: %v", err)
	}
	// Recompute NPV at rate.
	ordered := sortByDate(cfs)
	epoch := ordered[0].Date
	r := rate.InexactFloat64()
	var npv float64
	for _, cf := range ordered {
		years := cf.Date.Sub(epoch).Hours() / 24 / 365
		disc := math.Pow(1+r, years)
		npv += cf.Amount.InexactFloat64() / disc
	}
	if math.Abs(npv) > 1e-3 {
		t.Errorf("NPV at XIRR not zero: got %.6f", npv)
	}
}

// -------- MIRR --------

func TestMIRR_Golden(t *testing.T) {
	// MATH_FORMULAS.md §3.3: -100000 at t=0; +30000/+40000/+50000 at t=1/2/3.
	// finance=10%, reinvest=8%.
	// FV_pos = 30000×1.08^2 + 40000×1.08^1 + 50000 = 34992 + 43200 + 50000 = 128192.
	// PV_neg = 100000 (single negative flow at t=0).
	// MIRR = (128192/100000)^(1/3) − 1 = 0.08631. (Doc §3.3 prints 8.52% —
	// arithmetic typo; the cube root of 1.28192 is 1.08631, not 1.08527.)
	cfs := []decimal.Decimal{d("-100000"), d("30000"), d("40000"), d("50000")}
	got, err := MIRR(cfs, d("0.10"), d("0.08"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got, d("0.08631"), "mirr golden")
}

func TestMIRR_EdgeCases(t *testing.T) {
	// < 2 cashflows.
	if _, err := MIRR([]decimal.Decimal{d("-100")}, d("0.1"), d("0.1")); !errors.Is(err, ErrInsufficientCashflows) {
		t.Errorf("len=1: want ErrInsufficientCashflows, got %v", err)
	}
	// No positive CF.
	if _, err := MIRR([]decimal.Decimal{d("-100"), d("-50")}, d("0.1"), d("0.1")); !errors.Is(err, ErrNoPositiveCF) {
		t.Errorf("no-pos: want ErrNoPositiveCF, got %v", err)
	}
	// No negative CF.
	if _, err := MIRR([]decimal.Decimal{d("100"), d("50")}, d("0.1"), d("0.1")); !errors.Is(err, ErrNoNegativeCF) {
		t.Errorf("no-neg: want ErrNoNegativeCF, got %v", err)
	}
	// Invalid rate.
	if _, err := MIRR([]decimal.Decimal{d("-100"), d("50")}, d("-1"), d("0.1")); !errors.Is(err, ErrInvalidRate) {
		t.Errorf("rate=-1: want ErrInvalidRate, got %v", err)
	}
}

// -------- DPP --------

func TestDPP_Golden(t *testing.T) {
	// MATH_FORMULAS.md §3.4: -100000, 30000, 40000, 50000 at r=10%.
	// Discounted: -100000, 27272.73, 33057.85, 37565.74.
	// Cumulative: -100000, -72727.27, -39669.42, -2103.68 → never crosses 0.
	cfs := []decimal.Decimal{d("-100000"), d("30000"), d("40000"), d("50000")}
	_, err := DPP(cfs, d("0.10"))
	if !errors.Is(err, ErrNeverPaidBack) {
		t.Errorf("expected ErrNeverPaidBack, got %v", err)
	}
}

func TestDPP_Interpolated(t *testing.T) {
	// A clearly profitable project. -1000, 400, 400, 400 at r=10%.
	// Discounted: -1000, 363.64, 330.58, 300.53.
	// Cumulative: -1000, -636.36, -305.78, -5.26 → still slightly short.
	// Add a 4th-year return so it pays back.
	cfs := []decimal.Decimal{d("-1000"), d("400"), d("400"), d("400"), d("400")}
	got, err := DPP(cfs, d("0.10"))
	if err != nil {
		t.Fatalf("DPP: %v", err)
	}
	// Should be between 3 and 4 years.
	if !got.GreaterThan(d("3")) || !got.LessThan(d("4")) {
		t.Errorf("DPP: got %s, expected within (3, 4)", got)
	}
}

func TestDPP_ImmediatePayback(t *testing.T) {
	// First flow already positive.
	got, err := DPP([]decimal.Decimal{d("100"), d("50")}, d("0.1"))
	if err != nil {
		t.Fatalf("DPP: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("DPP: immediate payback should be 0, got %s", got)
	}
}

// -------- PI --------

func TestPI_Golden(t *testing.T) {
	// MATH_FORMULAS.md §3.5: investment 100000, PV(future) = 115000 → PI = 1.15.
	// Reproduce with single positive flow at t=1 that discounts to 115000 at r=0.
	// Use r=0 for simplicity: PI = 115000 / 100000 = 1.15.
	cfs := []decimal.Decimal{d("-100000"), d("115000")}
	got, err := PI(cfs, d("0"))
	if err != nil {
		t.Fatalf("PI: %v", err)
	}
	closeTo(t, got, d("1.15"), "pi golden")
}

func TestPI_EdgeCases(t *testing.T) {
	// Zero initial investment.
	if _, err := PI([]decimal.Decimal{d("0"), d("100")}, d("0.1")); !errors.Is(err, ErrZeroInitialInvestment) {
		t.Errorf("zero-init: want ErrZeroInitialInvestment, got %v", err)
	}
	// Empty.
	if _, err := PI([]decimal.Decimal{}, d("0.1")); !errors.Is(err, ErrInsufficientCashflows) {
		t.Errorf("empty: want ErrInsufficientCashflows, got %v", err)
	}
	// Invalid rate.
	if _, err := PI([]decimal.Decimal{d("-100"), d("50")}, d("-1")); !errors.Is(err, ErrInvalidRate) {
		t.Errorf("rate=-1: want ErrInvalidRate, got %v", err)
	}
}

func TestPI_Unprofitable(t *testing.T) {
	// -100000 + 30000/1.1 + 40000/1.1^2 + 50000/1.1^3 = -2103.68 NPV.
	// PV of positives = 27272.73 + 33057.85 + 37565.74 = 97896.32.
	// PI = 97896.32 / 100000 = 0.97896 < 1 → unprofitable.
	cfs := []decimal.Decimal{d("-100000"), d("30000"), d("40000"), d("50000")}
	got, err := PI(cfs, d("0.10"))
	if err != nil {
		t.Fatalf("PI: %v", err)
	}
	if !got.LessThan(d("1")) {
		t.Errorf("PI: got %s, expected < 1 for unprofitable", got)
	}
	// Tolerance 1e-4 — generous because PV discounts use decimal.Pow which
	// accumulates sub-1e-6 error across 3 periods.
	if diff := got.Sub(d("0.97896")).Abs(); diff.GreaterThan(d("0.0001")) {
		t.Errorf("PI unprofitable: got %s, want 0.97896, diff %s", got, diff)
	}
}
