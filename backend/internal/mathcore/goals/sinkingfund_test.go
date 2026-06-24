package goals

import (
	"testing"

	"github.com/shopspring/decimal"
)

// decimalsEqual проверяет равенство с допуском 1e-6 (как в tvm-тестах).
func decimalsEqual(a, b decimal.Decimal) bool {
	diff := a.Sub(b).Abs()
	return diff.LessThan(decimal.NewFromFloat(1e-6))
}

// moneyClose — допуск до копейки (0.01), для golden-эталонов, округляемых
// вручную (как в credit-тестах, где иррациональные числа округляются HALF_AWAY).
func moneyClose(a, b decimal.Decimal) bool {
	diff := a.Sub(b).Abs()
	return diff.LessThanOrEqual(decimal.NewFromFloat(0.01))
}

func TestSolveFutureValue_CompoundGrowth(t *testing.T) {
	// P=100000, A=10000, i=0.01 (годовая 0.12/12), n=12 мес.
	// (1.01)^12 = 1.12682503...
	// P·factor = 100000·1.12682503 = 112682.503
	// A·(factor−1)/i = 10000·0.12682503/0.01 = 126825.03
	// S = 239507.53
	P := decimal.NewFromInt(100000)
	A := decimal.NewFromInt(10000)
	i := decimal.NewFromFloat(0.01)
	n := 12
	got, err := SolveFutureValue(P, A, i, n)
	if err != nil {
		t.Fatalf("SolveFutureValue: %v", err)
	}
	want, errW := decimal.NewFromString("239507.53") // пересчитано вручную
	if errW != nil {
		t.Fatalf("bad want literal: %v", errW)
	}
	if !moneyClose(got, want) {
		t.Errorf("SolveFutureValue = %s, want near %s", got, want)
	}
}

func TestSolveFutureValue_ZeroRate_Fallback(t *testing.T) {
	// i=0 → S = P + A·n = 100000 + 10000·12 = 220000
	got, err := SolveFutureValue(decimal.NewFromInt(100000), decimal.NewFromInt(10000), decimal.Zero, 12)
	if err != nil {
		t.Fatalf("SolveFutureValue: %v", err)
	}
	if !got.Equal(decimal.NewFromInt(220000)) {
		t.Errorf("zero-rate fallback = %s, want 220000", got)
	}
}

func TestSolveFutureValue_ZeroPeriods_ReturnsPrincipal(t *testing.T) {
	// n=0 → S = P (нечего наращивать)
	got, err := SolveFutureValue(decimal.NewFromInt(100000), decimal.NewFromInt(10000), decimal.NewFromFloat(0.01), 0)
	if err != nil {
		t.Fatalf("SolveFutureValue: %v", err)
	}
	if !got.Equal(decimal.NewFromInt(100000)) {
		t.Errorf("n=0 = %s, want 100000 (principal only)", got)
	}
}

func TestSolveFutureValue_NegativePeriods_Error(t *testing.T) {
	_, err := SolveFutureValue(decimal.NewFromInt(100000), decimal.NewFromInt(10000), decimal.NewFromFloat(0.01), -1)
	if err == nil {
		t.Errorf("expected ErrInvalidPeriods for n<0")
	}
}
