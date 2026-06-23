package tax

// НПД (Налог на профессиональный доход) — ФЗ-422 от 27.11.2018.
// 4% on income from individuals, 6% on income from ИП/юрлиц.
// Annual limit 2.4M ₽ — above it the taxpayer must switch regimes.
//
// Source: MATH_FORMULAS.md §5.1.

import (
	"github.com/shopspring/decimal"
)

// NPDResult holds the breakdown of an НПД computation.
type NPDResult struct {
	Tax         decimal.Decimal // total tax owed
	ExceedsLimit bool            // true if income > AnnualLimit (regime switch required)
}

// NPD computes the self-employed tax for a single year, given the income
// split by counterparty type.
//
// Edge cases:
//   - income < 0 → ErrNegativeIncome
//   - income = 0 → tax = 0
//
// Source: MATH_FORMULAS.md §5.1; ФЗ-422 ст. 10.
func NPD(rules Rules, incomeIndividuals, incomeBusiness decimal.Decimal) (NPDResult, error) {
	if incomeIndividuals.IsNegative() || incomeBusiness.IsNegative() {
		return NPDResult{}, ErrNegativeIncome
	}
	taxIndiv := incomeIndividuals.Mul(rules.NPD.RateIndividuals.Value())
	taxBiz := incomeBusiness.Mul(rules.NPD.RateBusiness.Value())
	total := incomeIndividuals.Add(incomeBusiness)
	res := NPDResult{
		Tax: taxIndiv.Add(taxBiz).Round(2),
	}
	res.ExceedsLimit = total.GreaterThan(rules.NPD.AnnualLimit.Value())
	return res, nil
}

// NPDRateFor returns the НПД rate for a given counterparty type.
func NPDRateFor(rules Rules, c NPDCounterparty) (decimal.Decimal, error) {
	switch c {
	case NPDIndividuals:
		return rules.NPD.RateIndividuals.Value(), nil
	case NPDBusiness:
		return rules.NPD.RateBusiness.Value(), nil
	default:
		return decimal.Zero, ErrUnknownNPDCounterparty
	}
}
