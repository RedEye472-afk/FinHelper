// Package credit (test): credit_test.go unit-tests the stateless credit
// calculator service against the golden values already pinned in
// internal/mathcore/credit (annuity payment 47 073.47, ПСК ≈ 17.76 %, etc.).
// Tests run without a database — the service has no storage dependency.
package credit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal {
	v, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return v
}

func mustsvc() *Service {
	s := NewService()
	s.now = func() time.Time { return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) }
	return s
}

// TestCalculate_Annuity_Golden mirrors the mathcore golden: P=1M, 12%, 24m.
// Annuity payment must equal 47073.47 (rounded). ПСК no-fees ≈ 0.12.
func TestCalculate_Annuity_Golden(t *testing.T) {
	s := mustsvc()
	res, err := s.Calculate(context.Background(), Input{
		Principal:   d("1000000"),
		AnnualRate:  d("0.12"),
		TermMonths:  24,
		PaymentType: PaymentAnnuity,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	want := d("47073.47")
	if !res.MonthlyPayment.Equal(want) {
		t.Errorf("monthly_payment = %s, want %s", res.MonthlyPayment, want)
	}
	if len(res.Schedule) != 24 {
		t.Fatalf("schedule len = %d, want 24", len(res.Schedule))
	}
	// Balance closes to zero on the last row.
	last := res.Schedule[len(res.Schedule)-1]
	if !last.BalanceEnd.IsZero() {
		t.Errorf("last balance = %s, want 0", last.BalanceEnd)
	}
	// ПСК without fees ≈ 0.12 (nominal).
	if diff := res.PSK.Sub(d("0.12")).Abs(); diff.GreaterThan(d("0.0005")) {
		t.Errorf("PSK = %s, want ~0.12 (diff %s)", res.PSK, diff)
	}
	// Overpayment is positive.
	if !res.Overpayment.IsPositive() {
		t.Errorf("overpayment = %s, must be positive", res.Overpayment)
	}
}

// TestCalculate_Differentiated: declining payments; first > last; sum of
// principal parts equals principal.
func TestCalculate_Differentiated(t *testing.T) {
	s := mustsvc()
	res, err := s.Calculate(context.Background(), Input{
		Principal:   d("1000000"),
		AnnualRate:  d("0.12"),
		TermMonths:  12,
		PaymentType: PaymentDifferentiated,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if len(res.Schedule) != 12 {
		t.Fatalf("schedule len = %d, want 12", len(res.Schedule))
	}
	if !res.Schedule[0].Payment.GreaterThan(res.Schedule[11].Payment) {
		t.Errorf("differentiated should decline: first=%s last=%s",
			res.Schedule[0].Payment, res.Schedule[11].Payment)
	}
	sumPrincipal := decimal.Zero
	for _, r := range res.Schedule {
		sumPrincipal = sumPrincipal.Add(r.Principal)
	}
	if !sumPrincipal.Equal(d("1000000")) {
		t.Errorf("Σ principal = %s, want 1000000", sumPrincipal)
	}
}

// TestCalculate_ZeroRate: interest-free loan → payment = P/n.
func TestCalculate_ZeroRate(t *testing.T) {
	s := mustsvc()
	res, err := s.Calculate(context.Background(), Input{
		Principal:   d("120000"),
		AnnualRate:  d("0"),
		TermMonths:  12,
		PaymentType: PaymentAnnuity,
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	want := d("10000.00")
	if !res.MonthlyPayment.Equal(want) {
		t.Errorf("zero-rate payment = %s, want %s", res.MonthlyPayment, want)
	}
	// Overpayment should be zero (no interest, no fees).
	if !res.Overpayment.IsZero() {
		t.Errorf("zero-rate overpayment = %s, want 0", res.Overpayment)
	}
}

// TestCalculate_FeesPushPSKAboveRate: a 50k upfront fee + 1k monthly fee
// must push ПСК strictly above the nominal 12 %.
func TestCalculate_FeesPushPSKAboveRate(t *testing.T) {
	s := mustsvc()
	res, err := s.Calculate(context.Background(), Input{
		Principal:    d("1000000"),
		AnnualRate:   d("0.12"),
		TermMonths:   24,
		PaymentType:  PaymentAnnuity,
		UpfrontFees:  []decimal.Decimal{d("50000")},
		MonthlyFee:   d("1000"),
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if !res.PSK.GreaterThan(d("0.12")) {
		t.Errorf("PSK = %s, must be > 0.12 (fees raise PSK per spec)", res.PSK)
	}
	// Overpayment includes the fees.
	wantInterest := res.Overpayment
	// Check overpayment > fees + base interest by comparing to feeless case.
	res2, _ := s.Calculate(context.Background(), Input{
		Principal: d("1000000"), AnnualRate: d("0.12"), TermMonths: 24, PaymentType: PaymentAnnuity,
	})
	if !wantInterest.GreaterThan(res2.Overpayment) {
		t.Errorf("with fees overpayment %s must exceed no-fee %s", wantInterest, res2.Overpayment)
	}
}

// TestCalculate_EarlyShortenTerm: early lump sum → interest saved positive,
// new remaining term shorter than original (term − paidMonths), summary present.
func TestCalculate_EarlyShortenTerm(t *testing.T) {
	s := mustsvc()
	res, err := s.Calculate(context.Background(), Input{
		Principal:   d("1000000"),
		AnnualRate:  d("0.12"),
		TermMonths:  24,
		PaymentType: PaymentAnnuity,
		Early: &EarlyInput{
			PaidMonths: 12,
			Amount:     d("200000"),
			Mode:       EarlyShortenTerm,
		},
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if res.Early == nil {
		t.Fatal("early scenario missing")
	}
	if !res.Early.InterestSaved.IsPositive() {
		t.Errorf("interest_saved = %s, must be > 0", res.Early.InterestSaved)
	}
	orig := 24 - 12 // remaining after 12 paid
	if res.Early.NewRemainingTerm <= 0 || res.Early.NewRemainingTerm >= orig {
		t.Errorf("new_remaining_term = %d, want in (0, %d)", res.Early.NewRemainingTerm, orig)
	}
	if res.Early.Summary == "" {
		t.Error("summary must not be empty")
	}
}

// TestCalculate_EarlyLowerPayment: with lower_payment, term stays equal,
// new payment strictly less than the original.
func TestCalculate_EarlyLowerPayment(t *testing.T) {
	s := mustsvc()
	res, err := s.Calculate(context.Background(), Input{
		Principal:   d("1000000"),
		AnnualRate:  d("0.12"),
		TermMonths:  24,
		PaymentType: PaymentAnnuity,
		Early: &EarlyInput{
			PaidMonths: 12,
			Amount:     d("200000"),
			Mode:       EarlyLowerPayment,
		},
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	orig := 24 - 12
	if res.Early.NewRemainingTerm != orig {
		t.Errorf("new_remaining_term = %d, want %d (lower_payment keeps term)", res.Early.NewRemainingTerm, orig)
	}
	if !res.Early.NewPayment.LessThan(res.MonthlyPayment) {
		t.Errorf("new_payment %s must be < original %s", res.Early.NewPayment, res.MonthlyPayment)
	}
}

// TestCalculate_EarlyZeroMeansNoScenario: an early sub-object with amount=0
// is treated as "no early scenario requested" (BUSINESS_LOGIC edge case).
func TestCalculate_EarlyZeroMeansNoScenario(t *testing.T) {
	s := mustsvc()
	res, err := s.Calculate(context.Background(), Input{
		Principal:   d("1000000"),
		AnnualRate:  d("0.12"),
		TermMonths:  24,
		PaymentType: PaymentAnnuity,
		Early:       &EarlyInput{PaidMonths: 6, Amount: d("0")},
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if res.Early != nil {
		t.Errorf("expected no early scenario, got %+v", res.Early)
	}
}

// TestCalculate_Validation errors.
func TestCalculate_Validation(t *testing.T) {
	s := mustsvc()
	cases := []struct {
		name string
		in   Input
	}{
		{"zero principal", Input{Principal: d("0"), AnnualRate: d("0.12"), TermMonths: 12, PaymentType: PaymentAnnuity}},
		{"negative principal", Input{Principal: d("-1"), AnnualRate: d("0.12"), TermMonths: 12, PaymentType: PaymentAnnuity}},
		{"zero term", Input{Principal: d("100"), AnnualRate: d("0.12"), TermMonths: 0, PaymentType: PaymentAnnuity}},
		{"negative rate", Input{Principal: d("100"), AnnualRate: d("-0.01"), TermMonths: 12, PaymentType: PaymentAnnuity}},
		{"bad payment_type", Input{Principal: d("100"), AnnualRate: d("0.12"), TermMonths: 12, PaymentType: PaymentType("bullet")}},
		{"bad early.mode", Input{Principal: d("100"), AnnualRate: d("0.12"), TermMonths: 12, PaymentType: PaymentAnnuity,
			Early: &EarlyInput{PaidMonths: 6, Amount: d("10"), Mode: EarlyMode("tenfold")}}},
		{"early.paidMonths > term", Input{Principal: d("100"), AnnualRate: d("0.12"), TermMonths: 12, PaymentType: PaymentAnnuity,
			Early: &EarlyInput{PaidMonths: 99, Amount: d("10")}}},
		{"early.amount negative", Input{Principal: d("100"), AnnualRate: d("0.12"), TermMonths: 12, PaymentType: PaymentAnnuity,
			Early: &EarlyInput{PaidMonths: 6, Amount: d("-5")}}},
		{"monthly_fee negative", Input{Principal: d("100"), AnnualRate: d("0.12"), TermMonths: 12, PaymentType: PaymentAnnuity,
			MonthlyFee: d("-1")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := s.Calculate(context.Background(), c.in)
			if err == nil {
				t.Fatalf("expected ErrInvalidArgument, got nil")
			}
			if !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("expected ErrInvalidArgument, got %v", err)
			}
		})
	}
}

// TestCalculate_DefaultPaymentTypeIsAnnuity: an empty PaymentType defaults to
// annuity, not error. The HTTP layer sends "" when the client omits the field.
func TestCalculate_DefaultPaymentTypeIsAnnuity(t *testing.T) {
	s := mustsvc()
	res, err := s.Calculate(context.Background(), Input{
		Principal:   d("1000000"),
		AnnualRate:  d("0.12"),
		TermMonths:  24,
		PaymentType: "",
	})
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	if res.PaymentType != PaymentAnnuity {
		t.Errorf("default PaymentType = %q, want annuity", res.PaymentType)
	}
}