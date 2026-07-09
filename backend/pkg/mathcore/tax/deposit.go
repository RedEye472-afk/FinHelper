package tax

// Deposit interest tax per ФЗ-382 от 26.12.2023 (income from 2024+) and the
// progressive НДФЛ scale per ФЗ-257 от 12.12.2024 (income from 2025+).
//
// The non-taxable threshold = 1_000_000 × key_rate_jan1 (НК РФ ст. 214.2).
// The excess is taxed through the same progressive НДФЛ brackets as other
// personal income (13/15/18/20/22% from 2025; 13/15% in 2024). Computation
// delegates to NDFL with the threshold passed as a deduction.
//
//	Налог = NDFL(interest, deduction = 1_000_000 × key_rate_jan1)
//
// For the legacy 2021-2023 methodology (max key rate during the year), set
// Rules.Method = "max_rate" and pass the year's max rate via KeyRateJan1
// (the field is reused as the "max rate" under that method).
//
// Source: MATH_FORMULAS.md §5.4; НК РФ ст. 214.2, 210, 224; ФЗ-257.

import (
	"github.com/shopspring/decimal"
)

// DepositTax computes the personal income tax on bank-deposit interest for a
// given tax year using the progressive НДФЛ scale. The non-taxable threshold
// (1M × key_rate_jan1) is passed to NDFL as a deduction, so only the excess
// is taxed, and each slice of that excess is taxed at its marginal bracket
// rate per ФЗ-257.
//
// The flat DepositTaxRate field (13%) in Rules is no longer used by the
// computation — it remains in the config for backward compatibility and as
// the statutory base rate for documentation/display.
//
// Edge cases:
//   - interest < 0 → ErrNegativeIncome
//   - interest ≤ threshold → 0
//   - no НДФЛ brackets configured → ErrUnknownNDFLScale (propagated from NDFL)
//   - method == "max_rate" → KeyRateJan1 is treated as max-rate-for-year
//
// Source: MATH_FORMULAS.md §5.4; НК РФ ст. 214.2, 210, 224; ФЗ-257.
func DepositTax(rules Rules, interest decimal.Decimal) (decimal.Decimal, error) {
	if interest.IsNegative() {
		return decimal.Zero, ErrNegativeIncome
	}
	// The threshold acts as the deduction: NDFL floors the taxable base at
	// max(0, income − deduction) and then applies the progressive brackets.
	threshold := Threshold(rules)
	return NDFL(rules, interest, threshold)
}

// Threshold returns the non-taxable threshold for the year (1M × key rate).
// Exposed for UI display ("необлагаемый порог в {year}: N ₽").
func Threshold(rules Rules) decimal.Decimal {
	return rules.NonTaxableThreshold.Value().Mul(rules.KeyRateJan1.Value()).Round(2)
}
