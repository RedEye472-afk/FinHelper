package domain

import (
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

// RuleSource marks who created a categorization rule.
type RuleSource string

const (
	// RuleSystem — seeded defaults at registration.
	RuleSystem RuleSource = "system"
	// RuleUser — created or edited by the user.
	RuleUser RuleSource = "user"
)

// RolloverPolicy controls how a budget's unused balance carries into the next
// period (BUSINESS_LOGIC.md ф.4). Mirrors budgets.rollover_policy CHECK.
type RolloverPolicy string

const (
	RolloverNone      RolloverPolicy = "none"      // unused amount expires
	RolloverUnlimited RolloverPolicy = "unlimited" // accumulate indefinitely
	RolloverMonths3   RolloverPolicy = "months_3"  // cap at 3 months
)

// LearnThreshold is the number of confirmations required to promote a
// counterparty override to an authoritative learned rule (BUSINESS_LOGIC ф.2:
// "после 3 подтверждений → создание персонального правила"). Pure counter,
// not ML — scope-lock (Plane from GLM.md) keeps this deterministic.
const LearnThreshold = 3

// MinCategorizationConfidence is the floor below which the UI should ask the
// user to confirm a suggested category (BUSINESS_LOGIC ф.2: "при <70% → запрос
// подтверждения"). Keyword matches land above this; a "no match" fallback sits
// below it so the UI prompts for confirmation.
var MinCategorizationConfidence = decimal.NewFromFloat(0.70)

// NormalizeCounterparty lowercases + trims a counterparty string for matching
// and storage. Applied both at write (counterparty_overrides) and at lookup so
// the two sides always agree. The value is already PII-masked upstream
// (service/operations.Create), so normalization never touches raw names.
func NormalizeCounterparty(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// ValidateRolloverPolicy returns an error unless p is one of the allowed values.
func ValidateRolloverPolicy(p RolloverPolicy) error {
	switch p {
	case RolloverNone, RolloverUnlimited, RolloverMonths3:
		return nil
	default:
		return errors.New("domain: invalid rollover_policy")
	}
}
