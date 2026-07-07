package tvm

import (
	"errors"
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

// tol is the golden-case tolerance for non-terminating decimals.
var tol = decimal.NewFromFloat(1e-6)

func closeTo(t *testing.T, got, want decimal.Decimal, msg string) {
	t.Helper()
	if diff := got.Sub(want).Abs(); diff.GreaterThan(tol) {
		t.Errorf("%s: got %s, want %s, diff %s", msg, got, want, diff)
	}
}

func TestSimpleInterest_Golden(t *testing.T) {
	// P=100000, i=0.10, t=0.5 → 105000 exactly.
	got, err := SimpleInterest(d("100000"), d("0.10"), d("0.5"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Equal(d("105000")) {
		t.Errorf("got %s, want 105000", got)
	}
}

func TestSimpleInterest_EdgeCases(t *testing.T) {
	// t=0 → S=P
	got, err := SimpleInterest(d("1000"), d("0.5"), d("0"))
	if err != nil || !got.Equal(d("1000")) {
		t.Errorf("t=0: got %s err %v, want 1000", got, err)
	}
	// i=0 → S=P
	got, err = SimpleInterest(d("1000"), d("0"), d("5"))
	if err != nil || !got.Equal(d("1000")) {
		t.Errorf("i=0: got %s err %v, want 1000", got, err)
	}
	// P=0 → S=0
	got, err = SimpleInterest(d("0"), d("0.10"), d("1"))
	if err != nil || !got.IsZero() {
		t.Errorf("P=0: got %s err %v, want 0", got, err)
	}
	// t<0 → error
	_, err = SimpleInterest(d("1000"), d("0.10"), d("-0.5"))
	if !errors.Is(err, ErrNegativeTime) {
		t.Errorf("t<0: expected ErrNegativeTime, got %v", err)
	}
}

func TestCompoundInterest_Golden(t *testing.T) {
	// P=100000, i=0.10, m=12, t=1 → 110471.31 (to scale 2).
	got, err := CompoundInterest(d("100000"), d("0.10"), 12, d("1"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	rounded := got.Round(2)
	closeTo(t, rounded, d("110471.31"), "compound 12×1y")
}

func TestCompoundInterest_FractionalYears(t *testing.T) {
	// 6 months at 12% monthly-compounding: (1+0.01)^6 × P.
	// (1.01)^6 ≈ 1.0615201506 → 100000 × 1.0615 ≈ 106152.02
	got, err := CompoundInterest(d("100000"), d("0.12"), 12, d("0.5"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got.Round(2), d("106152.02"), "compound 0.5y")
}

func TestCompoundInterest_EdgeCases(t *testing.T) {
	// m<=0 → error
	if _, err := CompoundInterest(d("1000"), d("0.1"), 0, d("1")); !errors.Is(err, ErrInvalidCompounding) {
		t.Errorf("m=0: expected ErrInvalidCompounding, got %v", err)
	}
	if _, err := CompoundInterest(d("1000"), d("0.1"), -1, d("1")); !errors.Is(err, ErrInvalidCompounding) {
		t.Errorf("m=-1: expected ErrInvalidCompounding, got %v", err)
	}
	// t<0 → error
	if _, err := CompoundInterest(d("1000"), d("0.1"), 12, d("-1")); !errors.Is(err, ErrNegativeTime) {
		t.Errorf("t<0: expected ErrNegativeTime, got %v", err)
	}
	// t=0 → S=P (base^0 = 1)
	got, err := CompoundInterest(d("1000"), d("0.5"), 12, d("0"))
	if err != nil {
		t.Errorf("t=0: err %v", err)
	}
	if err == nil {
		closeTo(t, got, d("1000"), "compound t=0")
	}
	// i=0 → S=P
	got, err = CompoundInterest(d("1000"), d("0"), 12, d("5"))
	if err != nil {
		t.Errorf("i=0: err %v", err)
	}
	if err == nil {
		closeTo(t, got, d("1000"), "compound i=0")
	}
}

func TestEffectiveRate_Golden(t *testing.T) {
	// i_nom=0.10, m=12 → (1+0.1/12)^12 − 1 = 0.1047130674412...
	got, err := EffectiveRate(d("0.10"), 12)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got, d("0.104713"), "effective 12")
}

func TestEffectiveRate_M1_Annual(t *testing.T) {
	// m=1: effective == nominal.
	got, err := EffectiveRate(d("0.10"), 1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got, d("0.10"), "effective m=1")
}

func TestEffectiveRate_InvalidM(t *testing.T) {
	if _, err := EffectiveRate(d("0.1"), 0); !errors.Is(err, ErrInvalidCompounding) {
		t.Errorf("m=0: expected ErrInvalidCompounding, got %v", err)
	}
}

func TestFisherRealRate_Golden(t *testing.T) {
	// r_nom=0.10, π=0.08 → 1.10/1.08 − 1 ≈ 0.018519
	got, err := FisherRealRate(d("0.10"), d("0.08"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got, d("0.018519"), "fisher 10/8")
}

func TestFisherRealRate_HighInflation(t *testing.T) {
	// π > r_nom → negative real yield (valid, not an error).
	got, err := FisherRealRate(d("0.05"), d("0.08"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// 1.05/1.08 − 1 ≈ −0.027778
	closeTo(t, got, d("-0.027778"), "fisher negative")
	if !got.IsNegative() {
		t.Errorf("expected negative real yield, got %s", got)
	}
}

func TestFisherRealRate_Deflation100(t *testing.T) {
	// π = −1 → division by zero.
	_, err := FisherRealRate(d("0.10"), d("-1"))
	if !errors.Is(err, ErrDeflation100Percent) {
		t.Errorf("π=-1: expected ErrDeflation100Percent, got %v", err)
	}
}

func TestFisherRealRate_ZeroInflation(t *testing.T) {
	// π = 0 → real == nominal.
	got, err := FisherRealRate(d("0.10"), d("0"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got, d("0.10"), "fisher π=0")
}
