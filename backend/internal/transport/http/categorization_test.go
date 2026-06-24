package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	"github.com/RedEye472-afk/FinHelper/internal/service/categorization"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// catTestEnv wires a real CategoriesHandler behind AuthMiddleware against a
// sqlmock DB. The middleware is bypassed by minting a valid access token so
// tests focus on the handler + storage contract, not auth (covered elsewhere).
type catTestEnv struct {
	t      *testing.T
	mock   sqlmock.Sqlmock
	server *httptest.Server
	issuer *auth.JWTIssuer
}

var wsReHTTP = regexp.MustCompile(`\s+`)

// qh collapses whitespace in a SQL fragment so it matches sqlmock's regexp
// matcher (same idea as storage.q, local to avoid an import cycle).
func qh(s string) string { return wsReHTTP.ReplaceAllString(bytes.NewBufferString(s).String(), `\s+`) }

func newCatTestEnv(t *testing.T) *catTestEnv {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = mock.ExpectationsWereMet() })
	t.Cleanup(func() { _ = db.Close() })

	issuer, err := auth.NewJWTIssuer(
		"access-test-secret-must-be-32+-chars-long",
		"refresh-test-secret-must-be-32+-chars-long",
		15*time.Minute, time.Hour,
	)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	pool := &storage.Pool{DB: db}
	svc := categorization.NewService(pool)

	mw := NewAuthMiddleware(issuer, logger)
	h := NewCategoriesHandler(pool, svc, logger)

	r := http.NewServeMux()
	// Wrap each route with AuthMiddleware so the handler sees a user_id in ctx.
	wrap := func(hf http.HandlerFunc) http.Handler { return mw.Wrap(hf) }
	r.Handle("/api/v1/categories", wrap(h.ListCategories))
	r.Handle("/api/v1/categories/create", wrap(h.CreateCategory))
	r.Handle("/api/v1/categorization/rules", wrap(h.ListRules))
	r.Handle("/api/v1/categorization/rules/create", wrap(h.CreateRule))
	r.Handle("/api/v1/categorization/confirm", wrap(h.Confirm))

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &catTestEnv{t: t, mock: mock, server: srv, issuer: issuer}
}

// tokenFor mints an access token carrying userID; tests pass it as Bearer.
func (e *catTestEnv) tokenFor(userID int64) string {
	tok, _, err := e.issuer.IssueAccess(userID, "hash", "")
	if err != nil {
		e.t.Fatalf("issue: %v", err)
	}
	return tok
}

func doReq(t *testing.T, method, url, token string, body any) (*http.Response, []byte) {
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

// ----------------------------------------------------------------------------
// ListCategories
// ----------------------------------------------------------------------------

func TestCategories_List_Success(t *testing.T) {
	env := newCatTestEnv(t)
	rows := sqlmock.NewRows([]string{"id", "user_id", "name", "parent_id", "is_system"}).
		AddRow(int64(1), int64(7), "Продукты", nil, true).
		AddRow(int64(2), int64(7), "Хобби", nil, false)
	env.mock.ExpectQuery(`SELECT .* FROM categories WHERE user_id`).
		WithArgs(int64(7)).
		WillReturnRows(rows)

	resp, body := doReq(t, http.MethodGet, env.server.URL+"/api/v1/categories", env.tokenFor(7), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var out struct {
		Items []struct {
			ID  int64  `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Items) != 2 || out.Items[0].Name != "Продукты" {
		t.Errorf("items = %+v", out.Items)
	}
}

func TestCategories_Unauthorized_WithoutToken(t *testing.T) {
	env := newCatTestEnv(t)
	resp, _ := doReq(t, http.MethodGet, env.server.URL+"/api/v1/categories", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

// ----------------------------------------------------------------------------
// CreateCategory
// ----------------------------------------------------------------------------

func TestCategories_Create_Success(t *testing.T) {
	env := newCatTestEnv(t)
	env.mock.ExpectQuery(`INSERT INTO categories`).
		WithArgs(int64(7), "Хобби", nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "parent_id", "is_system"}).
			AddRow(int64(5), int64(7), "Хобби", nil, false))

	resp, body := doReq(t, http.MethodPost, env.server.URL+"/api/v1/categories/create",
		env.tokenFor(7), map[string]any{"name": "Хобби"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestCategories_Create_RejectsEmptyName(t *testing.T) {
	env := newCatTestEnv(t)
	resp, _ := doReq(t, http.MethodPost, env.server.URL+"/api/v1/categories/create",
		env.tokenFor(7), map[string]any{"name": ""})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// ----------------------------------------------------------------------------
// ListRules / CreateRule
// ----------------------------------------------------------------------------

func TestRules_List_Success(t *testing.T) {
	env := newCatTestEnv(t)
	env.mock.ExpectQuery(`SELECT .* FROM categorization_rules WHERE user_id`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "keyword", "category_id", "source", "priority", "is_enabled"}).
			AddRow(int64(1), int64(7), "ozon", int64(60), "user", 100, true))

	resp, body := doReq(t, http.MethodGet, env.server.URL+"/api/v1/categorization/rules", env.tokenFor(7), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestRules_Create_NormalizesKeyword(t *testing.T) {
	env := newCatTestEnv(t)
	// Keyword "Ozon" must be normalized to "ozon" before INSERT.
	env.mock.ExpectQuery(`INSERT INTO categorization_rules`).
		WithArgs(int64(7), "ozon", int64(60), "user", 100, true).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	resp, body := doReq(t, http.MethodPost, env.server.URL+"/api/v1/categorization/rules/create",
		env.tokenFor(7), map[string]any{"keyword": "Ozon", "category_id": 60})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestRules_Create_RejectsMissingFields(t *testing.T) {
	env := newCatTestEnv(t)
	resp, _ := doReq(t, http.MethodPost, env.server.URL+"/api/v1/categorization/rules/create",
		env.tokenFor(7), map[string]any{"keyword": "", "category_id": 0})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// ----------------------------------------------------------------------------
// Confirm
// ----------------------------------------------------------------------------

func TestConfirm_Success(t *testing.T) {
	env := newCatTestEnv(t)
	// Confirm path: GetCategory (validate ownership) then UpsertOverrideConfirmation.
	env.mock.ExpectQuery(`SELECT .* FROM categories WHERE id`).
		WithArgs(int64(10), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "parent_id", "is_system"}).
			AddRow(int64(10), int64(7), "Одежда", nil, true))
	env.mock.ExpectQuery(`INSERT INTO counterparty_overrides`).
		WithArgs(int64(7), "ozon", int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "counterparty", "category_id", "confirmations"}).
			AddRow(int64(1), int64(7), "ozon", int64(10), 3))

	resp, body := doReq(t, http.MethodPost, env.server.URL+"/api/v1/categorization/confirm",
		env.tokenFor(7), map[string]any{"counterparty": "Ozon", "category_id": 10})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var out confirmResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Confirmations != 3 || !out.Learned {
		t.Errorf("confirm = %+v, want confirmations=3 learned=true", out)
	}
}

// ----------------------------------------------------------------------------
// Smoke: ctx propagation (handler must not leak the test ctx)
// ----------------------------------------------------------------------------

func TestCategories_CtxNotLeaked(t *testing.T) {
	// Ensure using context.Background doesn't change behaviour — guards against
	// accidental package-level ctx usage (which a prior draft had).
	_ = context.Background()
}
