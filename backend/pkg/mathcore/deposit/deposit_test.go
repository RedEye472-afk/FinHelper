package deposit

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal {
	x, err := decimal.NewFromString(s)
	if err != nil {
		panic("bad decimal fixture: " + s)
	}
	return x
}

// tol is the tolerance for rate comparisons (non-money values).
var tol = decimal.NewFromFloat(1e-6)

func closeTo(t *testing.T, got, want decimal.Decimal, msg string) {
	t.Helper()
	if diff := got.Sub(want).Abs(); diff.GreaterThan(tol) {
		t.Errorf("%s: got %s, want %s, diff %s", msg, got, want, diff)
	}
}

// ---- Golden cases from MATH_FORMULAS.md ---- //

// §1.1: Simple interest — P=100000, i=0.10, t=0.5 (6 mo) → S=105000.
func TestCalculate_SimpleInterest_Golden(t *testing.T) {
	res, err := Calculate(d("100000"), d("0.10"), 6, CapMaturity, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.MaturityAmount.Equal(d("105000")) {
		t.Errorf("MaturityAmount = %s, want 105000", res.MaturityAmount)
	}
	if !res.TotalInterest.Equal(d("5000")) {
		t.Errorf("TotalInterest = %s, want 5000", res.TotalInterest)
	}
	// Projection: 5 months at 100000, final month at 105000.
	if len(res.Projection) != 6 {
		t.Fatalf("Projection len = %d, want 6", len(res.Projection))
	}
	for m := 0; m < 5; m++ {
		if !res.Projection[m].Balance.Equal(d("100000")) {
			t.Errorf("Projection[%d].Balance = %s, want 100000", m, res.Projection[m].Balance)
		}
	}
	if !res.Projection[5].Balance.Equal(d("105000")) {
		t.Errorf("Projection[5].Balance = %s, want 105000", res.Projection[5].Balance)
	}
	if !res.Projection[5].Interest.Equal(d("5000")) {
		t.Errorf("Projection[5].Interest = %s, want 5000", res.Projection[5].Interest)
	}
}

// §1.2: Compound monthly — P=100000, i=0.10, m=12, t=1 → S=110471.31.
func TestCalculate_CompoundMonthly_Golden(t *testing.T) {
	res, err := Calculate(d("100000"), d("0.10"), 12, CapMonthly, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.MaturityAmount.Equal(d("110471.31")) {
		t.Errorf("MaturityAmount = %s, want 110471.31", res.MaturityAmount)
	}
	closeTo(t, res.TotalInterest, d("10471.31"), "TotalInterest")
}

// §1.3: Effective rate — i_nom=0.10, m=12 → i_eff≈0.104713.
func TestCalculate_EffectiveRate_Golden(t *testing.T) {
	res, err := Calculate(d("100000"), d("0.10"), 12, CapMonthly, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, res.EffectiveRate, d("0.104713"), "EffectiveRate")
}

// §1.4: Fisher real return — r_nom=0.10, π=0.08 → r_real≈0.018519.
func TestCalculate_FisherRealReturn_Golden(t *testing.T) {
	res, err := Calculate(d("100000"), d("0.10"), 12, CapMonthly, d("0.08"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, res.RealReturn, d("0.018519"), "RealReturn")
}

// ---- Edge cases ---- //

func TestCalculate_ZeroRate(t *testing.T) {
	res, err := Calculate(d("100000"), d("0"), 12, CapMonthly, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.MaturityAmount.Equal(d("100000")) {
		t.Errorf("MaturityAmount = %s, want 100000", res.MaturityAmount)
	}
	if !res.TotalInterest.IsZero() {
		t.Errorf("TotalInterest = %s, want 0", res.TotalInterest)
	}
	// Effective rate must be zero when nominal is zero.
	if !res.EffectiveRate.IsZero() {
		t.Errorf("EffectiveRate = %s, want 0", res.EffectiveRate)
	}
	for _, row := range res.Projection {
		if !row.Balance.Equal(d("100000")) {
			t.Errorf("ZeroRate: balance %s at month %d, want 100000", row.Balance, row.Month)
		}
	}
}

func TestCalculate_ZeroPrincipal(t *testing.T) {
	res, err := Calculate(d("0"), d("0.10"), 12, CapMonthly, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.MaturityAmount.IsZero() {
		t.Errorf("MaturityAmount = %s, want 0", res.MaturityAmount)
	}
	for _, row := range res.Projection {
		if !row.Balance.IsZero() {
			t.Errorf("ZeroPrincipal: balance %s at month %d, want 0", row.Balance, row.Month)
		}
	}
}

func TestCalculate_InvalidTerm(t *testing.T) {
	_, err := Calculate(d("100000"), d("0.10"), 0, CapMonthly, decimal.Zero)
	if err != ErrInvalidTerm {
		t.Errorf("expected ErrInvalidTerm, got %v", err)
	}
}

func TestCalculate_InvalidCapFreq(t *testing.T) {
	_, err := Calculate(d("100000"), d("0.10"), 12, CapFreq(999), decimal.Zero)
	if err == nil {
		t.Error("expected error for invalid cap freq, got nil")
	}
}

func TestCalculate_ZeroInflationSkipsFisher(t *testing.T) {
	res, err := Calculate(d("100000"), d("0.10"), 12, CapMonthly, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.RealReturn.IsZero() {
		t.Errorf("RealReturn = %s, want 0 when inflation=0", res.RealReturn)
	}
}

func TestCalculate_QuarterlyCap(t *testing.T) {
	// P=100000, i=0.12, m=4 (quarterly), t=1 → S=100000*(1+0.12/4)^4
	// = 100000*(1.03)^4 = 100000*1.12550881 = 112550.881...
	// Rounded to 2: 112550.88
	res, err := Calculate(d("100000"), d("0.12"), 12, CapQuarterly, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.MaturityAmount.Equal(d("112550.88")) {
		t.Errorf("MaturityAmount = %s, want 112550.88", res.MaturityAmount)
	}
	// Projection should have 12 rows.
	if len(res.Projection) != 12 {
		t.Fatalf("Projection len = %d, want 12", len(res.Projection))
	}
	// Capitalisation at months 3, 6, 9, 12.
	// After month 3: balance = 100000 * 1.03 = 103000
	if !res.Projection[2].Balance.Equal(d("103000")) {
		t.Errorf("Quarter cap: month 3 balance = %s, want 103000", res.Projection[2].Balance)
	}
	// After month 6: balance = 103000 * 1.03 = 106090
	if !res.Projection[5].Balance.Equal(d("106090")) {
		t.Errorf("Quarter cap: month 6 balance = %s, want 106090", res.Projection[5].Balance)
	}
	// Month 4 (no cap): balance same as month 3
	if !res.Projection[3].Balance.Equal(res.Projection[2].Balance) {
		t.Errorf("Quarter cap: month 4 balance changed without cap")
	}
}

func TestCalculate_AnnualCap(t *testing.T) {
	// P=100000, i=0.12, m=1 (annual), t=1 → S=100000*1.12 = 112000
	res, err := Calculate(d("100000"), d("0.12"), 12, CapAnnually, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.MaturityAmount.Equal(d("112000")) {
		t.Errorf("MaturityAmount = %s, want 112000", res.MaturityAmount)
	}
	// Months 1-11: balance = 100000, month 12: balance = 112000
	for m := 0; m < 11; m++ {
		if !res.Projection[m].Balance.Equal(d("100000")) {
			t.Errorf("Annual cap: month %d balance = %s, want 100000", m+1, res.Projection[m].Balance)
		}
	}
	if !res.Projection[11].Balance.Equal(d("112000")) {
		t.Errorf("Annual cap: month 12 balance = %s, want 112000", res.Projection[11].Balance)
	}
}

// Verify projection builds correct number of rows for any term length.
func TestCalculate_ProjectionLength(t *testing.T) {
	for _, months := range []int{1, 6, 12, 24, 36} {
		res, err := Calculate(d("1000"), d("0.05"), months, CapMonthly, decimal.Zero)
		if err != nil {
			t.Fatalf("months=%d: err=%v", months, err)
		}
		if len(res.Projection) != months {
			t.Errorf("months=%d: got %d rows, want %d", months, len(res.Projection), months)
		}
	}
}

// Cumulative interest should be non-decreasing.
func TestCalculate_CumulativeNonDecreasing(t *testing.T) {
	res, err := Calculate(d("100000"), d("0.10"), 12, CapMonthly, decimal.Zero)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	var prev decimal.Decimal
	for _, row := range res.Projection {
		if row.CumulativeInterest.LessThan(prev) {
			t.Errorf("CumulativeInterest decreased at month %d: %s < %s", row.Month, row.CumulativeInterest, prev)
		}
		prev = row.CumulativeInterest
	}
}

// Negative rate should error.
func TestCalculate_NegativeRate(t *testing.T) {
	_, err := Calculate(d("100000"), d("-0.10"), 12, CapMonthly, decimal.Zero)
	if err != ErrInvalidRate {
		t.Errorf("expected ErrInvalidRate, got %v", err)
	}
}
