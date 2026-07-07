package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/pkg/auth"
	"github.com/RedEye472-afk/FinHelper/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/pkg/service/budget"
	"github.com/RedEye472-afk/FinHelper/pkg/storage"
)

// budgetFakeRepo is an in-memory budget.Repo for HTTP tests. It stages a
// single budget and reports a fixed spend so the /status path is testable
// without a DB.
type budgetFakeRepo struct {
	budget storage.Budget
	spend  domain.Money
}

func (f *budgetFakeRepo) CreateBudget(_ context.Context, b storage.Budget) (storage.Budget, error) {
	b.ID = 1
	f.budget = b
	return b, nil
}
func (f *budgetFakeRepo) GetBudget(_ context.Context, userID, id int64) (storage.Budget, error) {
	if f.budget.ID == 0 || f.budget.UserID != userID || f.budget.ID != id {
		return storage.Budget{}, storage.ErrBudgetNotFound
	}
	return f.budget, nil
}
func (f *budgetFakeRepo) ListBudgets(_ context.Context, userID int64) ([]storage.Budget, error) {
	if f.budget.UserID == userID {
		return []storage.Budget{f.budget}, nil
	}
	return []storage.Budget{}, nil
}
func (f *budgetFakeRepo) UpdateBudget(_ context.Context, b storage.Budget) (storage.Budget, error) {
	if f.budget.ID != b.ID {
		return storage.Budget{}, storage.ErrBudgetNotFound
	}
	f.budget.LimitAmount = b.LimitAmount
	f.budget.RolloverPolicy = b.RolloverPolicy
	f.budget.IsActive = b.IsActive
	return f.budget, nil
}
func (f *budgetFakeRepo) DeleteBudget(_ context.Context, _, _ int64) error {
	f.budget = storage.Budget{}
	return nil
}
func (f *budgetFakeRepo) SpendForCategory(_ context.Context, _, _ int64, _, _ time.Time) (domain.Money, error) {
	return f.spend, nil
}

func newBudgetTestEnv(t *testing.T) (*httptest.Server, *auth.JWTIssuer, *budgetFakeRepo) {
	t.Helper()
	issuer, err := auth.NewJWTIssuer(
		"access-test-secret-must-be-32+-chars-long",
		"refresh-test-secret-must-be-32+-chars-long",
		15*time.Minute, time.Hour,
	)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	logger := testSlogLogger()
	repo := &budgetFakeRepo{}
	// Pin the clock to the last nanosecond of a 30-day month so projectSpend
	// returns spent as-is (no extrapolation) and the status tests are
	// deterministic regardless of the real run date.
	now := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	svc := budget.NewServiceWithNow(repo, func() time.Time { return now })
	mw := NewAuthMiddleware(issuer, logger)
	h := NewBudgetHandler(svc, logger)

	// Use chi so {id} path params resolve exactly as in production routes.
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(mw.Wrap)
		r.Get("/budgets", h.List)
		r.Post("/budgets", h.Create)
		r.Get("/budgets/{id}/status", h.Status)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, issuer, repo
}

// testSlogLogger returns a discard logger, local to this file so we don't
// collide with helpers in other test files.
func testSlogLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

func TestBudget_Create_Success(t *testing.T) {
	srv, issuer, repo := newBudgetTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")

	resp, body := doBudgetReq(t, http.MethodPost, srv.URL+"/budgets", tok,
		map[string]any{"category_id": 10, "limit_amount": "15000.00", "rollover_policy": "months_3"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	if repo.budget.RolloverPolicy != domain.RolloverMonths3 {
		t.Errorf("repo rollover = %s, want months_3", repo.budget.RolloverPolicy)
	}
}

func TestBudget_Create_RejectsZeroLimit(t *testing.T) {
	srv, issuer, _ := newBudgetTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")

	resp, _ := doBudgetReq(t, http.MethodPost, srv.URL+"/budgets", tok,
		map[string]any{"category_id": 10, "limit_amount": "0"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestBudget_Unauthorized_WithoutToken(t *testing.T) {
	srv, _, _ := newBudgetTestEnv(t)
	resp, _ := doBudgetReq(t, http.MethodGet, srv.URL+"/budgets", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestBudget_Status_OK(t *testing.T) {
	srv, issuer, repo := newBudgetTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")
	// Seed an active budget directly in the repo (bypass Create).
	repo.budget = storage.Budget{
		ID: 1, UserID: 7, CategoryID: 10,
		LimitAmount:    domain.MustParseMoney("15000.00"),
		RolloverPolicy: domain.RolloverNone, IsActive: true,
	}
	// Spend well under limit → status ok.
	repo.spend = domain.MustParseMoney("5000.00")

	resp, body := doBudgetReq(t, http.MethodGet, srv.URL+"/budgets/1/status", tok, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var out statusResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Status != string(budget.StatusOK) {
		t.Errorf("status = %s, want ok (spent=%s)", out.Status, out.Spent)
	}
}

func TestBudget_Status_NotFound(t *testing.T) {
	srv, issuer, _ := newBudgetTestEnv(t)
	tok, _, _ := issuer.IssueAccess(7, "hash", "")

	resp, _ := doBudgetReq(t, http.MethodGet, srv.URL+"/budgets/999/status", tok, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// doBudgetReq posts/gets JSON with a Bearer token. nil body → GET-style.
func doBudgetReq(t *testing.T, method, url, token string, body any) (*http.Response, []byte) {
	t.Helper()
	var req *http.Request
	if body != nil {
		buf, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, url, bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	var out bytes.Buffer
	_, _ = out.ReadFrom(resp.Body)
	return resp, out.Bytes()
}
