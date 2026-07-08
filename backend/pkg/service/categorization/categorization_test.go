package categorization

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
)

// fakeRepo is an in-memory categorization.Repo for service tests.
type fakeRepo struct {
	rules     []storage.CategorizationRule
	overrides map[string]storage.CounterpartyOverride // key = normalized counterparty
	cats      map[int64]storage.Category
	// upsertRepo captures Confirm writes; nil = Confirm will fail to write
	upsert *fakeUpsertRepo
	// failWith, if set, makes read methods return it
	failWith error
}

type fakeUpsertRepo struct {
	lastCall struct {
		userID      int64
		counterparty string
		categoryID  int64
	}
	returnConfirmations int
	failWith            error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		overrides: make(map[string]storage.CounterpartyOverride),
		cats:      make(map[int64]storage.Category),
		upsert:    &fakeUpsertRepo{returnConfirmations: 1},
	}
}

func (f *fakeRepo) ListRules(_ context.Context, userID int64) ([]storage.CategorizationRule, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	out := make([]storage.CategorizationRule, 0, len(f.rules))
	for _, r := range f.rules {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetOverride(_ context.Context, userID int64, cp string) (storage.CounterpartyOverride, error) {
	if f.failWith != nil {
		return storage.CounterpartyOverride{}, f.failWith
	}
	o, ok := f.overrides[cp]
	if !ok || o.UserID != userID {
		return storage.CounterpartyOverride{}, storage.ErrOverrideNotFound
	}
	return o, nil
}

func (f *fakeRepo) GetCategory(_ context.Context, userID, id int64) (storage.Category, error) {
	c, ok := f.cats[id]
	if !ok || c.UserID != userID {
		return storage.Category{}, storage.ErrCategoryNotFound
	}
	return c, nil
}

func (u *fakeUpsertRepo) UpsertOverrideConfirmation(_ context.Context, userID int64, cp string, catID int64) (storage.CounterpartyOverride, error) {
	if u.failWith != nil {
		return storage.CounterpartyOverride{}, u.failWith
	}
	u.lastCall.userID = userID
	u.lastCall.counterparty = cp
	u.lastCall.categoryID = catID
	return storage.CounterpartyOverride{
		ID: 1, UserID: userID, Counterparty: cp, CategoryID: catID,
		Confirmations: u.returnConfirmations,
	}, nil
}

// ----------------------------------------------------------------------------
// Categorize — precedence
// ----------------------------------------------------------------------------

func TestCategorize_LearnedOverride_Wins(t *testing.T) {
	repo := newFakeRepo()
	repo.overrides["ozon"] = storage.CounterpartyOverride{
		UserID: 7, Counterparty: "ozon", CategoryID: 50, Confirmations: domain.LearnThreshold,
	}
	// A keyword rule points to a different category; override must win.
	repo.rules = []storage.CategorizationRule{
		{ID: 1, UserID: 7, Keyword: "ozon", CategoryID: 60, Source: domain.RuleUser, Priority: 100},
	}
	svc := NewService(repo)

	sug, err := svc.Categorize(context.Background(), 7, MatchRequest{Counterparty: "Ozon"})
	if err != nil {
		t.Fatalf("Categorize: %v", err)
	}
	if sug.Tier != TierOverrideLearned {
		t.Errorf("tier = %v, want TierOverrideLearned", sug.Tier)
	}
	if sug.CategoryID != 50 {
		t.Errorf("category = %d, want 50 (learned override)", sug.CategoryID)
	}
	if !sug.Confidence.Equal(mustParse(ConfidenceOverrideLearned)) {
		t.Errorf("confidence = %s, want %s", sug.Confidence, ConfidenceOverrideLearned)
	}
}

func TestCategorize_TentativeOverride_BelowThreshold(t *testing.T) {
	repo := newFakeRepo()
	repo.overrides["starbucks"] = storage.CounterpartyOverride{
		UserID: 7, Counterparty: "starbucks", CategoryID: 50, Confirmations: 1,
	}
	svc := NewService(repo)

	sug, err := svc.Categorize(context.Background(), 7, MatchRequest{Counterparty: "Starbucks"})
	if err != nil {
		t.Fatalf("Categorize: %v", err)
	}
	if sug.Tier != TierOverrideTentative {
		t.Errorf("tier = %v, want TierOverrideTentative", sug.Tier)
	}
	if !sug.Confidence.Equal(mustParse(ConfidenceOverrideTentative)) {
		t.Errorf("confidence = %s, want %s", sug.Confidence, ConfidenceOverrideTentative)
	}
}

func TestCategorize_KeywordRule_UserPriority_OverSystem(t *testing.T) {
	repo := newFakeRepo()
	// Two rules match "ozon"; user rule (priority 100) must outrank system.
	repo.rules = []storage.CategorizationRule{
		{ID: 1, UserID: 7, Keyword: "ozon", CategoryID: 60, Source: domain.RuleUser, Priority: 100},
		{ID: 2, UserID: 7, Keyword: "ozon", CategoryID: 70, Source: domain.RuleSystem, Priority: 0},
	}
	svc := NewService(repo)

	sug, err := svc.Categorize(context.Background(), 7, MatchRequest{Counterparty: "Ozon"})
	if err != nil {
		t.Fatalf("Categorize: %v", err)
	}
	if sug.Tier != TierKeyword {
		t.Errorf("tier = %v, want TierKeyword", sug.Tier)
	}
	if sug.CategoryID != 60 {
		t.Errorf("category = %d, want 60 (user rule wins)", sug.CategoryID)
	}
	if sug.RuleID != 1 {
		t.Errorf("rule_id = %d, want 1", sug.RuleID)
	}
}

func TestCategorize_Keyword_MatchesDescription(t *testing.T) {
	repo := newFakeRepo()
	repo.rules = []storage.CategorizationRule{
		{ID: 3, UserID: 7, Keyword: "аптек", CategoryID: 80, Source: domain.RuleSystem, Priority: 0},
	}
	svc := NewService(repo)

	// Counterparty empty, keyword only in description.
	sug, err := svc.Categorize(context.Background(), 7, MatchRequest{Description: "Покупка в Аптеке"})
	if err != nil {
		t.Fatalf("Categorize: %v", err)
	}
	if sug.CategoryID != 80 {
		t.Errorf("category = %d, want 80 (keyword in description)", sug.CategoryID)
	}
}

func TestCategorize_NoMatch_ReturnsTierNone(t *testing.T) {
	repo := newFakeRepo()
	repo.rules = []storage.CategorizationRule{
		{ID: 1, UserID: 7, Keyword: "ozon", CategoryID: 60, Source: domain.RuleSystem, Priority: 0},
	}
	svc := NewService(repo)

	sug, err := svc.Categorize(context.Background(), 7, MatchRequest{Counterparty: "неизвестный контрагент"})
	if err != nil {
		t.Fatalf("Categorize: %v", err)
	}
	if sug.Tier != TierNone {
		t.Errorf("tier = %v, want TierNone", sug.Tier)
	}
	if sug.CategoryID != 0 {
		t.Errorf("category = %d, want 0 on no match", sug.CategoryID)
	}
}

func TestCategorize_Normalization_CaseInsensitive(t *testing.T) {
	repo := newFakeRepo()
	repo.overrides["wildberries"] = storage.CounterpartyOverride{
		UserID: 7, Counterparty: "wildberries", CategoryID: 90, Confirmations: domain.LearnThreshold,
	}
	svc := NewService(repo)

	// Mixed-case + whitespace must normalize to the stored lowercase key.
	sug, err := svc.Categorize(context.Background(), 7, MatchRequest{Counterparty: "  WildBerries  "})
	if err != nil {
		t.Fatalf("Categorize: %v", err)
	}
	if sug.CategoryID != 90 {
		t.Errorf("category = %d, want 90 (normalized override)", sug.CategoryID)
	}
}

// ----------------------------------------------------------------------------
// CategorizeForCreate — adapter view
// ----------------------------------------------------------------------------

func TestCategorizeForCreate_NoMatch_NilConfidence(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	catID, conf, err := svc.CategorizeForCreate(context.Background(), 7, "ghost", "")
	if err != nil {
		t.Fatalf("CategorizeForCreate: %v", err)
	}
	if catID != 0 || conf != nil {
		t.Errorf("expected (0, nil) on no match, got (%d, %v)", catID, conf)
	}
}

func TestCategorizeForCreate_Match_ReturnsNonNilConfidence(t *testing.T) {
	repo := newFakeRepo()
	repo.rules = []storage.CategorizationRule{
		{ID: 1, UserID: 7, Keyword: "магнит", CategoryID: 10, Source: domain.RuleSystem, Priority: 0},
	}
	svc := NewService(repo)

	catID, conf, err := svc.CategorizeForCreate(context.Background(), 7, "МАГНИТ", "")
	if err != nil {
		t.Fatalf("CategorizeForCreate: %v", err)
	}
	if catID != 10 || conf == nil {
		t.Errorf("expected (10, non-nil), got (%d, %v)", catID, conf)
	}
}

// ----------------------------------------------------------------------------
// Confirm — Learn path
// ----------------------------------------------------------------------------

func TestConfirm_Success_IncrementsCounter(t *testing.T) {
	repo := newFakeRepo()
	repo.cats[10] = storage.Category{ID: 10, UserID: 7}
	repo.upsert.returnConfirmations = 3
	svc := NewService(repo)

	n, err := svc.Confirm(context.Background(), 7, LearnInput{Counterparty: "Ozon", CategoryID: 10}, repo.upsert)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if n != 3 {
		t.Errorf("confirmations = %d, want 3", n)
	}
	if repo.upsert.lastCall.counterparty != "ozon" {
		t.Errorf("upsert got counterparty %q, want normalized 'ozon'", repo.upsert.lastCall.counterparty)
	}
}

func TestConfirm_RejectsInvalidInput(t *testing.T) {
	repo := newFakeRepo()
	repo.cats[10] = storage.Category{ID: 10, UserID: 7}
	svc := NewService(repo)

	cases := []struct {
		name string
		in   LearnInput
	}{
		{"empty counterparty", LearnInput{CategoryID: 10}},
		{"zero category", LearnInput{Counterparty: "x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.Confirm(context.Background(), 7, c.in, repo.upsert)
			if !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("expected ErrInvalidArgument, got %v", err)
			}
		})
	}
}

func TestConfirm_CategoryNotFound(t *testing.T) {
	repo := newFakeRepo() // no categories seeded
	svc := NewService(repo)

	_, err := svc.Confirm(context.Background(), 7, LearnInput{Counterparty: "x", CategoryID: 999}, repo.upsert)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for unknown category, got %v", err)
	}
}

func TestConfirm_StorageError_Propagates(t *testing.T) {
	repo := newFakeRepo()
	repo.cats[10] = storage.Category{ID: 10, UserID: 7}
	repo.upsert.failWith = errors.New("db down")
	svc := NewService(repo)

	_, err := svc.Confirm(context.Background(), 7, LearnInput{Counterparty: "x", CategoryID: 10}, repo.upsert)
	if err == nil {
		t.Fatalf("expected error from upsert, got nil")
	}
}

// ----------------------------------------------------------------------------
// defaults sanity — every rule's category must exist in SystemCategories
// ----------------------------------------------------------------------------

func TestSystemKeywordRules_ReferenceExistingCategories(t *testing.T) {
	known := make(map[string]struct{}, len(SystemCategories))
	for _, c := range SystemCategories {
		known[c] = struct{}{}
	}
	for _, r := range SystemKeywordRules {
		if strings.TrimSpace(r.Keyword) == "" {
			t.Errorf("empty keyword in SystemKeywordRules")
			continue
		}
		if _, ok := known[r.Category]; !ok {
			t.Errorf("keyword %q references unknown category %q", r.Keyword, r.Category)
		}
	}
}

func TestConfidenceForTier_ConsistentWithFloor(t *testing.T) {
	// Keyword + learned must be ABOVE the confirm floor; tentative BELOW it.
	floor := domain.MinCategorizationConfidence
	for _, tier := range []MatchTier{TierKeyword, TierOverrideLearned} {
		d := mustParse(ConfidenceForTier(tier))
		if !d.GreaterThan(floor) {
			t.Errorf("tier %d confidence %s not above floor %s", tier, d, floor)
		}
	}
	d := mustParse(ConfidenceForTier(TierOverrideTentative))
	if !d.LessThan(floor) {
		t.Errorf("tentative confidence %s not below floor %s", d, floor)
	}
}
