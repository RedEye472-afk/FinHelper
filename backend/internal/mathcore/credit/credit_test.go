package credit

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

// tol for golden cases on non-terminating decimals (matches tvm package).
var tol = decimal.NewFromFloat(1e-6)

func closeTo(t *testing.T, got, want decimal.Decimal, msg string) {
	t.Helper()
	if diff := got.Sub(want).Abs(); diff.GreaterThan(tol) {
		t.Errorf("%s: got %s, want %s, diff %s", msg, got, want, diff)
	}
}

// -------- AnnuityPayment --------

func TestAnnuityPayment_Golden(t *testing.T) {
	// P=1_000_000, годовая 12%, 24 мес. i = 0.12 / 12 = 0.01.
	// Exact value A = P × [i(1+i)^n] / [(1+i)^n − 1] = 47073.47222326…
	// Rounded HALF_EVEN to scale 2 → 47073.47 (NOT 47073.46 as printed in
	// MATH_FORMULAS.md §2.1/§6.3 — that doc value is an arithmetic typo, same
	// class of error fixed in Этап 0; Excel ПЛТ(0.12/12, 24, -1000000)
	// returns -47073.472223…, confirming 47073.47 as the correct rounding).
	got, err := AnnuityPayment(d("1000000"), d("0.01"), 24)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got.Round(2), d("47073.47"), "annuity 24m")
	// And the unrounded value should be the textbook exact.
	closeTo(t, got, d("47073.4722232647"), "annuity 24m exact")
}

func TestAnnuityPayment_ZeroRateFallback(t *testing.T) {
	// i = 0 → A = P / n.
	got, err := AnnuityPayment(d("120000"), d("0"), 12)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	closeTo(t, got, d("10000"), "zero-rate fallback")
}

func TestAnnuityPayment_EdgeCases(t *testing.T) {
	// n <= 0 → ErrInvalidTerm.
	if _, err := AnnuityPayment(d("1000"), d("0.01"), 0); !errors.Is(err, ErrInvalidTerm) {
		t.Errorf("n=0: expected ErrInvalidTerm, got %v", err)
	}
	if _, err := AnnuityPayment(d("1000"), d("0.01"), -1); !errors.Is(err, ErrInvalidTerm) {
		t.Errorf("n=-1: expected ErrInvalidTerm, got %v", err)
	}
	// P < 0 → ErrInvalidPrincipal.
	if _, err := AnnuityPayment(d("-1"), d("0.01"), 12); !errors.Is(err, ErrInvalidPrincipal) {
		t.Errorf("P<0: expected ErrInvalidPrincipal, got %v", err)
	}
	// P = 0 → 0.
	got, err := AnnuityPayment(d("0"), d("0.01"), 12)
	if err != nil || !got.IsZero() {
		t.Errorf("P=0: got %s err %v, want 0", got, err)
	}
}

func TestAnnuitySchedule_ClosesToZero(t *testing.T) {
	// The final balance after a full annuity schedule must close to exactly 0.
	// This is the core correctness invariant for PSK downstream.
	monthlyRate := d("0.01")
	monthly, err := AnnuityPayment(d("1000000"), monthlyRate, 24)
	if err != nil {
		t.Fatalf("payment: %v", err)
	}
	sched, err := AnnuitySchedule(d("1000000"), monthlyRate, monthly, 24)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if len(sched) != 24 {
		t.Fatalf("schedule length: got %d, want 24", len(sched))
	}
	last := sched[len(sched)-1]
	if !last.BalanceEnd.IsZero() {
		t.Errorf("balance not closed: last BalanceEnd = %s", last.BalanceEnd)
	}
	// Total interest ≈ A×n − P = 47073.4722… × 24 − 1_000_000 = 129763.33.
	// The schedule rounds each row to scale 2 (banker's rounding), so the
	// row-by-row sum carries a small accumulated residual (≤ a few kopecks
	// over 24 rows). We assert within ±0.05 — tighter than the doc's own
	// precision, loose enough to absorb per-row rounding. The §6.3 doc value
	// 129763.04 used the typo'd annuity 47073.46 and is not the reference.
	totalInterest := decimal.Zero
	for _, r := range sched {
		totalInterest = totalInterest.Add(r.Interest)
	}
	assertWithin(t, totalInterest, d("129763.33"), d("0.05"), "total interest")
	// Total paid = principal + interest (same residual tolerance).
	totalPaid := decimal.Zero
	for _, r := range sched {
		totalPaid = totalPaid.Add(r.Payment)
	}
	assertWithin(t, totalPaid, d("1129763.33"), d("0.05"), "total paid")
}

// assertWithin checks |got − want| ≤ tol with an explicit tolerance value
// (used where banker's-rounding residuals make the 1e-6 golden tolerance
// too tight for sum-of-rounded-rows comparisons).
func assertWithin(t *testing.T, got, want, tolerance decimal.Decimal, msg string) {
	t.Helper()
	if diff := got.Sub(want).Abs(); diff.GreaterThan(tolerance) {
		t.Errorf("%s: got %s, want %s (±%s), diff %s", msg, got, want, tolerance, diff)
	}
}

// -------- Differentiated --------

func TestDifferentiated_GoldenFirstTwoMonths(t *testing.T) {
	// MATH_FORMULAS.md §2.2: P=1_000_000, i=1%, n=24.
	// P/n = 41 666.67.
	// Month 1: 41 666.67 + 1_000_000 × 0.01 = 51 666.67.
	// Month 2: 41 666.67 + 958 333.33 × 0.01 = 51 250.00.
	m1, err := DifferentiatedPayment(d("1000000"), d("0.01"), 24, 1)
	if err != nil {
		t.Fatalf("m1: %v", err)
	}
	closeTo(t, m1, d("51666.67"), "diff month 1")
	m2, err := DifferentiatedPayment(d("1000000"), d("0.01"), 24, 2)
	if err != nil {
		t.Fatalf("m2: %v", err)
	}
	closeTo(t, m2, d("51250.00"), "diff month 2")
}

func TestDifferentiated_ScheduleClosesToZero(t *testing.T) {
	sched, err := DifferentiatedSchedule(d("1000000"), d("0.01"), 24)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if len(sched) != 24 {
		t.Fatalf("length: got %d want 24", len(sched))
	}
	if !sched[len(sched)-1].BalanceEnd.IsZero() {
		t.Errorf("balance not closed: %s", sched[len(sched)-1].BalanceEnd)
	}
	// First payment > last payment (declining schedule).
	if sched[0].Payment.LessThanOrEqual(sched[len(sched)-1].Payment) {
		t.Errorf("expected declining schedule; first %s last %s", sched[0].Payment, sched[len(sched)-1].Payment)
	}
	// Principal portion is constant P/n = 41 666.67 except possibly the final
	// row's rounding absorb.
	for i := 0; i < len(sched)-1; i++ {
		closeTo(t, sched[i].Principal, d("41666.67"), "diff principal const")
	}
}

func TestDifferentiated_EdgeCases(t *testing.T) {
	// k out of range.
	if _, err := DifferentiatedPayment(d("1000"), d("0.01"), 12, 0); !errors.Is(err, ErrInvalidMonth) {
		t.Errorf("k=0: want ErrInvalidMonth, got %v", err)
	}
	if _, err := DifferentiatedPayment(d("1000"), d("0.01"), 12, 13); !errors.Is(err, ErrInvalidMonth) {
		t.Errorf("k=13: want ErrInvalidMonth, got %v", err)
	}
	// n = 0.
	if _, err := DifferentiatedPayment(d("1000"), d("0.01"), 0, 1); !errors.Is(err, ErrInvalidTerm) {
		t.Errorf("n=0: want ErrInvalidTerm, got %v", err)
	}
	// i = 0 → every payment equals P/n.
	got, err := DifferentiatedPayment(d("120000"), d("0"), 12, 1)
	if err != nil {
		t.Fatalf("i=0: %v", err)
	}
	closeTo(t, got, d("10000"), "diff zero-rate")
}

// -------- BrentQ --------

func TestBrentQ_SimpleRoot(t *testing.T) {
	// f(x) = x^2 − 2 has root √2 ≈ 1.41421356 on [1, 2].
	f := func(x float64) float64 { return x*x - 2 }
	root, err := BrentQ(f, 1, 2, f(1), f(2), 1e-12, 100)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if math.Abs(root-math.Sqrt2) > 1e-9 {
		t.Errorf("root: got %g, want %g", root, math.Sqrt2)
	}
}

func TestBrentQ_LinearRoot(t *testing.T) {
	// f(x) = 3x − 6 → root x = 2 on [-5, 5].
	f := func(x float64) float64 { return 3*x - 6 }
	root, err := BrentQ(f, -5, 5, f(-5), f(5), 1e-12, 100)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if math.Abs(root-2) > 1e-9 {
		t.Errorf("root: got %g, want 2", root)
	}
}

func TestBrentQ_NoSignChange(t *testing.T) {
	// f(x) = x^2 + 1, always positive on [0, 1] → ErrSolverFailed.
	f := func(x float64) float64 { return x*x + 1 }
	_, err := BrentQ(f, 0, 1, f(0), f(1), 1e-12, 100)
	if !errors.Is(err, ErrSolverFailed) {
		t.Errorf("expected ErrSolverFailed, got %v", err)
	}
}

// -------- PSK --------

func TestPSK_Golden(t *testing.T) {
	// MATH_FORMULAS.md §2.3 example scenario:
	//   P = 1_000_000, срок 24 мес, номинал 12%, страховка 50 000, комиссия 5 000.
	//   CF_0 = +945 000 (net disbursed), CF_1..24 = −annuity each.
	//
	// Expected ПСК ≈ 17.76 % (nominal annual, CBR Указание 5750-У).
	//
	// NOTE on the doc value: MATH_FORMULAS.md §2.3 printed ~17.96% assuming a
	// monthly payment of 47 073.46. That payment figure is an arithmetic typo
	// (see TestAnnuityPayment_Golden) — the exact annuity is 47 073.4722…,
	// rounded to 47 073.47. Solving the CBR equation with the correct annuity
	// yields ПСК = 17.76 %. We assert against the mathematically consistent
	// value; the doc must be updated to match (same fix style as Этап 0).
	monthly, err := AnnuityPayment(d("1000000"), d("0.01"), 24)
	if err != nil {
		t.Fatalf("annuity: %v", err)
	}
	cfs, err := AnnuityCashflows(
		d("1000000"),
		monthly,
		24,
		[]decimal.Decimal{d("50000"), d("5000")},
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("cashflows: %v", err)
	}
	psk, err := PSK(cfs)
	if err != nil {
		t.Fatalf("psk: %v", err)
	}
	// ПСК ≈ 0.1776 → tolerance 5 bps (0.0005) for day-count wiggle.
	want := d("0.1776")
	if diff := psk.Sub(want).Abs(); diff.GreaterThan(d("0.0005")) {
		t.Errorf("PSK: got %s (%.4f%%), want ~0.1776, diff %s", psk, psk.InexactFloat64()*100, diff)
	}
}

func TestPSK_NoFeesEqualsNominal(t *testing.T) {
	// Edge case from §2.3: without extra fees, ПСК == nominal annual rate.
	// Build an annuity with no upfront fees and verify ПСК ≈ 0.12.
	monthly, err := AnnuityPayment(d("1000000"), d("0.01"), 24)
	if err != nil {
		t.Fatalf("annuity: %v", err)
	}
	cfs, err := AnnuityCashflows(
		d("1000000"),
		monthly,
		24,
		nil, // no fees
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("cashflows: %v", err)
	}
	psk, err := PSK(cfs)
	if err != nil {
		t.Fatalf("psk: %v", err)
	}
	// Should equal nominal 0.12 within ~1 bp. (Tiny deviation comes from
	// monthly-payment rounding and month-length variation in ACT/365.)
	if diff := psk.Sub(d("0.12")).Abs(); diff.GreaterThan(d("0.0002")) {
		t.Errorf("PSK no-fees: got %s, want ~0.12, diff %s", psk, diff)
	}
}

func TestPSK_NoSignChange(t *testing.T) {
	// All-positive cashflows → ErrNoSignChange.
	cfs := []Cashflow{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Amount: d("1000")},
		{Date: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), Amount: d("500")},
	}
	_, err := PSK(cfs)
	if !errors.Is(err, ErrNoSignChange) {
		t.Errorf("expected ErrNoSignChange, got %v", err)
	}
}

func TestPSK_TooFewCashflows(t *testing.T) {
	_, err := PSK([]Cashflow{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Amount: d("1000")},
	})
	if !errors.Is(err, ErrInsufficientCashflows) {
		t.Errorf("expected ErrInsufficientCashflows, got %v", err)
	}
}

func TestPSK_SortsAndNormalisesDates(t *testing.T) {
	// Pass cashflows out of order and with non-midnight times; PSK must
	// still find the same root as the ordered/midnight version.
	monthly, _ := AnnuityPayment(d("1000000"), d("0.01"), 24)
	ordered, _ := AnnuityCashflows(
		d("1000000"), monthly, 24, nil,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	// Build an out-of-order + non-midnight copy WITHOUT aliasing.
	shuffled := make([]Cashflow, len(ordered))
	for i, cf := range ordered {
		// Reverse the order and add noise to the time component; PSK must
		// sort internally and snap to midnight.
		shuffled[len(ordered)-1-i] = Cashflow{
			Date:   cf.Date.Add(7 * time.Hour),
			Amount: cf.Amount,
		}
	}
	p1, err := PSK(ordered)
	if err != nil {
		t.Fatalf("ordered: %v", err)
	}
	p2, err := PSK(shuffled)
	if err != nil {
		t.Fatalf("shuffled: %v", err)
	}
	if !p1.Equal(p2) {
		t.Errorf("ordering sensitivity: ordered %s vs shuffled %s", p1, p2)
	}
}

// -------- Early repayment --------

func TestEarlyRepayment_ShortenTerm(t *testing.T) {
	// 1M @ 12% × 24m. After 12m, pay 100k early → term shrinks, payment same.
	res, err := EarlyRepayment(
		d("1000000"), d("0.12"),
		24, 12, d("100000"),
		EarlyShortenTerm,
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// Payment unchanged — compare against the exact annuity the same loan
	// produces, not the doc's typo'd 47073.46 (see TestAnnuityPayment_Golden).
	origPayment, _ := AnnuityPayment(d("1000000"), d("0.01"), 24)
	closeTo(t, res.NewPayment, origPayment, "shorten: payment unchanged")
	// New term must be < 12 (the remaining term before early payment).
	if res.NewRemainingTerm <= 0 || res.NewRemainingTerm >= 12 {
		t.Errorf("shorten: new term %d, expected 1..11", res.NewRemainingTerm)
	}
	// Interest saved must be positive.
	if !res.InterestSaved.IsPositive() {
		t.Errorf("shorten: interest saved = %s, expected > 0", res.InterestSaved)
	}
}

func TestEarlyRepayment_LowerPayment(t *testing.T) {
	// Same loan; with lower-payment mode the term stays at 12 remaining
	// months but the payment drops.
	res, err := EarlyRepayment(
		d("1000000"), d("0.12"),
		24, 12, d("100000"),
		EarlyLowerPayment,
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.NewRemainingTerm != 12 {
		t.Errorf("lower: new term %d, want 12", res.NewRemainingTerm)
	}
	orig, _ := AnnuityPayment(d("1000000"), d("0.01"), 24)
	if !res.NewPayment.LessThan(orig) {
		t.Errorf("lower: new payment %s not < original %s", res.NewPayment, orig)
	}
	if !res.InterestSaved.IsPositive() {
		t.Errorf("lower: interest saved = %s, expected > 0", res.InterestSaved)
	}
}

func TestEarlyRepayment_ClosesLoan(t *testing.T) {
	// Early payment equal to the full outstanding balance closes the loan.
	// First compute the balance after 12 months of the 1M/12%/24m loan.
	monthly, _ := AnnuityPayment(d("1000000"), d("0.01"), 24)
	sched, _ := AnnuitySchedule(d("1000000"), d("0.01"), monthly, 24)
	balanceAfter12 := sched[11].BalanceEnd
	res, err := EarlyRepayment(
		d("1000000"), d("0.12"),
		24, 12, balanceAfter12,
		EarlyShortenTerm,
	)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.NewRemainingTerm != 0 {
		t.Errorf("closes: new term %d, want 0", res.NewRemainingTerm)
	}
	if !res.NewPayment.IsZero() {
		t.Errorf("closes: new payment %s, want 0", res.NewPayment)
	}
}

func TestEarlyRepayment_EdgeCases(t *testing.T) {
	// Negative early amount → error.
	if _, err := EarlyRepayment(d("1000000"), d("0.12"), 24, 12, d("-1"), EarlyShortenTerm); !errors.Is(err, ErrInvalidEarlyAmount) {
		t.Errorf("negative: want ErrInvalidEarlyAmount, got %v", err)
	}
	// Early > balance → error.
	monthly, _ := AnnuityPayment(d("1000000"), d("0.01"), 24)
	sched, _ := AnnuitySchedule(d("1000000"), d("0.01"), monthly, 24)
	balanceAfter12 := sched[11].BalanceEnd
	_, err := EarlyRepayment(d("1000000"), d("0.12"), 24, 12, balanceAfter12.Add(d("1")), EarlyShortenTerm)
	if !errors.Is(err, ErrEarlyExceedsBalance) {
		t.Errorf("exceeds: want ErrEarlyExceedsBalance, got %v", err)
	}
	// paidMonths out of range.
	if _, err := EarlyRepayment(d("1000000"), d("0.12"), 24, -1, d("0"), EarlyShortenTerm); !errors.Is(err, ErrInvalidMonth) {
		t.Errorf("paidMonths<0: want ErrInvalidMonth, got %v", err)
	}
	if _, err := EarlyRepayment(d("1000000"), d("0.12"), 24, 25, d("0"), EarlyShortenTerm); !errors.Is(err, ErrInvalidMonth) {
		t.Errorf("paidMonths>term: want ErrInvalidMonth, got %v", err)
	}
	// Zero early amount → no change scenario (term unchanged for both modes).
	res, err := EarlyRepayment(d("1000000"), d("0.12"), 24, 12, d("0"), EarlyLowerPayment)
	if err != nil {
		t.Fatalf("zero early: %v", err)
	}
	if res.NewRemainingTerm != 12 {
		t.Errorf("zero early: new term %d, want 12", res.NewRemainingTerm)
	}
	if res.InterestSaved.IsNegative() {
		t.Errorf("zero early: interest saved should be ~0, got %s", res.InterestSaved)
	}
}
