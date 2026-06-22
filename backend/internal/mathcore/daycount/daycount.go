// Package daycount implements day-count conventions used to convert a period
// between two dates into a year fraction.
//
// Conventions follow MATH_FORMULAS.md §4 (source: Люу К. "Методы финансовых
// расчётов", DayCount conventions). All math is integer/decimal — never float.
//
// Supported conventions:
//   - ACT/365   actual days, denominator fixed at 365 (Russian deposits default)
//   - 30/360    ISDA adjusted (corporate loans, bonds)
//   - ACT/ACT   ISMA — splits period across calendar years, using each year's
//     actual length (365 or 366) as denominator
package daycount

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Convention identifies a day-count convention.
type Convention string

const (
	ACT365   Convention = "ACT/365"
	Thirty360 Convention = "30/360" // ISDA
	ACTACT   Convention = "ACT/ACT" // ISMA
)

// Sentinel errors. Callers branch on errors.Is.
var (
	ErrReversedPeriod    = errors.New("daycount: start must be <= end")
	ErrUnknownConvention = errors.New("daycount: unknown convention")
)

// YearFraction returns the year-fraction between start and end (inclusive of
// start day, exclusive of end day, per market standard) under the given
// convention. Returns ErrReversedPeriod if start > end, ErrUnknownConvention
// if the convention is not recognised.
//
// Source: MATH_FORMULAS.md §4; Люу К. "Методы финансовых расчётов".
func YearFraction(start, end time.Time, convention Convention) (decimal.Decimal, error) {
	days, err := DaysBetween(start, end, convention)
	if err != nil {
		return decimal.Zero, err
	}
	switch convention {
	case ACT365:
		// Denominator fixed at 365 even for leap years — the ACT/365 convention.
		return decimal.NewFromInt(int64(days)).Div(decimal.NewFromInt(365)), nil
	case Thirty360:
		return decimal.NewFromInt(int64(days)).Div(decimal.NewFromInt(360)), nil
	case ACTACT:
		// ACT/ACT ISMA: split across calendar years, denominator per-year length.
		return actActYearFraction(start, end), nil
	default:
		return decimal.Zero, fmt.Errorf("%w: %q", ErrUnknownConvention, convention)
	}
}

// DaysBetween returns the day count between start and end under the convention.
// For 30/360 this is the *adjusted* count (D1/D2 collapsed to 30 on day 31),
// not the actual calendar days. For ACT/365 and ACT/ACT it is actual days.
//
// Edge cases:
//   - start == end → 0
//   - start > end  → ErrReversedPeriod
func DaysBetween(start, end time.Time, convention Convention) (int, error) {
	// Normalise to UTC midnight so day arithmetic is DST-independent. We only
	// use Y/M/D from this point on, so timezone choice does not change results
	// except by avoiding 23-hour DST days.
	start = utcMidnight(start)
	end = utcMidnight(end)
	if end.Before(start) {
		return 0, ErrReversedPeriod
	}
	switch convention {
	case ACT365, ACTACT:
		// Actual days: end - start in 24h units.
		return int(end.Sub(start).Hours() / 24), nil
	case Thirty360:
		return thirty360Days(start, end), nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrUnknownConvention, convention)
	}
}

// utcMidnight collapses t to YYYY-MM-DD 00:00 UTC. Day math then sees only
// calendar days, immune to DST shifts.
func utcMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// thirty360Days implements the 30/360 ISDA adjustment.
//
//	d1 = 31 → 30
//	d2 = 31 AND d1_adjusted >= 30 → 30
//	days = 360*(Y2-Y1) + 30*(M2-M1) + (D2-D1)
func thirty360Days(start, end time.Time) int {
	d1, m1, y1 := start.Day(), int(start.Month()), start.Year()
	d2, m2, y2 := end.Day(), int(end.Month()), end.Year()

	if d1 == 31 {
		d1 = 30
	}
	if d2 == 31 && d1 >= 30 {
		d2 = 30
	}
	return 360*(y2-y1) + 30*(m2-m1) + (d2 - d1)
}

// actActYearFraction splits [start, end) across calendar years and sums
// each sub-period divided by that year's length (365 or 366). This is the
// ISMA "real-world" ACT/ACT used by sovereign bonds (ОФЗ).
//
// Source: ICMA rule 251; MATH_FORMULAS.md §4.3.
func actActYearFraction(start, end time.Time) decimal.Decimal {
	start = utcMidnight(start)
	end = utcMidnight(end)
	if !end.After(start) {
		return decimal.Zero
	}

	var total decimal.Decimal
	cursor := start
	for cursor.Before(end) {
		yearEnd := time.Date(cursor.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
		segmentEnd := end
		if yearEnd.Before(segmentEnd) {
			segmentEnd = yearEnd
		}
		days := int(segmentEnd.Sub(cursor).Hours() / 24)
		yearLen := daysInYear(cursor.Year())
		total = total.Add(
			decimal.NewFromInt(int64(days)).Div(decimal.NewFromInt(int64(yearLen))),
		)
		cursor = segmentEnd
	}
	return total
}

// daysInYear returns 366 for leap years, 365 otherwise (Gregorian).
func daysInYear(y int) int {
	if (y%4 == 0 && y%100 != 0) || y%400 == 0 {
		return 366
	}
	return 365
}
