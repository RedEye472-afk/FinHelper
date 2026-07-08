package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/RedEye472-afk/FinHelper/backend/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/service/operations"
	"github.com/RedEye472-afk/FinHelper/backend/pkg/storage"
)

// opsTestEnv wires a real OperationsHandler with a fake repo, so we exercise
// the HTTP + service layer end-to-end without a database. The fake is the
// same one used by the service unit tests (kept in the service package and
// promoted via a thin wrapper here so we can reuse it).
type opsTestEnv struct {
	t      *testing.T
	srv    *httptest.Server
	repo   *fakeOpsRepo
	userID int64
	token  string
}

// fakeOpsRepo is a self-contained in-memory repo for the http tests. (The
// service package has its own fake; we duplicate here to avoid an import cycle
// — the service fake is internal to its package's tests.)
type fakeOpsRepo struct {
	accounts map[int64]storage.Account
	ops      map[int64]storage.Operation
	nextID   int64
	sums     map[int64]domain.Money
	balances map[int64]domain.Money
}

func newFakeOpsRepo() *fakeOpsRepo {
	return &fakeOpsRepo{
		accounts: make(map[int64]storage.Account),
		ops:      make(map[int64]storage.Operation),
		sums:     make(map[int64]domain.Money),
		balances: make(map[int64]domain.Money),
	}
}

func (f *fakeOpsRepo) CreateOperation(_ context.Context, op storage.Operation) (storage.Operation, error) {
	op.ID = f.nextID + 1
	f.nextID = op.ID
	op.CreatedAt = time.Now()
	op.UpdatedAt = op.CreatedAt
	f.ops[op.ID] = op
	return op, nil
}
func (f *fakeOpsRepo) GetOperation(_ context.Context, userID, id int64) (storage.Operation, error) {
	o, ok := f.ops[id]
	if !ok || o.UserID != userID {
		return storage.Operation{}, storage.ErrOperationNotFound
	}
	return o, nil
}
func (f *fakeOpsRepo) GetOperationByCalcID(_ context.Context, userID int64, calcID string) (storage.Operation, error) {
	for _, o := range f.ops {
		if o.UserID == userID && o.CalcID == calcID {
			return o, nil
		}
	}
	return storage.Operation{}, storage.ErrOperationNotFound
}
func (f *fakeOpsRepo) ListOperations(_ context.Context, userID int64, _ storage.OperationFilter, page storage.Page) ([]storage.Operation, error) {
	limit := page.Limit
	if limit <= 0 {
		limit = 50
	}
	var out []storage.Operation
	for _, o := range f.ops {
		if o.UserID == userID {
			out = append(out, o)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (f *fakeOpsRepo) DeleteOperation(_ context.Context, userID, id int64) error {
	o, ok := f.ops[id]
	if !ok || o.UserID != userID {
		return storage.ErrOperationNotFound
	}
	delete(f.ops, id)
	return nil
}
func (f *fakeOpsRepo) UpdateOperationCategory(_ context.Context, userID, id int64, cat *int64, _ *decimal.Decimal) error {
	o, ok := f.ops[id]
	if !ok || o.UserID != userID {
		return storage.ErrOperationNotFound
	}
	o.CategoryID = cat
	f.ops[id] = o
	return nil
}
func (f *fakeOpsRepo) GetAccount(_ context.Context, userID, id int64) (storage.Account, error) {
	a, ok := f.accounts[id]
	if !ok || a.UserID != userID {
		return storage.Account{}, storage.ErrAccountNotFound
	}
	return a, nil
}
func (f *fakeOpsRepo) SetAccountBalance(_ context.Context, userID, id int64, b domain.Money) error {
	f.balances[id] = b
	f.accounts[id] = storage.Account{ID: id, UserID: userID, Balance: b}
	return nil
}
func (f *fakeOpsRepo) SumByAccountSince(_ context.Context, accountID int64) (domain.Money, error) {
	return f.sums[accountID], nil
}

func newOpsTestEnv(t *testing.T) *opsTestEnv {
	t.Helper()
	repo := newFakeOpsRepo()
	repo.accounts[10] = storage.Account{ID: 10, UserID: 7}
	repo.accounts[11] = storage.Account{ID: 11, UserID: 7}
	svc := operations.NewService(repo)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	h := NewOperationsHandler(svc, logger)

	// Use chi (same router lib the real app uses) so {id} URL params work.
	r := chi.NewRouter()
	// Fake auth: inject a fixed user_id into context for every request.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), keyUserID, int64(7))
			ctx = context.WithValue(ctx, keyUserHash, "uhsh-test")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	// Mount under /api/v1 so test URLs mirror the real API shape.
	r.Route("/api/v1", func(r chi.Router) {
		h.Register(r)
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &opsTestEnv{t: t, srv: srv, repo: repo, userID: 7}
}

func TestOperationsHandler_Create_Success(t *testing.T) {
	env := newOpsTestEnv(t)
	body := map[string]any{
		"type":       "expense",
		"amount":     "470.50",
		"account_id": 10,
		"counterparty": "Перевод от ИВАН ИВАНОВИЧ И.",
	}
	resp, raw := postJSON(t, env.srv.URL+"/api/v1/operations", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", resp.StatusCode, raw)
	}
	var out operationResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Amount != "470.50" {
		t.Errorf("amount = %s, want 470.50", out.Amount)
	}
	// PII masked on the way through.
	if out.Counterparty != "Перевод от [PERSON]" {
		t.Errorf("counterparty not masked: %q", out.Counterparty)
	}
}

func TestOperationsHandler_Create_InvalidType(t *testing.T) {
	env := newOpsTestEnv(t)
	body := map[string]any{
		"type": "bogus", "amount": "1.00", "account_id": 10,
	}
	resp, raw := postJSON(t, env.srv.URL+"/api/v1/operations", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body=%s", resp.StatusCode, raw)
	}
}

func TestOperationsHandler_Create_InvalidAmount(t *testing.T) {
	env := newOpsTestEnv(t)
	body := map[string]any{
		"type": "expense", "amount": "not-a-number", "account_id": 10,
	}
	resp, _ := postJSON(t, env.srv.URL+"/api/v1/operations", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestOperationsHandler_Get_NotFound(t *testing.T) {
	env := newOpsTestEnv(t)
	resp, err := http.Get(env.srv.URL + "/api/v1/operations/999")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestOperationsHandler_Create_ThenGet(t *testing.T) {
	env := newOpsTestEnv(t)

	// create
	body := map[string]any{
		"type": "income", "amount": "1000.00", "account_id": 10,
		"description": "Зарплата",
	}
	resp, raw := postJSON(t, env.srv.URL+"/api/v1/operations", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", resp.StatusCode, raw)
	}
	var created operationResponse
	_ = json.Unmarshal(raw, &created)

	// get by id
	resp2, err := http.Get(env.srv.URL + "/api/v1/operations/" + strconv.FormatInt(created.ID, 10))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp2.Body.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(resp2.Body)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", resp2.StatusCode, out.String())
	}
	var got operationResponse
	_ = json.Unmarshal(out.Bytes(), &got)
	if got.ID != created.ID || got.Amount != "1000.00" {
		t.Errorf("got = %+v", got)
	}
}

func TestOperationsHandler_List_FiltersAndPaginates(t *testing.T) {
	env := newOpsTestEnv(t)
	// Seed 3 ops.
	for i := 0; i < 3; i++ {
		body := map[string]any{
			"type": "expense", "amount": "10.00", "account_id": 10,
		}
		resp, _ := postJSON(t, env.srv.URL+"/api/v1/operations", body)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("seed status = %d", resp.StatusCode)
		}
	}
	// List with limit=2 — expect 2 items + more=true.
	resp, err := http.Get(env.srv.URL + "/api/v1/operations?limit=2")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer resp.Body.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.StatusCode, out.String())
	}
	var lr listResponse
	_ = json.Unmarshal(out.Bytes(), &lr)
	if len(lr.Items) != 2 || !lr.More {
		t.Errorf("len=%d more=%v (want 2, true)", len(lr.Items), lr.More)
	}
}

func TestOperationsHandler_Delete_Success(t *testing.T) {
	env := newOpsTestEnv(t)
	body := map[string]any{
		"type": "expense", "amount": "5.00", "account_id": 10,
	}
	createResp, raw := postJSON(t, env.srv.URL+"/api/v1/operations", body)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", createResp.StatusCode, raw)
	}
	var created operationResponse
	_ = json.Unmarshal(raw, &created)

	req, _ := http.NewRequest(http.MethodDelete, env.srv.URL+"/api/v1/operations/"+strconv.FormatInt(created.ID, 10), nil)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", resp2.StatusCode)
	}
}

func TestOperationsHandler_SetCategory_Success(t *testing.T) {
	env := newOpsTestEnv(t)
	body := map[string]any{
		"type": "expense", "amount": "5.00", "account_id": 10,
	}
	createResp, raw := postJSON(t, env.srv.URL+"/api/v1/operations", body)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", createResp.StatusCode, raw)
	}
	var created operationResponse
	_ = json.Unmarshal(raw, &created)

	catID := int64(3)
	conf := "0.92"
	body2, _ := json.Marshal(map[string]any{"category_id": catID, "confidence": conf})
	req, _ := http.NewRequest(http.MethodPatch, env.srv.URL+"/api/v1/operations/"+strconv.FormatInt(created.ID, 10)+"/category", bytes.NewReader(body2))
	req.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("patch status = %d, want 204", resp2.StatusCode)
	}
}

func TestOperationsHandler_Unauthorized_WithoutContext(t *testing.T) {
	// A bare handler with no user in context returns 401.
	repo := newFakeOpsRepo()
	svc := operations.NewService(repo)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	h := NewOperationsHandler(svc, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/operations", h.Create)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{"type": "expense", "amount": "1.00", "account_id": 10})
	resp, err := http.Post(srv.URL+"/api/v1/operations", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}
