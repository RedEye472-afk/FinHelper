package tax

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

// -------- DepositTax --------

func TestDepositTax_Golden2024(t *testing.T) {
	// MATH_FORMULAS.md §5.4 / §6.4: 2024, key_rate_jan1=0.16, interest=200000.
	// threshold = 1M × 0.16 = 160000. excess = 40000. tax = 40000 × 0.13 = 5200.
	r := MustLoadRules(2024)
	if !r.KeyRateJan1.Value().Equal(d("0.16")) {
		t.Fatalf("2024 key_rate_jan1: got %s, want 0.16", r.KeyRateJan1.Value())
	}
	got, err := DepositTax(r, d("200000"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Equal(d("5200")) {
		t.Errorf("deposit tax 2024: got %s, want 5200", got)
	}
}

func TestDepositTax_Golden2025(t *testing.T) {
	// 2025: key_rate_jan1=0.21 → threshold = 210000. interest=300000 → tax = 11700.
	r := MustLoadRules(2025)
	if got := Threshold(r); !got.Equal(d("210000")) {
		t.Fatalf("2025 threshold: got %s, want 210000", got)
	}
	got, err := DepositTax(r, d("300000"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Equal(d("11700")) {
		t.Errorf("deposit tax 2025: got %s, want 11700 (300000-210000)×0.13", got)
	}
}

func TestDepositTax_BelowThreshold(t *testing.T) {
	r := MustLoadRules(2024)
	// interest = threshold → 0.
	got, err := DepositTax(r, d("160000"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("below threshold: got %s, want 0", got)
	}
	// interest < threshold → 0.
	got, err = DepositTax(r, d("100000"))
	if err != nil || !got.IsZero() {
		t.Errorf("below threshold: got %s err %v, want 0", got, err)
	}
}

func TestDepositTax_NegativeInterest(t *testing.T) {
	r := MustLoadRules(2024)
	_, err := DepositTax(r, d("-1"))
	if !errors.Is(err, ErrNegativeIncome) {
		t.Errorf("negative: want ErrNegativeIncome, got %v", err)
	}
}

// -------- Loader --------

func TestLoadRules_UnsupportedYear(t *testing.T) {
	_, err := LoadRules(1999)
	if !errors.Is(err, ErrUnsupportedYear) {
		t.Errorf("unsupported year: want ErrUnsupportedYear, got %v", err)
	}
}

func TestLoadRules_AllYearsPresent(t *testing.T) {
	for _, y := range []int{2024, 2025, 2026} {
		r, err := LoadRules(y)
		if err != nil {
			t.Errorf("LoadRules(%d): %v", y, err)
			continue
		}
		if r.Year != y {
			t.Errorf("year mismatch: got %d want %d", r.Year, y)
		}
	}
}

// -------- NPD --------

func TestNPD_Golden(t *testing.T) {
	// MATH_FORMULAS.md §5.1: 500000 from individuals, 300000 from business.
	// tax = 500000 × 0.04 + 300000 × 0.06 = 20000 + 18000 = 38000.
	r := MustLoadRules(2024)
	got, err := NPD(r, d("500000"), d("300000"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Tax.Equal(d("38000")) {
		t.Errorf("NPD tax: got %s, want 38000", got.Tax)
	}
	if got.ExceedsLimit {
		t.Errorf("NPD: 800000 should not exceed 2.4M limit")
	}
}

func TestNPD_ExceedsLimit(t *testing.T) {
	r := MustLoadRules(2024)
	got, err := NPD(r, d("2000000"), d("500000"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.ExceedsLimit {
		t.Errorf("NPD: 2.5M should exceed 2.4M limit")
	}
}

func TestNPD_NegativeIncome(t *testing.T) {
	r := MustLoadRules(2024)
	_, err := NPD(r, d("-1"), d("0"))
	if !errors.Is(err, ErrNegativeIncome) {
		t.Errorf("negative: want ErrNegativeIncome, got %v", err)
	}
}

// -------- USN --------

func TestUSN_Golden(t *testing.T) {
	// MATH_FORMULAS.md §5.2 (УСН 15%): revenue 2M, expenses 1.2M → tax = 800000 × 0.15 = 120000.
	r := MustLoadRules(2024)
	got, err := USN(r, USNIncomeMinusExpenses, d("2000000"), d("1200000"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Equal(d("120000")) {
		t.Errorf("USN 15%%: got %s, want 120000", got)
	}
	// УСН 6%: revenue 2M → 120000.
	got, err = USN(r, USNIncome, d("2000000"), d("0"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Equal(d("120000")) {
		t.Errorf("USN 6%%: got %s, want 120000", got)
	}
}

func TestUSN_LossNotRefund(t *testing.T) {
	// expenses > revenue → tax = 0 (no refund).
	r := MustLoadRules(2024)
	got, err := USN(r, USNIncomeMinusExpenses, d("100000"), d("200000"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("USN loss: got %s, want 0 (no refund)", got)
	}
}

func TestUSN_NegativeInputs(t *testing.T) {
	r := MustLoadRules(2024)
	if _, err := USN(r, USNIncome, d("-1"), d("0")); !errors.Is(err, ErrNegativeIncome) {
		t.Errorf("neg revenue: want ErrNegativeIncome, got %v", err)
	}
	if _, err := USN(r, USNIncome, d("100"), d("-1")); !errors.Is(err, ErrNegativeExpenses) {
		t.Errorf("neg expenses: want ErrNegativeExpenses, got %v", err)
	}
}

// -------- NDFL --------

func TestNDFL_Golden(t *testing.T) {
	// MATH_FORMULAS.md §5.3: income 1M, child deduction 1400 × 12 = 16800.
	// tax = (1000000 − 16800) × 0.13 = 983200 × 0.13 = 127816.
	r := MustLoadRules(2024)
	deduction := ChildDeduction(r, 1).Mul(decimal.NewFromInt(12))
	got, err := NDFL(r, d("1000000"), deduction)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Equal(d("127816")) {
		t.Errorf("NDFL: got %s, want 127816", got)
	}
}

func TestNDFL_ProgressiveHighRate(t *testing.T) {
	// Income 7M, no deductions: 5M × 0.13 + 2M × 0.15 = 650000 + 300000 = 950000.
	r := MustLoadRules(2024)
	got, err := NDFL(r, d("7000000"), d("0"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.Equal(d("950000")) {
		t.Errorf("NDFL progressive: got %s, want 950000", got)
	}
}

func TestNDFL_DeductionsExceedIncome(t *testing.T) {
	r := MustLoadRules(2024)
	got, err := NDFL(r, d("100000"), d("200000"))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("NDFL deductions>income: got %s, want 0", got)
	}
}

func TestChildDeduction(t *testing.T) {
	r := MustLoadRules(2024)
	// 1 child: 1400.
	if got := ChildDeduction(r, 1); !got.Equal(d("1400")) {
		t.Errorf("1 child: got %s, want 1400", got)
	}
	// 2 children: 2 × 1400 = 2800.
	if got := ChildDeduction(r, 2); !got.Equal(d("2800")) {
		t.Errorf("2 children: got %s, want 2800", got)
	}
	// 3 children: 2 × 1400 + 1 × 3000 = 5800.
	if got := ChildDeduction(r, 3); !got.Equal(d("5800")) {
		t.Errorf("3 children: got %s, want 5800", got)
	}
	// 4 children: 2 × 1400 + 2 × 3000 = 8800.
	if got := ChildDeduction(r, 4); !got.Equal(d("8800")) {
		t.Errorf("4 children: got %s, want 8800", got)
	}
	// 0 children: 0.
	if got := ChildDeduction(r, 0); !got.IsZero() {
		t.Errorf("0 children: got %s, want 0", got)
	}
}
