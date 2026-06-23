// Package tax implements Russian tax calculations for ФинПомощник: НПД
// (self-employed), УСН (small business), НДФЛ (personal income), and the
// deposit interest tax.
//
// Design principle (CLAUDE.md §1 "Детерминизм"): all monetary math uses
// decimal.Decimal — never float64. No solver is needed here, so (unlike
// credit/PSK and investment/XIRR) there is NO float bridge in this package.
//
// Rules are versioned per calendar year in configs/tax_rules_{year}.yaml.
// Loading and parsing happens via Rules / LoadRules; the computation
// functions take an explicit Rules value so they are pure and testable.
//
// Sources:
//   - НК РФ Гл. 23 (НДФЛ), Гл. 26.2 (УСН)
//   - ФЗ-422 от 27.11.2018 (НПД)
//   - ФЗ-382 от 26.12.2023 (deposit tax methodology, effective 2024+)
//   - MATH_FORMULAS.md §5
package tax

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// Sentinel errors. Callers branch on errors.Is.
var (
	// ErrNegativeIncome — income base must be >= 0.
	ErrNegativeIncome = errors.New("tax: income must be >= 0")
	// ErrNegativeExpenses — expenses must be >= 0.
	ErrNegativeExpenses = errors.New("tax: expenses must be >= 0")
	// ErrNegativeDeductions — deductions must be >= 0.
	ErrNegativeDeductions = errors.New("tax: deductions must be >= 0")
	// ErrUnsupportedYear — no tax_rules config loaded for the requested year.
	ErrUnsupportedYear = errors.New("tax: no rules loaded for this year")
	// ErrUnknownNPDCounterparty — NPD rate depends on counterparty type.
	ErrUnknownNPDCounterparty = errors.New("tax: unknown NPD counterparty type")
	// ErrUnknownUSNRegime — УСН regime not recognised.
	ErrUnknownUSNRegime = errors.New("tax: unknown УСН regime")
)

// NPDCounterparty selects the НПД rate.
type NPDCounterparty int

const (
	// NPDIndividuals — 4% rate (работа с физлицами).
	NPDIndividuals NPDCounterparty = iota
	// NPDBusiness — 6% rate (работа с ИП/юрлицами).
	NPDBusiness
)

// USNRegime selects the УСН computation method.
type USNRegime int

const (
	// USNIncome — 6% of revenue.
	USNIncome USNRegime = iota
	// USNIncomeMinusExpenses — 15% of (revenue − expenses).
	USNIncomeMinusExpenses
)

// Dec is a YAML-friendly decimal: parses string values to avoid float
// precision loss in the config files. All money/rate fields in Rules use
// Dec; callers obtain the underlying decimal.Decimal via Dec.Value().
type Dec struct {
	V decimal.Decimal
}

// UnmarshalYAML implements yaml.Unmarshaler. Accepts both quoted strings
// ("0.13") and plain scalars; strings are mandatory for money/rate fields
// to preserve precision.
func (x *Dec) UnmarshalYAML(un func(interface{}) error) error {
	var s string
	if err := un(&s); err != nil {
		return err
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return fmt.Errorf("tax: invalid decimal %q: %w", s, err)
	}
	x.V = d
	return nil
}

// Value returns the underlying decimal.Decimal.
func (x Dec) Value() decimal.Decimal { return x.V }

// Rules is the parsed content of configs/tax_rules_{year}.yaml. All money
// and rate fields use Dec so the YAML loader parses them from string and
// preserves precision. Use Rules' accessor methods in computations.
type Rules struct {
	Year                int    `yaml:"year"`
	Method              string `yaml:"method"` // "jan1_rate" (2024+) or "max_rate" (legacy 2021-2023)
	KeyRateJan1         Dec    `yaml:"key_rate_jan1"`
	NonTaxableThreshold Dec    `yaml:"non_taxable_threshold"`
	DepositTaxRate      Dec    `yaml:"deposit_tax_rate"`
	NPD                 NPDRules `yaml:"npd"`
	USN                 USNRules `yaml:"usn"`
	NDFL                NDFLRules `yaml:"ndfl"`
}

// NPD rules (ФЗ-422).
type NPDRules struct {
	RateIndividuals Dec    `yaml:"rate_individuals"`
	RateBusiness    Dec    `yaml:"rate_business"`
	AnnualLimit     Dec    `yaml:"annual_limit"`
	Source          string `yaml:"source"`
}

// USN rules (НК РФ Гл. 26.2).
type USNRules struct {
	RateIncome              Dec    `yaml:"rate_income"`
	RateIncomeMinusExpenses Dec    `yaml:"rate_income_minus_expenses"`
	Source                  string `yaml:"source"`
}

// NDFL rules (НК РФ Гл. 23).
type NDFLRules struct {
	BaseRate             Dec    `yaml:"base_rate"`
	HighRate             Dec    `yaml:"high_rate"`
	HighIncomeThreshold  Dec    `yaml:"high_income_threshold"`
	ChildDeduction1      Dec    `yaml:"child_deduction_1"`
	ChildDeduction3      Dec    `yaml:"child_deduction_3"`
	PropertyDeductionMax Dec    `yaml:"property_deduction_max"`
	IISDeductionMax      Dec    `yaml:"iis_deduction_max"`
	Source               string `yaml:"source"`
}
