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

// NDFL rules (НК РФ Гл. 23). Progressive scale is encoded as ordered Brackets
// (preferred). Legacy 2-step configs without Brackets fall back to
// BaseRate/HighRate/HighIncomeThreshold via NDFLBrackets().
type NDFLRules struct {
	BaseRate             Dec       `yaml:"base_rate"`
	HighRate             Dec       `yaml:"high_rate,omitempty"`
	HighIncomeThreshold  Dec       `yaml:"high_income_threshold,omitempty"`
	// Brackets — progressive-scale brackets, ordered by ascending up_to. Each
	// bracket applies its rate to the slice of taxable income in
	// (previous up_to, this up_to]. up_to == 0 means "to infinity" (the top
	// bracket). Empty → NDFLBrackets() synthesises a 2-step scale from the
	// legacy fields. Active for 2025+ (ФЗ-257 от 12.12.2024).
	Brackets             []Bracket `yaml:"brackets,omitempty"`
	ChildDeduction1      Dec       `yaml:"child_deduction_1"`
	ChildDeduction3      Dec       `yaml:"child_deduction_3"`
	PropertyDeductionMax Dec       `yaml:"property_deduction_max"`
	IISDeductionMax      Dec       `yaml:"iis_deduction_max"`
	Source               string    `yaml:"source"`
}

// Bracket is one step of the progressive НДФЛ scale. UpTo == 0 denotes the
// uncapped top bracket. Rate is the marginal rate applied to the slice of
// income falling inside this bracket.
type Bracket struct {
	UpTo  Dec `yaml:"up_to"`
	Rate  Dec `yaml:"rate"`
}

// NDFLBrackets returns the effective progressive brackets: explicit Brackets
// if configured, otherwise a 2-step scale synthesised from the legacy
// HighIncomeThreshold/HighRate fields (with BaseRate below the threshold).
// This keeps NDFL() working for pre-2025 configs that predate the brackets
// field and lets callers inspect the active scale for UI display.
func (n NDFLRules) NDFLBrackets() []Bracket {
	if len(n.Brackets) > 0 {
		return n.Brackets
	}
	return []Bracket{
		{UpTo: n.HighIncomeThreshold, Rate: n.BaseRate},
		{UpTo: Dec{}, Rate: n.HighRate},
	}
}
