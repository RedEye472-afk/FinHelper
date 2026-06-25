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

func TestSolveContribution_Compound(t *testing.T) {
	// Симметричный к SolveFutureValue: P=100000, S=239507.53, i=0.01, n=12 → A≈10000
	P := decimal.NewFromInt(100000)
	S, _ := decimal.NewFromString("239507.53")
	i := decimal.NewFromFloat(0.01)
	got, err := SolveContribution(P, S, i, 12)
	if err != nil {
		t.Fatalf("SolveContribution: %v", err)
	}
	// Возврат должно дать ~10000 (с точностью до копеек округления эталона)
	if !moneyClose(got, decimal.NewFromInt(10000)) {
		t.Errorf("SolveContribution = %s, want ~10000", got)
	}
}

func TestSolveContribution_ZeroRate_Fallback(t *testing.T) {
	// i=0 → A = (S−P)/n = (200000−100000)/12 = 8333.33...
	got, err := SolveContribution(decimal.NewFromInt(100000), decimal.NewFromInt(200000), decimal.Zero, 12)
	if err != nil {
		t.Fatalf("SolveContribution: %v", err)
	}
	want, _ := decimal.NewFromString("8333.33")
	if !moneyClose(got, want) {
		t.Errorf("zero-rate fallback = %s, want ~8333.33", got)
	}
}

func TestSolveContribution_NegativePeriods_Error(t *testing.T) {
	_, err := SolveContribution(decimal.NewFromInt(100000), decimal.NewFromInt(200000), decimal.NewFromFloat(0.01), -1)
	if err == nil {
		t.Errorf("expected ErrInvalidPeriods for n<0")
	}
}

func TestSolveContribution_AlreadyReached_Zero(t *testing.T) {
	// P=200000, S=100000, n=12 → grown > S, A должно быть 0 (не отрицательным)
	got, err := SolveContribution(decimal.NewFromInt(200000), decimal.NewFromInt(100000), decimal.NewFromFloat(0.01), 12)
	if err != nil {
		t.Fatalf("SolveContribution: %v", err)
	}
	if got.IsNegative() {
		t.Errorf("contribution must not be negative when target reached, got %s", got)
	}
}

func TestSolveTerm_BasicCase(t *testing.T) {
	// P=0, S=1000000, A=50000, i=0.01 → ln(1.2)/ln(1.01) = 0.18232/0.00995 ≈ 18.32 мес
	got, err := SolveTerm(decimal.Zero, decimal.NewFromInt(1000000), decimal.NewFromInt(50000), decimal.NewFromFloat(0.01))
	if err != nil {
		t.Fatalf("SolveTerm: %v", err)
	}
	// Эталон ≈18.32 (между 18 и 19)
	if got.LessThan(decimal.NewFromInt(18)) || got.GreaterThan(decimal.NewFromInt(19)) {
		t.Errorf("SolveTerm = %s, want between 18 and 19", got)
	}
}

func TestSolveTerm_Unreachable_ContributionTooSmall(t *testing.T) {
	// P=1000000, i=0.50 (50%/мес), A=1 → A < P·i (1 < 500000) → цель убегает
	_, err := SolveTerm(decimal.NewFromInt(1000000), decimal.NewFromInt(2000000), decimal.NewFromInt(1), decimal.NewFromFloat(0.50))
	if err != ErrUnreachable {
		t.Errorf("expected ErrUnreachable, got %v", err)
	}
}

func TestSolveTerm_ZeroRate_Error(t *testing.T) {
	// i=0 → ln(1)=0, деление на 0 → ErrInvalidRate
	_, err := SolveTerm(decimal.Zero, decimal.NewFromInt(1000000), decimal.NewFromInt(50000), decimal.Zero)
	if err != ErrInvalidRate {
		t.Errorf("expected ErrInvalidRate for i=0, got %v", err)
	}
}

func TestSolveTerm_AlreadyReached_Zero(t *testing.T) {
	// P >= S → уже накопили, n=0
	got, err := SolveTerm(decimal.NewFromInt(1000000), decimal.NewFromInt(500000), decimal.NewFromInt(100), decimal.NewFromFloat(0.01))
	if err != nil {
		t.Fatalf("SolveTerm: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("expected n=0 when P>=S, got %s", got)
	}
}

func TestInflateTarget_PositiveInflation(t *testing.T) {
	// S=1000000, π=0.06, n=24 мес (2 года) → 1000000·(1.06)^2 = 1123600
	got, err := InflateTarget(decimal.NewFromInt(1000000), decimal.NewFromFloat(0.06), 24)
	if err != nil {
		t.Fatalf("InflateTarget: %v", err)
	}
	want, _ := decimal.NewFromString("1123600")
	if !moneyClose(got, want) {
		t.Errorf("InflateTarget = %s, want near %s", got, want)
	}
}

func TestInflateTarget_PartialYear(t *testing.T) {
	// n=6 мес → степень 0.5: (1.06)^0.5 = √1.06 ≈ 1.0295630141
	// 1000000·1.0295630141 ≈ 1029563.01
	got, err := InflateTarget(decimal.NewFromInt(1000000), decimal.NewFromFloat(0.06), 6)
	if err != nil {
		t.Fatalf("InflateTarget: %v", err)
	}
	want, _ := decimal.NewFromString("1029563.01")
	if !moneyClose(got, want) {
		t.Errorf("InflateTarget partial-year = %s, want near %s", got, want)
	}
}

func TestInflateTarget_ZeroInflation_Unchanged(t *testing.T) {
	got, err := InflateTarget(decimal.NewFromInt(1000000), decimal.Zero, 24)
	if err != nil {
		t.Fatalf("InflateTarget: %v", err)
	}
	if !got.Equal(decimal.NewFromInt(1000000)) {
		t.Errorf("zero inflation should return S unchanged, got %s", got)
	}
}

func TestInflateTarget_ZeroMonths_Unchanged(t *testing.T) {
	got, err := InflateTarget(decimal.NewFromInt(1000000), decimal.NewFromFloat(0.06), 0)
	if err != nil {
		t.Fatalf("InflateTarget: %v", err)
	}
	if !got.Equal(decimal.NewFromInt(1000000)) {
		t.Errorf("zero months should return S unchanged, got %s", got)
	}
}

func TestInflateTarget_Deflation100_Error(t *testing.T) {
	_, err := InflateTarget(decimal.NewFromInt(1000000), decimal.NewFromInt(-1), 24)
	if err != ErrDeflation100Percent {
		t.Errorf("expected ErrDeflation100Percent, got %v", err)
	}
}
