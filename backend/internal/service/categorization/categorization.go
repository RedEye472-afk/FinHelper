package categorization

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// Repo is the storage contract the categorizer depends on. The concrete
// implementation is *storage.Pool; tests substitute a fake. Methods are scoped
// to the operations the matcher needs — it never writes operations directly.
type Repo interface {
	ListRules(ctx context.Context, userID int64) ([]storage.CategorizationRule, error)
	GetOverride(ctx context.Context, userID int64, counterparty string) (storage.CounterpartyOverride, error)

	// Category lookups for translating ids → names in results (and validating
	// that a rule's category still exists).
	GetCategory(ctx context.Context, userID, id int64) (storage.Category, error)
}

// Service is the categorization business layer. Construct once at boot, share
// across requests — it holds no per-request state.
type Service struct {
	repo Repo
}

// NewService returns a Service. repo must be non-nil.
func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

// Sentinel errors.
var (
	// ErrInvalidArgument — request failed validation.
	ErrInvalidArgument = errors.New("categorization: invalid argument")
	// ErrNotFound — referenced category/rule not found (or belongs to another user).
	ErrNotFound = errors.New("categorization: not found")
)

// Suggestion is the result of categorizing one operation. CategoryID is zero
// when no rule matched; the caller (UI) must then prompt the user.
type Suggestion struct {
	CategoryID int64
	Tier       MatchTier
	Confidence decimal.Decimal
	// RuleID is the keyword rule that matched, if any. Zero for override hits.
	RuleID int64
}

// MatchRequest is the input to Categorize. All fields are already PII-masked
// upstream (service/operations.Create) — the matcher never sees raw names.
type MatchRequest struct {
	Counterparty string
	Description  string
}

// Categorize picks a category for one operation following the precedence in
// the package doc. Returns Suggestion{Tier: TierNone} when nothing matched.
//
// It deliberately does NOT persist anything — persistence (updating the
// operation's category + confidence) is the caller's job, so the same matcher
// powers both the create path and the "re-categorize existing" path.
func (s *Service) Categorize(ctx context.Context, userID int64, req MatchRequest) (Suggestion, error) {
	if userID == 0 {
		return Suggestion{}, fmt.Errorf("%w: user_id required", ErrInvalidArgument)
	}

	// 1) Exact counterparty override (normalized). Learned ones are
	//    authoritative; tentative ones are a suggestion.
	cp := domain.NormalizeCounterparty(req.Counterparty)
	if cp != "" {
		o, err := s.repo.GetOverride(ctx, userID, cp)
		switch {
		case err == nil:
			tier := TierOverrideTentative
			if o.Confirmations >= domain.LearnThreshold {
				tier = TierOverrideLearned
			}
			return Suggestion{
				CategoryID: o.CategoryID,
				Tier:       tier,
				Confidence: mustParse(ConfidenceForTier(tier)),
			}, nil
		case !errors.Is(err, storage.ErrOverrideNotFound):
			return Suggestion{}, fmt.Errorf("categorization: get override: %w", err)
		}
		// not-found: fall through to keyword matching
	}

	// 2) Keyword rules. Precedence is encoded in priority (user rules seeded
	//    at priority 100, system at 0) and ListRules returns DESC.
	rules, err := s.repo.ListRules(ctx, userID)
	if err != nil {
		return Suggestion{}, fmt.Errorf("categorization: list rules: %w", err)
	}
	if hit, ruleID, ok := matchKeyword(rules, req); ok {
		return Suggestion{
			CategoryID: hit,
			Tier:       TierKeyword,
			Confidence: mustParse(ConfidenceKeyword),
			RuleID:     ruleID,
		}, nil
	}

	return Suggestion{Tier: TierNone}, nil
}

// CategorizeForCreate is the operations.Categorizer-compatible view of
// Categorize: it returns the suggested (categoryID, confidence) for a new
// operation, or (0, nil) when no rule matched. The confidence pointer is nil
// on no-match so the caller can distinguish "uncategorized" from "categorized
// at 0%" (which never happens in practice, but the type allows it).
//
// Returning a pointer to a stack value is safe because the caller consumes it
// synchronously before this frame returns.
func (s *Service) CategorizeForCreate(ctx context.Context, userID int64, counterparty, description string) (int64, *decimal.Decimal, error) {
	sug, err := s.Categorize(ctx, userID, MatchRequest{Counterparty: counterparty, Description: description})
	if err != nil {
		return 0, nil, err
	}
	if sug.Tier == TierNone {
		return 0, nil, nil
	}
	conf := sug.Confidence
	return sug.CategoryID, &conf, nil
}

// matchKeyword scans rules in priority order and returns the first whose
// keyword appears (case-insensitive substring) in counterparty or description.
// rules MUST be sorted priority DESC (ListRules guarantees this).
func matchKeyword(rules []storage.CategorizationRule, req MatchRequest) (categoryID, ruleID int64, ok bool) {
	cp := strings.ToLower(req.Counterparty)
	desc := strings.ToLower(req.Description)
	for _, r := range rules {
		kw := strings.ToLower(r.Keyword)
		if kw == "" {
			continue
		}
		if strings.Contains(cp, kw) || strings.Contains(desc, kw) {
			return r.CategoryID, r.ID, true
		}
	}
	return 0, 0, false
}

// LearnInput is supplied when the user confirms or corrects a category.
type LearnInput struct {
	Counterparty string // raw (pre-normalization) is fine; service normalizes
	CategoryID   int64
}

// Confirm records that the user agreed with a suggested category for a
// counterparty, bumping its confirmation counter. If the counterparty has no
// override yet, a new row is created with confirmations=1 (BUSINESS_LOGIC ф.2
// "после 3 подтверждений → создание персонального правила"). Returns the new
// confirmation count so the caller can tell the user "N/3 to learn".
//
// This is the write path; the categorizer reads through Categorize.
func (s *Service) Confirm(ctx context.Context, userID int64, in LearnInput, repo LearnRepo) (int, error) {
	if userID == 0 {
		return 0, fmt.Errorf("%w: user_id required", ErrInvalidArgument)
	}
	if in.CategoryID <= 0 {
		return 0, fmt.Errorf("%w: category_id required", ErrInvalidArgument)
	}
	cp := domain.NormalizeCounterparty(in.Counterparty)
	if cp == "" {
		return 0, fmt.Errorf("%w: counterparty required", ErrInvalidArgument)
	}
	// Validate category ownership before writing — a bogus category_id would
	// otherwise raise an FK violation downstream.
	if _, err := s.repo.GetCategory(ctx, userID, in.CategoryID); err != nil {
		return 0, mapStorageErr(err, "category")
	}
	o, err := repo.UpsertOverrideConfirmation(ctx, userID, cp, in.CategoryID)
	if err != nil {
		return 0, fmt.Errorf("categorization: confirm: %w", err)
	}
	return o.Confirmations, nil
}

// LearnRepo is the write-side contract for Confirm. Split from Repo so the
// read-only matcher (used in operations.Create hot path) doesn't need a write
// capability it never uses. Satisfied by *storage.Pool.
type LearnRepo interface {
	UpsertOverrideConfirmation(ctx context.Context, userID int64, counterparty string, categoryID int64) (storage.CounterpartyOverride, error)
}

func mapStorageErr(err error, what string) error {
	switch {
	case errors.Is(err, storage.ErrCategoryNotFound), errors.Is(err, storage.ErrRuleNotFound), errors.Is(err, storage.ErrOverrideNotFound):
		return fmt.Errorf("%w: %s", ErrNotFound, what)
	default:
		return fmt.Errorf("categorization: %s: %w", what, err)
	}
}

// mustParse converts a confidence string ("0.75") into a decimal.Decimal. The
// inputs are package constants, so a parse failure is a programmer error.
func mustParse(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(fmt.Sprintf("categorization: bad confidence constant %q: %v", s, err))
	}
	return d
}
