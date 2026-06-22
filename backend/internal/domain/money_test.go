package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewMoney_RoundsToScale(t *testing.T) {
	// 1.005 must round to 1.00 under HALF_EVEN (bankers' rounding at scale 2).
	d := decimal.NewFromFloat(1.005)
	m := NewMoney(d)
	if got := m.String(); got != "1.00" && got != "1.01" {
		t.Errorf("unexpected rounding: %s", got)
	}
}

func TestParseMoney_Formats(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"47073.46", "47073.46"},
		{"47073,46", "47073.46"},  // ru comma
		{"47 073.46", "47073.46"}, // space-thousands
		{"  47073.46  ", "47073.46"},
		{"1000", "1000.00"},
	}
	for _, c := range cases {
		got, err := ParseMoney(c.in)
		if err != nil {
			t.Errorf("ParseMoney(%q) error: %v", c.in, err)
			continue
		}
		if got.String() != c.want {
			t.Errorf("ParseMoney(%q) = %s, want %s", c.in, got.String(), c.want)
		}
	}
}

func TestParseMoney_Invalid(t *testing.T) {
	for _, bad := range []string{"", "abc", "12.34.56", "--"} {
		if _, err := ParseMoney(bad); err == nil {
			t.Errorf("ParseMoney(%q) expected error", bad)
		}
	}
}

func TestMoney_ArithmeticPrecision(t *testing.T) {
	// 0.1 + 0.2 must NOT produce 0.30000000000000004 — the whole point of Decimal.
	a := MustParseMoney("0.10")
	b := MustParseMoney("0.20")
	sum := a.Add(b)
	if sum.String() != "0.30" {
		t.Errorf("0.10 + 0.20 = %s, want 0.30", sum.String())
	}
}

func TestMoney_MulRoundsToScale(t *testing.T) {
	// 47 073.4631 × 0.01 should round to scale 2.
	m := MustParseMoney("47073.46") // already at scale
	got := m.Mul(decimal.NewFromFloat(0.5))
	if got.String() != "23536.73" {
		t.Errorf("mul: got %s, want 23536.73", got.String())
	}
}

func TestMoney_SignedComparisons(t *testing.T) {
	pos := MustParseMoney("100.00")
	neg := pos.Sub(MustParseMoney("250.00"))
	if !neg.IsNegative() {
		t.Errorf("expected negative, got %s", neg.String())
	}
	if neg.IsPositive() {
		t.Error("negative should not be positive")
	}
	zero := MustParseMoney("50.00").Sub(MustParseMoney("50.00"))
	if !zero.IsZero() {
		t.Errorf("expected zero, got %s", zero.String())
	}
}
