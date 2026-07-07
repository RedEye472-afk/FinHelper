package daycount

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// date parses YYYY-MM-DD as UTC midnight — test fixture helper.
func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic("bad date fixture: " + s)
	}
	return t.UTC()
}

func TestDaysBetween_Actual(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want int
	}{
		{"ACT365_first_half_2024", "2024-01-01", "2024-07-01", 182},
		{"ACT365_same_day", "2024-03-15", "2024-03-15", 0},
		{"ACT365_one_day", "2024-03-15", "2024-03-16", 1},
		{"ACT365_across_leap_day", "2024-02-28", "2024-03-01", 2},
		{"ACT365_across_year", "2023-12-30", "2024-01-02", 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := DaysBetween(date(c.a), date(c.b), ACT365)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestDaysBetween_Thirty360(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want int
	}{
		// 31→30 adjustment on both sides.
		{"both_31", "2024-01-31", "2024-03-31", 60},
		// d1=31→30, d2=15 (not adjusted since d1<30 after adj → 30, wait d1=30>=30,
		// but original d2=15 != 31 so no adj). days = 30*(3-1) + (15-30) = 45.
		{"start_31_end_15", "2024-01-31", "2024-03-15", 45},
		// d1=15, d2=31: d1<30 so d2 NOT adjusted → days = 60 + 0 + 16 = 76.
		{"start_15_end_31", "2024-01-15", "2024-03-31", 76},
		// Whole-year-ish: 2024-01-01..2024-12-31 = 360.
		{"full_year", "2024-01-01", "2024-12-31", 360},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := DaysBetween(date(c.a), date(c.b), Thirty360)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestYearFraction_Golden(t *testing.T) {
	tol := decimal.NewFromFloat(1e-10)
	cases := []struct {
		name        string
		a, b        string
		conv        Convention
		expected    decimal.Decimal
		expectedStr string // for documentation
	}{
		{
			name:        "ACT365_2024_H1",
			a:           "2024-01-01",
			b:           "2024-07-01",
			conv:        ACT365,
			expected:    decimal.NewFromInt(182).Div(decimal.NewFromInt(365)),
			expectedStr: "182/365 ≈ 0.49863",
		},
		{
			name:        "Thirty360_jan31_mar31",
			a:           "2024-01-31",
			b:           "2024-03-31",
			conv:        Thirty360,
			expected:    decimal.NewFromInt(60).Div(decimal.NewFromInt(360)),
			expectedStr: "60/360 ≈ 0.16667",
		},
		{
			name:        "ACTACT_2024_leap_H1",
			a:           "2024-01-01",
			b:           "2024-07-01",
			conv:        ACTACT,
			expected:    decimal.NewFromInt(182).Div(decimal.NewFromInt(366)),
			expectedStr: "182/366 ≈ 0.49727",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := YearFraction(date(c.a), date(c.b), c.conv)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			diff := got.Sub(c.expected).Abs()
			if diff.GreaterThan(tol) {
				t.Errorf("%s: got %s, want %s (%s), diff %s",
					c.name, got.String(), c.expected.String(), c.expectedStr, diff.String())
			}
		})
	}
}

// ACT/ACT spanning a non-leap → leap year boundary.
// 2023-10-15..2024-03-20:
//   2023 segment: 2023-10-15..2024-01-01 = 78 days / 365
//   2024 segment: 2024-01-01..2024-03-20 = 79 days / 366
func TestYearFraction_ACTACT_SpansBoundary(t *testing.T) {
	tol := decimal.NewFromFloat(1e-10)
	got, err := YearFraction(date("2023-10-15"), date("2024-03-20"), ACTACT)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	expected := decimal.NewFromInt(78).Div(decimal.NewFromInt(365)).
		Add(decimal.NewFromInt(79).Div(decimal.NewFromInt(366)))
	diff := got.Sub(expected).Abs()
	if diff.GreaterThan(tol) {
		t.Errorf("got %s, want %s, diff %s", got.String(), expected.String(), diff.String())
	}
}

func TestYearFraction_SameDay_IsZero(t *testing.T) {
	for _, conv := range []Convention{ACT365, Thirty360, ACTACT} {
		got, err := YearFraction(date("2024-06-15"), date("2024-06-15"), conv)
		if err != nil {
			t.Errorf("%s: err %v", conv, err)
			continue
		}
		if !got.IsZero() {
			t.Errorf("%s: same-day fraction should be 0, got %s", conv, got)
		}
	}
}

func TestReversedPeriod(t *testing.T) {
	for _, conv := range []Convention{ACT365, Thirty360, ACTACT} {
		_, err := YearFraction(date("2024-06-16"), date("2024-06-15"), conv)
		if !errors.Is(err, ErrReversedPeriod) {
			t.Errorf("%s: expected ErrReversedPeriod, got %v", conv, err)
		}
	}
}

func TestUnknownConvention(t *testing.T) {
	_, err := YearFraction(date("2024-01-01"), date("2024-02-01"), Convention("ACT/ACT-ISDA"))
	if !errors.Is(err, ErrUnknownConvention) {
		t.Errorf("expected ErrUnknownConvention, got %v", err)
	}
}
