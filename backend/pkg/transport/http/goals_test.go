// Package http (test): goals_test.go integration-tests the goals HTTP handler
// via httptest.Server + chi router + an in-memory goalsFakeRepo. It exercises
// the full HTTP → service → fake-repo stack end-to-end without a database,
// mirroring the patterns established by operations_test.go and budgets_test.go.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/RedEye472-afk/FinHelper/pkg/domain"
	"github.com/RedEye472-afk/FinHelper/pkg/service/goals"
	"github.com/RedEye472-afk/FinHelper/pkg/storage"
)

// ---- in-memory goalsFakeRepo ----

// goalsFakeRepo is an in-memory goals.Repo for HTTP integration tests. It
// mirrors the service package's unexported fakeRepo (kept here to avoid an
// import cycle into the service's test-only helpers). Goals are keyed by id;
// contributions by clientKey(userID, goalID, contributionID) for idempotency.
type goalsFakeRepo struct {
	goals         map[int64]domain.Goal
	contributions map[string]domain.GoalContribution
	nextGoalID    int64
	nextContribID int64
}

func newGoalsFakeRepo() *goalsFakeRepo {
	return &goalsFakeRepo{
		goals:         make(map[int64]domain.Goal),
		contributions: make(map[string]domain.GoalContribution),
	}
}

// clientKey mirrors goals.clientKey (unexported there). Format: "u:g:cid".
func goalsClientKey(userID, goalID int64, contributionID string) string {
	return strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(goalID, 10) + ":" + contributionID
}

func (f *goalsFakeRepo) CreateGoal(_ context.Context, g domain.Goal) (domain.Goal, error) {
	f.nextGoalID++
	g.ID = f.nextGoalID
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now
	f.goals[g.ID] = g
	return g, nil
}

func (f *goalsFakeRepo) GetGoal(_ context.Context, userID, id int64) (domain.Goal, error) {
	g, ok := f.goals[id]
	if !ok || g.UserID != userID {
		return domain.Goal{}, storage.ErrGoalNotFound
	}
	return g, nil
}

func (f *goalsFakeRepo) ListGoals(_ context.Context, userID int64) ([]domain.Goal, error) {
	var out []domain.Goal
	for _, g := range f.goals {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *goalsFakeRepo) UpdateGoal(_ context.Context, g domain.Goal) (domain.Goal, error) {
	existing, ok := f.goals[g.ID]
	if !ok || existing.UserID != g.UserID {
		return domain.Goal{}, storage.ErrGoalNotFound
	}
	g.CreatedAt = existing.CreatedAt
	g.UpdatedAt = time.Now()
	f.goals[g.ID] = g
	return g, nil
}

func (f *goalsFakeRepo) DeleteGoal(_ context.Context, userID, id int64) error {
	g, ok := f.goals[id]
	if !ok || g.UserID != userID {
		return storage.ErrGoalNotFound
	}
	delete(f.goals, id)
	return nil
}

func (f *goalsFakeRepo) SumContributions(_ context.Context, userID, goalID int64) (domain.Money, error) {
	sum := domain.Zero
	for _, c := range f.contributions {
		if c.UserID == userID && c.GoalID == goalID {
			sum = sum.Add(c.Amount)
		}
	}
	return sum, nil
}

func (f *goalsFakeRepo) CreateContribution(_ context.Context, c domain.GoalContribution) (domain.GoalContribution, error) {
	k := goalsClientKey(c.UserID, c.GoalID, c.ContributionID)
	if _, exists := f.contributions[k]; exists {
		return domain.GoalContribution{}, storage.ErrContributionExists
	}
	f.nextContribID++
	c.ID = f.nextContribID
	c.CreatedAt = time.Now()
	f.contributions[k] = c
	return c, nil
}

func (f *goalsFakeRepo) GetContributionByClientID(_ context.Context, userID, goalID int64, contributionID string) (domain.GoalContribution, error) {
	k := goalsClientKey(userID, goalID, contributionID)
	c, ok := f.contributions[k]
	if !ok {
		return domain.GoalContribution{}, storage.ErrContributionNotFound
	}
	return c, nil
}

func (f *goalsFakeRepo) ListContributions(_ context.Context, userID, goalID int64) ([]domain.GoalContribution, error) {
	var out []domain.GoalContribution
	for _, c := range f.contributions {
		if c.UserID == userID && c.GoalID == goalID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *goalsFakeRepo) DeleteContribution(_ context.Context, userID, goalID, id int64) error {
	for k, c := range f.contributions {
		if c.UserID == userID && c.GoalID == goalID && c.ID == id {
			delete(f.contributions, k)
			return nil
		}
	}
	return storage.ErrContributionNotFound
}

// ---- test env ----

// goalsTestEnv wires a real GoalsHandler with a goalsFakeRepo behind a chi
// router + a fake auth middleware that injects a fixed user_id (7) into
// context for every request. httptest.NewServer gives us a real base URL.
type goalsTestEnv struct {
	t      *testing.T
	srv    *httptest.Server
	repo   *goalsFakeRepo
	userID int64
}

func newGoalsTestEnv(t *testing.T) *goalsTestEnv {
	t.Helper()
	repo := newGoalsFakeRepo()
	svc := goals.NewService(repo)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	h := NewGoalsHandler(svc, logger)

	// Use chi so {id} and {cid} URL params resolve exactly as in production.
	r := chi.NewRouter()
	// Fake auth: inject a fixed user_id into context for every request.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), keyUserID, int64(7))
			ctx = context.WithValue(ctx, keyUserHash, "uhsh-goals")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h.Register(r)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &goalsTestEnv{t: t, srv: srv, repo: repo, userID: 7}
}

// goalReq performs an HTTP request with a JSON-encoded body (nil → no body)
// and returns the response + raw body bytes. Auth is implicit (test mw).
func goalReq(t *testing.T, method, url string, body any) (*http.Response, []byte) {
	t.Helper()
	var req *http.Request
	if body != nil {
		buf, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, url, bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, raw
}

// goalReqRaw is goalReq when the body should NOT be json.Marshal'd (e.g. for
// sending malformed JSON to exercise decodeJSON's error path).
func goalReqRaw(t *testing.T, method, url string, body []byte) (*http.Response, []byte) {
	t.Helper()
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, raw
}

// futureDate returns an RFC3339 timestamp ~2 years from now, suitable for
// target_date (ValidateGoal rejects past dates using the service's now()).
func futureDate() string {
	return time.Now().AddDate(2, 0, 0).Format(time.RFC3339)
}

// mustCreateGoal creates a goal via the HTTP API and returns its id. Used as
// a seed step by tests that need an existing goal (Get/Update/Projection…).
func mustCreateGoal(t *testing.T, env *goalsTestEnv, body map[string]any) int64 {
	t.Helper()
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("seed create: status=%d body=%s", resp.StatusCode, raw)
	}
	var g goalResp
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("seed create unmarshal: %v", err)
	}
	return g.ID
}

// ---- response shape structs ----

type goalResp struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	TargetAmount        string `json:"target_amount"`
	CurrentAmount       string `json:"current_amount"`
	MonthlyContribution string `json:"monthly_contribution"`
	TargetDate          string `json:"target_date"`
	ExpectedYield       string `json:"expected_yield"`
}

type contribResp struct {
	ID             int64  `json:"id"`
	GoalID         int64  `json:"goal_id"`
	ContributionID string `json:"contribution_id"`
	Amount         string `json:"amount"`
	Date           string `json:"date"`
	Comment        string `json:"comment"`
}

type errResp struct {
	Type   string `json:"type"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// ---- tests: CRUD on /goals ----

func TestGoalsHandler_Create_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals", map[string]any{
		"name":                 "Подушка безопасности",
		"target_amount":        "100000.00",
		"current_amount":       "5000.00",
		"monthly_contribution": "10000.00",
		"expected_yield":       "0.08",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var g goalResp
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if g.ID <= 0 {
		t.Errorf("id = %d, want > 0", g.ID)
	}
	if g.Name != "Подушка безопасности" {
		t.Errorf("name = %q", g.Name)
	}
	if g.TargetAmount != "100000.00" {
		t.Errorf("target_amount = %q, want 100000.00", g.TargetAmount)
	}
	if g.CurrentAmount != "5000.00" {
		t.Errorf("current_amount = %q, want 5000.00", g.CurrentAmount)
	}
	if g.MonthlyContribution != "10000.00" {
		t.Errorf("monthly_contribution = %q, want 10000.00", g.MonthlyContribution)
	}
	if g.ExpectedYield != "0.08" {
		t.Errorf("expected_yield = %q, want 0.08", g.ExpectedYield)
	}
}

func TestGoalsHandler_Create_DefaultsCurrentZero(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals", map[string]any{
		"name":          "Без взносов",
		"target_amount": "50000.00",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var g goalResp
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if g.CurrentAmount != "0.00" {
		t.Errorf("current_amount = %q, want 0.00 (default)", g.CurrentAmount)
	}
	if g.MonthlyContribution != "" {
		t.Errorf("monthly_contribution = %q, want empty (omitted)", g.MonthlyContribution)
	}
}

func TestGoalsHandler_Create_MalformedJSON(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReqRaw(t, http.MethodPost, env.srv.URL+"/goals", []byte(`{"name":"bad",`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
	var e errResp
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != "goals.invalid_body" {
		t.Errorf("type = %q, want goals.invalid_body", e.Type)
	}
}

// TestGoalsHandler_Create_NonPositiveTarget documents a pre-existing gap in
// writeServiceError: domain.ValidateGoal returns a bare errors.New(...) for a
// non-positive target, which does NOT satisfy errors.Is(err, goals.ErrInvalidArgument),
// so the handler's default branch surfaces it as 500 instead of 400. The test
// pins the current behavior so a future fix to writeServiceError (wrapping or
// matching ValidateGoal errors) will flip this expectation and force a review.
func TestGoalsHandler_Create_NonPositiveTarget(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals", map[string]any{
		"name":          "Zero",
		"target_amount": "0",
	})
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d (want 500, see comment), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Create_BadTargetAmount(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals", map[string]any{
		"name":           "X",
		"target_amount":  "not-a-number",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Create_BadTargetDate(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals", map[string]any{
		"name":          "X",
		"target_amount": "1000.00",
		"target_date":   "not-a-date",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

// TestGoalsHandler_Create_PastTargetDate documents the same pre-existing gap
// as TestGoalsHandler_Create_NonPositiveTarget: domain.ValidateGoal returns a
// bare error for a past target_date, which writeServiceError does not map to
// 400, so the handler surfaces it as 500. Pinned here so a future fix flips
// the expectation.
func TestGoalsHandler_Create_PastTargetDate(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals", map[string]any{
		"name":          "Past",
		"target_amount": "1000.00",
		"target_date":   "2000-01-01T00:00:00Z",
	})
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d (want 500, see comment), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Create_FutureTargetDate(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals", map[string]any{
		"name":          "Future",
		"target_amount": "1000.00",
		"target_date":   futureDate(),
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_List_Empty(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var out struct {
		Items []goalResp `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Items) != 0 {
		t.Errorf("items = %d, want 0", len(out.Items))
	}
}

func TestGoalsHandler_List_AfterCreate(t *testing.T) {
	env := newGoalsTestEnv(t)
	mustCreateGoal(t, env, map[string]any{
		"name": "A", "target_amount": "1000.00",
	})
	mustCreateGoal(t, env, map[string]any{
		"name": "B", "target_amount": "2000.00",
	})
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var out struct {
		Items []goalResp `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Items) != 2 {
		t.Errorf("items = %d, want 2", len(out.Items))
	}
}

func TestGoalsHandler_Get_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "Get", "target_amount": "9999.00", "current_amount": "10.00",
	})
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var g goalResp
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if g.ID != id {
		t.Errorf("id = %d, want %d", g.ID, id)
	}
	if g.TargetAmount != "9999.00" {
		t.Errorf("target_amount = %q", g.TargetAmount)
	}
}

func TestGoalsHandler_Get_NotFound(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals/9999", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d (want 404), body = %s", resp.StatusCode, raw)
	}
	var e errResp
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != "goals.not_found" {
		t.Errorf("type = %q, want goals.not_found", e.Type)
	}
}

func TestGoalsHandler_Get_InvalidID(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals/abc", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Get_NegativeID(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals/-5", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Update_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "Old", "target_amount": "1000.00", "current_amount": "0",
	})
	resp, raw := goalReq(t, http.MethodPatch, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10), map[string]any{
		"name":           "New",
		"target_amount":  "2000.00",
		"current_amount": "0",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var g goalResp
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if g.Name != "New" {
		t.Errorf("name = %q, want New", g.Name)
	}
	if g.TargetAmount != "2000.00" {
		t.Errorf("target_amount = %q, want 2000.00", g.TargetAmount)
	}
}

func TestGoalsHandler_Update_NotFound(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPatch, env.srv.URL+"/goals/9999", map[string]any{
		"name": "X", "target_amount": "1000.00", "current_amount": "0",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d (want 404), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Update_BadTargetAmount(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "X", "target_amount": "1000.00",
	})
	resp, _ := goalReq(t, http.MethodPatch, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10), map[string]any{
		"name": "X", "target_amount": "not-money", "current_amount": "0",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestGoalsHandler_Delete_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "Del", "target_amount": "1000.00",
	})
	resp, raw := goalReq(t, http.MethodDelete, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d (want 204), body = %s", resp.StatusCode, raw)
	}
	// Subsequent Get → 404.
	resp2, _ := goalReq(t, http.MethodGet, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10), nil)
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("after delete, GET got status = %d, want 404", resp2.StatusCode)
	}
}

func TestGoalsHandler_Delete_NotFound(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodDelete, env.srv.URL+"/goals/9999", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d (want 404), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Delete_InvalidID(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, _ := goalReq(t, http.MethodDelete, env.srv.URL+"/goals/abc", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// ---- tests: projection & simulation ----

func TestGoalsHandler_Projection_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "Proj", "target_amount": "100000.00", "current_amount": "0",
	})
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/projection", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var proj struct {
		Goal             goalResp `json:"goal"`
		EffectiveCurrent string   `json:"effective_current"`
		TargetEffective  string   `json:"target_effective"`
		Progress         string   `json:"progress"`
		Status           string   `json:"status"`
		AsOf             string   `json:"as_of"`
	}
	if err := json.Unmarshal(raw, &proj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if proj.Status == "" {
		t.Errorf("status empty; body = %s", raw)
	}
	if proj.Goal.ID != id {
		t.Errorf("goal.id = %d, want %d", proj.Goal.ID, id)
	}
	if proj.EffectiveCurrent != "0.00" {
		t.Errorf("effective_current = %q, want 0.00", proj.EffectiveCurrent)
	}
}

func TestGoalsHandler_Projection_NotFound(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals/9999/projection", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d (want 404), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Projection_InvalidID(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, _ := goalReq(t, http.MethodGet, env.srv.URL+"/goals/abc/projection", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestGoalsHandler_Projection_Achieved(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "Done", "target_amount": "1000.00", "current_amount": "1000.00",
	})
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/projection", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var proj struct {
		Status   string `json:"status"`
		Progress string `json:"progress"`
	}
	if err := json.Unmarshal(raw, &proj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if proj.Status != string(domain.StatusGoalAchieved) {
		t.Errorf("status = %q, want %q", proj.Status, domain.StatusGoalAchieved)
	}
}

func TestGoalsHandler_Simulate_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/calc/goal", map[string]any{
		"target_amount":  "100000.00",
		"current_amount": "0",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var proj struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &proj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if proj.Status != string(domain.StatusGoalNoDeadline) {
		t.Errorf("status = %q, want %q", proj.Status, domain.StatusGoalNoDeadline)
	}
}

func TestGoalsHandler_Simulate_MissingTarget(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/calc/goal", map[string]any{
		"current_amount": "0",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_Simulate_MalformedJSON(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, _ := goalReqRaw(t, http.MethodPost, env.srv.URL+"/calc/goal", []byte(`{"bad"`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestGoalsHandler_SimulateSaved_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "Sim", "target_amount": "100000.00", "current_amount": "0",
	})
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/simulate", map[string]any{
		"target_amount": "100000.00",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var proj struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &proj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if proj.Status == "" {
		t.Errorf("status empty; body = %s", raw)
	}
}

func TestGoalsHandler_SimulateSaved_NotFound(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals/9999/simulate", map[string]any{
		"target_amount": "100000.00",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d (want 404), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_SimulateSaved_InvalidID(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, _ := goalReq(t, http.MethodPost, env.srv.URL+"/goals/abc/simulate", map[string]any{
		"target_amount": "100000.00",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// ---- tests: contributions ----

func TestGoalsHandler_ListContributions_Empty(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	resp, raw := goalReq(t, http.MethodGet, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/contributions", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var out struct {
		Items []contribResp `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Items) != 0 {
		t.Errorf("items = %d, want 0", len(out.Items))
	}
}

func TestGoalsHandler_AddContribution_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/contributions", map[string]any{
		"contribution_id": "client-1",
		"amount":          "500.00",
		"comment":         "first",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var c contribResp
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.ID <= 0 {
		t.Errorf("id = %d, want > 0", c.ID)
	}
	if c.GoalID != id {
		t.Errorf("goal_id = %d, want %d", c.GoalID, id)
	}
	if c.ContributionID != "client-1" {
		t.Errorf("contribution_id = %q, want client-1", c.ContributionID)
	}
	if c.Amount != "500.00" {
		t.Errorf("amount = %q, want 500.00", c.Amount)
	}
	if c.Comment != "first" {
		t.Errorf("comment = %q, want first", c.Comment)
	}
	if c.Date == "" {
		t.Errorf("date empty (should be now())")
	}
}

func TestGoalsHandler_AddContribution_Duplicate_Idempotent(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	url := env.srv.URL + "/goals/" + strconv.FormatInt(id, 10) + "/contributions"
	body := map[string]any{
		"contribution_id": "dup-1",
		"amount":          "250.00",
	}
	// First POST → 201 Created.
	resp1, raw1 := goalReq(t, http.MethodPost, url, body)
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first post: status = %d, body = %s", resp1.StatusCode, raw1)
	}
	var c1 contribResp
	if err := json.Unmarshal(raw1, &c1); err != nil {
		t.Fatalf("first unmarshal: %v", err)
	}
	// Replay the same (user, goal, contribution_id) → 200 OK, same row.
	resp2, raw2 := goalReq(t, http.MethodPost, url, body)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("replay: status = %d (want 200), body = %s", resp2.StatusCode, raw2)
	}
	var c2 contribResp
	if err := json.Unmarshal(raw2, &c2); err != nil {
		t.Fatalf("replay unmarshal: %v", err)
	}
	if c2.ID != c1.ID {
		t.Errorf("replay id = %d, want %d (same row)", c2.ID, c1.ID)
	}
}

func TestGoalsHandler_AddContribution_MissingContributionID(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/contributions", map[string]any{
		"amount": "100.00",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_AddContribution_NonPositiveAmount(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/contributions", map[string]any{
		"contribution_id": "client-x",
		"amount":          "0",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_AddContribution_BadAmount(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/contributions", map[string]any{
		"contribution_id": "client-x",
		"amount":          "not-money",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_AddContribution_GoalNotFound(t *testing.T) {
	env := newGoalsTestEnv(t)
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals/9999/contributions", map[string]any{
		"contribution_id": "client-x",
		"amount":          "100.00",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d (want 404, service checks ownership), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_AddContribution_BadDate(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	resp, raw := goalReq(t, http.MethodPost, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/contributions", map[string]any{
		"contribution_id": "client-x",
		"amount":          "100.00",
		"date":            "not-a-date",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (want 400), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_AddContribution_MalformedJSON(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	resp, _ := goalReqRaw(t, http.MethodPost, env.srv.URL+"/goals/"+strconv.FormatInt(id, 10)+"/contributions", []byte(`{"bad"`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestGoalsHandler_ListContributions_AfterAdd(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	url := env.srv.URL + "/goals/" + strconv.FormatInt(id, 10) + "/contributions"
	_, _ = goalReq(t, http.MethodPost, url, map[string]any{
		"contribution_id": "a-1", "amount": "100.00",
	})
	_, _ = goalReq(t, http.MethodPost, url, map[string]any{
		"contribution_id": "a-2", "amount": "200.00",
	})
	resp, raw := goalReq(t, http.MethodGet, url, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}
	var out struct {
		Items []contribResp `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Items) != 2 {
		t.Errorf("items = %d, want 2", len(out.Items))
	}
}

func TestGoalsHandler_DeleteContribution_Success(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	addURL := env.srv.URL + "/goals/" + strconv.FormatInt(id, 10) + "/contributions"
	_, raw := goalReq(t, http.MethodPost, addURL, map[string]any{
		"contribution_id": "del-1", "amount": "300.00",
	})
	var c contribResp
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	delURL := addURL + "/" + strconv.FormatInt(c.ID, 10)
	resp, raw := goalReq(t, http.MethodDelete, delURL, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d (want 204), body = %s", resp.StatusCode, raw)
	}
	// Verify the contribution list is empty again.
	resp2, raw2 := goalReq(t, http.MethodGet, addURL, nil)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", resp2.StatusCode, raw2)
	}
	var out struct {
		Items []contribResp `json:"items"`
	}
	if err := json.Unmarshal(raw2, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Items) != 0 {
		t.Errorf("after delete, items = %d, want 0", len(out.Items))
	}
}

func TestGoalsHandler_DeleteContribution_NotFound(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	delURL := env.srv.URL + "/goals/" + strconv.FormatInt(id, 10) + "/contributions/9999"
	resp, raw := goalReq(t, http.MethodDelete, delURL, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d (want 404), body = %s", resp.StatusCode, raw)
	}
}

func TestGoalsHandler_DeleteContribution_InvalidCID(t *testing.T) {
	env := newGoalsTestEnv(t)
	id := mustCreateGoal(t, env, map[string]any{
		"name": "C", "target_amount": "1000.00",
	})
	delURL := env.srv.URL + "/goals/" + strconv.FormatInt(id, 10) + "/contributions/abc"
	resp, _ := goalReq(t, http.MethodDelete, delURL, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// ---- tests: auth ----

func TestGoalsHandler_Unauthorized_WithoutContext(t *testing.T) {
	// No auth middleware: MustUserID returns false → 401.
	repo := newGoalsFakeRepo()
	svc := goals.NewService(repo)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	h := NewGoalsHandler(svc, logger)
	r := chi.NewRouter()
	h.Register(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, _ := goalReq(t, http.MethodGet, srv.URL+"/goals", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestGoalsHandler_Unauthorized_Create_WithoutContext(t *testing.T) {
	repo := newGoalsFakeRepo()
	svc := goals.NewService(repo)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	h := NewGoalsHandler(svc, logger)
	r := chi.NewRouter()
	h.Register(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, _ := goalReq(t, http.MethodPost, srv.URL+"/goals", map[string]any{
		"name": "X", "target_amount": "1000.00",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}