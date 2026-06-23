package operations

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/internal/domain"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// fakeRepo is an in-memory OperationRepo for service tests. It does NOT
// implement SQL semantics; it records calls so we can assert on the orchestration.
type fakeRepo struct {
	// Data state
	accounts  map[int64]storage.Account        // keyed by account id
	ops       map[int64]storage.Operation       // keyed by op id
	calcIndex map[string]int64                  // user_id:calc_id -> op id
	sums      map[int64]domain.Money            // account id -> precomputed sum
	balances  map[int64]domain.Money            // account id -> last set balance

	// Call tracking
	createdOps []storage.Operation
	deletedIDs []int64
	failWith   error // if set, every method returns this
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		accounts:  make(map[int64]storage.Account),
		ops:       make(map[int64]storage.Operation),
		calcIndex: make(map[string]int64),
		sums:      make(map[int64]domain.Money),
		balances:  make(map[int64]domain.Money),
	}
}

func (f *fakeRepo) CreateOperation(_ context.Context, op storage.Operation) (storage.Operation, error) {
	if f.failWith != nil {
		return storage.Operation{}, f.failWith
	}
	if op.UserID == 0 {
		return storage.Operation{}, errors.New("no user")
	}
	// Duplicate calc_id check.
	key := fmt.Sprintf("%d:%s", op.UserID, op.CalcID)
	if _, exists := f.calcIndex[key]; exists {
		return storage.Operation{}, storage.ErrOperationExists
	}
	op.ID = int64(len(f.ops) + 1)
	op.CreatedAt = time.Now()
	op.UpdatedAt = op.CreatedAt
	f.ops[op.ID] = op
	f.calcIndex[key] = op.ID
	f.createdOps = append(f.createdOps, op)
	return op, nil
}

func (f *fakeRepo) GetOperation(_ context.Context, userID, id int64) (storage.Operation, error) {
	if f.failWith != nil {
		return storage.Operation{}, f.failWith
	}
	op, ok := f.ops[id]
	if !ok || op.UserID != userID {
		return storage.Operation{}, storage.ErrOperationNotFound
	}
	return op, nil
}

func (f *fakeRepo) GetOperationByCalcID(_ context.Context, userID int64, calcID string) (storage.Operation, error) {
	id, ok := f.calcIndex[fmt.Sprintf("%d:%s", userID, calcID)]
	if !ok {
		return storage.Operation{}, storage.ErrOperationNotFound
	}
	return f.ops[id], nil
}

func (f *fakeRepo) ListOperations(_ context.Context, userID int64, _ storage.OperationFilter, page storage.Page) ([]storage.Operation, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	limit := page.Limit
	if limit <= 0 {
		limit = 5
	}
	var out []storage.Operation
	for _, op := range f.ops {
		if op.UserID == userID {
			out = append(out, op)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeRepo) DeleteOperation(_ context.Context, userID, id int64) error {
	op, ok := f.ops[id]
	if !ok || op.UserID != userID {
		return storage.ErrOperationNotFound
	}
	delete(f.ops, id)
	f.deletedIDs = append(f.deletedIDs, id)
	return nil
}

func (f *fakeRepo) UpdateOperationCategory(_ context.Context, userID, id int64, cat *int64, _ *decimal.Decimal) error {
	op, ok := f.ops[id]
	if !ok || op.UserID != userID {
		return storage.ErrOperationNotFound
	}
	op.CategoryID = cat
	f.ops[id] = op
	return nil
}

func (f *fakeRepo) GetAccount(_ context.Context, userID, id int64) (storage.Account, error) {
	a, ok := f.accounts[id]
	if !ok || a.UserID != userID {
		return storage.Account{}, storage.ErrAccountNotFound
	}
	return a, nil
}

func (f *fakeRepo) SetAccountBalance(_ context.Context, userID, id int64, b domain.Money) error {
	if _, ok := f.accounts[id]; !ok {
		return storage.ErrAccountNotFound
	}
	f.balances[id] = b
	f.accounts[id] = storage.Account{ID: id, UserID: userID, Balance: b}
	return nil
}

func (f *fakeRepo) SumByAccountSince(_ context.Context, accountID int64) (domain.Money, error) {
	return f.sums[accountID], nil
}

// ----------------------------------------------------------------------------
// Tests
// ----------------------------------------------------------------------------

// fakeCategorizer is an operations.Categorizer stub. It returns whatever the
// test wired into next, and records the call so we can assert the matcher saw
// the PII-masked (not raw) counterparty.
type fakeCategorizer struct {
	nextCat  int64
	nextConf *decimal.Decimal
	called   bool
	gotCP    string
	gotDesc  string
}

func (f *fakeCategorizer) CategorizeForCreate(_ context.Context, _ int64, counterparty, description string) (int64, *decimal.Decimal, error) {
	f.called = true
	f.gotCP = counterparty
	f.gotDesc = description
	return f.nextCat, f.nextConf, nil
}

func TestCreate_AutoCategorizes_WhenCategoryAbsent(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	repo.sums[10] = domain.Zero
	svc := NewService(repo)
	cat := &fakeCategorizer{nextCat: 42, nextConf: ptrConf("0.75")}
	svc.SetCategorizer(cat)

	op, err := svc.Create(context.Background(), 7, CreateInput{
		Type:         domain.OpExpense,
		Amount:       domain.MustParseMoney("500.00"),
		AccountID:    10,
		Counterparty: "Магнит",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Categorizer was invoked and its suggestion applied.
	if !cat.called {
		t.Errorf("categorizer not called")
	}
	if op.CategoryID == nil || *op.CategoryID != 42 {
		t.Errorf("category_id = %v, want 42", op.CategoryID)
	}
	// The matcher sees the masked counterparty (same string here, but the
	// contract is "post-mask").
	if cat.gotCP != "Магнит" {
		t.Errorf("categorizer got counterparty %q", cat.gotCP)
	}
}

func TestCreate_DoesNotOverrideExplicitCategory(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	repo.sums[10] = domain.Zero
	svc := NewService(repo)
	cat := &fakeCategorizer{nextCat: 42}
	svc.SetCategorizer(cat)

	explicit := int64(99)
	op, err := svc.Create(context.Background(), 7, CreateInput{
		Type:      domain.OpExpense,
		Amount:    domain.MustParseMoney("500.00"),
		AccountID: 10,
		CategoryID: &explicit,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cat.called {
		t.Errorf("categorizer should NOT be called when category is explicit")
	}
	if op.CategoryID == nil || *op.CategoryID != 99 {
		t.Errorf("category_id = %v, want explicit 99", op.CategoryID)
	}
}

func TestCreate_CategorizerNil_LeavesCategoryEmpty(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	repo.sums[10] = domain.Zero
	svc := NewService(repo) // no categorizer attached

	op, err := svc.Create(context.Background(), 7, CreateInput{
		Type:      domain.OpExpense,
		Amount:    domain.MustParseMoney("500.00"),
		AccountID: 10,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if op.CategoryID != nil {
		t.Errorf("category_id = %v, want nil when no categorizer", op.CategoryID)
	}
}

// ptrConf parses a confidence string into a *decimal.Decimal for the stub.
func ptrConf(s string) *decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return &d
}

func TestCreate_Success_PII_Masked_And_Balance_Recomputed(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	repo.sums[10] = domain.MustParseMoney("-500.00") // simulate recompute result
	svc := NewService(repo)

	in := CreateInput{
		Type:         domain.OpExpense,
		Amount:       domain.MustParseMoney("500.00"),
		AccountID:    10,
		Counterparty: "Перевод от ИВАН ИВАНОВИЧ И.",
		Description:  "Звонок +7 (999) 123-45-67",
	}
	op, err := svc.Create(context.Background(), 7, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// PII masked before persistence.
	if op.Counterparty != "Перевод от [PERSON]" {
		t.Errorf("counterparty not masked: %q", op.Counterparty)
	}
	if op.Description != "Звонок [PHONE]" {
		t.Errorf("description not masked: %q", op.Description)
	}
	// calc_id was generated server-side.
	if op.CalcID == "" {
		t.Errorf("calc_id should be auto-generated")
	}
	// Balance recomputed to the repo's precomputed sum.
	if !repo.balances[10].Equal(domain.MustParseMoney("-500.00")) {
		t.Errorf("balance = %s, want -500.00", repo.balances[10])
	}
}

func TestCreate_Idempotent_OnCalcID(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	repo.sums[10] = domain.Zero
	svc := NewService(repo)

	first, err := svc.Create(context.Background(), 7, CreateInput{
		Type: domain.OpExpense, Amount: domain.MustParseMoney("100.00"),
		AccountID: 10, CalcID: "dup-1",
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	// Second call with same calc_id should return the SAME operation, not error.
	second, err := svc.Create(context.Background(), 7, CreateInput{
		Type: domain.OpExpense, Amount: domain.MustParseMoney("100.00"),
		AccountID: 10, CalcID: "dup-1",
	})
	if err != nil {
		t.Fatalf("idempotent Create: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("idempotency broken: first=%d second=%d", first.ID, second.ID)
	}
}

func TestCreate_RejectsInvalidInput(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	svc := NewService(repo)

	cases := []struct {
		name string
		in   CreateInput
	}{
		{"zero amount", CreateInput{Type: domain.OpExpense, Amount: domain.Zero, AccountID: 10}},
		{"unknown type", CreateInput{Type: "bogus", Amount: domain.MustParseMoney("1.00"), AccountID: 10}},
		{"missing account", CreateInput{Type: domain.OpExpense, Amount: domain.MustParseMoney("1.00"), AccountID: 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), 7, c.in)
			if !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("expected ErrInvalidArgument, got %v", err)
			}
		})
	}
}

func TestCreate_AccountNotOwned(t *testing.T) {
	repo := newFakeRepo()
	// account 10 exists but belongs to a different user (we never seed it for user 7)
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), 7, CreateInput{
		Type: domain.OpExpense, Amount: domain.MustParseMoney("1.00"), AccountID: 10,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for unowned account, got %v", err)
	}
}

func TestCreate_Transfer_RecomputesBothBalances(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	repo.accounts[11] = storage.Account{ID: 11, UserID: 7}
	dst := int64(11)
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), 7, CreateInput{
		Type: domain.OpTransfer,
		Amount: domain.MustParseMoney("1000.00"),
		AccountID: 10,
		AccountDstID: &dst,
	})
	if err != nil {
		t.Fatalf("Create transfer: %v", err)
	}
	// Both accounts should have been touched by SetAccountBalance.
	if _, ok := repo.balances[10]; !ok {
		t.Errorf("source balance not recomputed")
	}
	if _, ok := repo.balances[11]; !ok {
		t.Errorf("destination balance not recomputed")
	}
}

func TestList_More_Flag(t *testing.T) {
	repo := newFakeRepo()
	// Seed 7 ops for user 7.
	for i := 1; i <= 7; i++ {
		op := storage.Operation{
			ID: int64(i), UserID: 7,
			Type: domain.OpIncome, Amount: domain.MustParseMoney("1.00"),
			AccountID: 10, OperationDate: time.Now(),
		}
		op.CalcID = fmt.Sprintf("c%d", i)
		repo.ops[op.ID] = op
		repo.calcIndex[fmt.Sprintf("%d:%s", op.UserID, op.CalcID)] = op.ID
	}
	svc := NewService(repo)

	// Ask for page of 5; storage returns min(limit+1=6, available=7)=6 rows;
	// service slices to 5 and signals more=true.
	items, more, err := svc.List(context.Background(), 7, storage.OperationFilter{}, storage.Page{Limit: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 5 {
		t.Errorf("len = %d, want 5", len(items))
	}
	if !more {
		t.Errorf("expected more=true")
	}
}

func TestDelete_RecomputesBalance(t *testing.T) {
	repo := newFakeRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	repo.ops[1] = storage.Operation{ID: 1, UserID: 7, AccountID: 10, Amount: domain.MustParseMoney("1.00"), OperationDate: time.Now()}
	svc := NewService(repo)

	if err := svc.Delete(context.Background(), 7, 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := repo.balances[10]; !ok {
		t.Errorf("balance should have been recomputed after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	err := svc.Delete(context.Background(), 7, 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
