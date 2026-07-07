package tax

// Deposit interest tax per ФЗ-382 от 26.12.2023 (effective for income from
// 2024). The non-taxable threshold = 1_000_000 × key_rate_jan1, and the
// excess is taxed at DepositTaxRate (13%).
//
//	Налог = max(0, interest − 1_000_000 × key_rate_jan1) × deposit_tax_rate
//
// For the legacy 2021-2023 methodology (max key rate during the year), set
// Rules.Method = "max_rate" and pass the year's max rate via KeyRateJan1
// (the field is reused as the "max rate" under that method).
//
// Source: MATH_FORMULAS.md §5.4; НК РФ ст. 214.2.

import (
	"github.com/shopspring/decimal"
)

// DepositTax computes the personal income tax on bank-deposit interest for a
// given tax year.
//
// Edge cases:
//   - interest < 0 → ErrNegativeIncome
//   - interest ≤ threshold → 0
//   - method == "max_rate" → KeyRateJan1 is treated as max-rate-for-year
//
// Source: MATH_FORMULAS.md §5.4.
func DepositTax(rules Rules, interest decimal.Decimal) (decimal.Decimal, error) {
	if interest.IsNegative() {
		return decimal.Zero, ErrNegativeIncome
	}
	threshold := rules.NonTaxableThreshold.Value().Mul(rules.KeyRateJan1.Value())
	excess := interest.Sub(threshold)
	if !excess.IsPositive() {
		return decimal.Zero, nil
	}
	tax := excess.Mul(rules.DepositTaxRate.Value())
	// Bank-rounding: tax authorities round NDFL to whole rubles per
	// НК РФ ст. 225, but the deposit tax is computed on the ruble portion
	// of excess. We round to kopecks (scale 2) here for consistency with
	// the rest of mathcore; downstream reporting may round further.
	return tax.Round(2), nil
}

// Threshold returns the non-taxable threshold for the year (1M × key rate).
// Exposed for UI display ("необлагаемый порог в {year}: N ₽").
func Threshold(rules Rules) decimal.Decimal {
	return rules.NonTaxableThreshold.Value().Mul(rules.KeyRateJan1.Value()).Round(2)
}
