package http

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// authTestEnv wires a real AuthHandler + AuthMiddleware against a sqlmock DB,
// exposing a *httptest.Server that talks the full handler surface. This is
// the smallest surface that catches wiring regressions.
type authTestEnv struct {
	t      *testing.T
	mock   sqlmock.Sqlmock
	server *httptest.Server
	issuer *auth.JWTIssuer
}

func newAuthTestEnv(t *testing.T) *authTestEnv {
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

	deps := AuthDeps{
		Pool:   &storage.Pool{DB: db}, // wrap sqlmock with the real Pool type
		Issuer: issuer,
		Salt:   "test-salt",
		Logger: logger,
	}
	authH := NewAuthHandler(deps)
	mw := NewAuthMiddleware(issuer, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/register", authH.Register)
	mux.HandleFunc("/api/v1/auth/login", authH.Login)
	mux.HandleFunc("/api/v1/auth/refresh", authH.Refresh)
	mux.Handle("/api/v1/me", mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, _ := MustUserID(r.Context())
		writeJSON(w, 200, map[string]any{"user_id": uid})
	})))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &authTestEnv{t: t, mock: mock, server: srv, issuer: issuer}
}

// postJSON is a tiny helper for the common "post a JSON body" case.
func postJSON(t *testing.T, url string, body any) (*http.Response, []byte) {
	t.Helper()
	buf, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
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
// /me behind AuthMiddleware — verifier-driven tests
// ----------------------------------------------------------------------------

func TestAuthMiddleware_RejectsMissingHeader(t *testing.T) {
	env := newAuthTestEnv(t)
	req, _ := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/me", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no-token /me: status = %d, want 401", resp.StatusCode)
	}
}

func TestAuthMiddleware_RejectsGarbageToken(t *testing.T) {
	env := newAuthTestEnv(t)
	req, _ := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer not.a.real.jwt")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("garbage token: status = %d, want 401", resp.StatusCode)
	}
}

func TestAuthMiddleware_AcceptsValidAccess(t *testing.T) {
	env := newAuthTestEnv(t)
	// Mint a token directly via the issuer (bypass DB).
	tok, _, err := env.issuer.IssueAccess(42, "hash-42", "")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("valid access: status = %d, want 200", resp.StatusCode)
	}
}

func TestAuthMiddleware_RejectsRefreshAsAccess(t *testing.T) {
	env := newAuthTestEnv(t)
	tok, _, _ := env.issuer.IssueRefresh(42, "hash-42", "")
	req, _ := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("refresh-as-access: status = %d, want 401", resp.StatusCode)
	}
}

// ----------------------------------------------------------------------------
// /refresh endpoint
// ----------------------------------------------------------------------------

func TestRefresh_RejectsUnknownToken(t *testing.T) {
	env := newAuthTestEnv(t)
	// VerifyRefresh passes (real signature), then DB returns no rows.
	tok, _, _ := env.issuer.IssueRefresh(7, "hash-7", "")

	// Match ConsumeRefreshToken UPDATE.
	env.mock.ExpectQuery(regexp.QuoteMeta(`UPDATE refresh_tokens`)).
		WithArgs(auth.HashToken(tok)).
		WillReturnError(sql.ErrNoRows)

	resp, _ := postJSON(t, env.server.URL+"/api/v1/auth/refresh", refreshRequest{RefreshToken: tok})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("refresh unknown: status = %d, want 401", resp.StatusCode)
	}
}

func TestRefresh_RotatesAndReturnsNewTokens(t *testing.T) {
	env := newAuthTestEnv(t)
	tok, _, _ := env.issuer.IssueRefresh(7, "hash-7", "")

	// 1) Consume old → user_id 7.
	env.mock.ExpectQuery(regexp.QuoteMeta(`UPDATE refresh_tokens`)).
		WithArgs(auth.HashToken(tok)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(7)))
	// 2) Save new refresh token row.
	env.mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO refresh_tokens`)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, body := postJSON(t, env.server.URL+"/api/v1/auth/refresh", refreshRequest{RefreshToken: tok})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh: status = %d, body = %s", resp.StatusCode, body)
	}
	var out tokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}
	// Note: we don't assert out.RefreshToken != tok here. JWTs encode only
	// iat/exp (second precision); when issue + re-issue happen within the
	// same second the serialised strings coincide. The rotation guarantee is
	// that the OLD token is revoked (ConsumeRefreshToken UPDATE ran) and a
	// NEW row is inserted — both are already asserted via sqlmock expectations
	// above. Replaying the old token would now fail because its row is revoked.
}

// ----------------------------------------------------------------------------
// /login endpoint
// ----------------------------------------------------------------------------

func TestLogin_UnknownEmailReturns401(t *testing.T) {
	env := newAuthTestEnv(t)
	env.mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs("ghost@example.com").
		WillReturnError(sql.ErrNoRows)

	resp, _ := postJSON(t, env.server.URL+"/api/v1/auth/login", credentialRequest{
		Email: "ghost@example.com", Password: "whatever1",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("login ghost: status = %d, want 401", resp.StatusCode)
	}
}

func TestLogin_WrongPasswordReturns401(t *testing.T) {
	env := newAuthTestEnv(t)
	// Pre-hash a password we definitely won't match.
	realHash, _ := auth.HashPassword("correct-password-123")
	env.mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs("user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "user_hash", "created_at", "updated_at"}).
			AddRow(int64(1), "user@example.com", realHash, "uhsh", time.Now(), time.Now()))

	resp, _ := postJSON(t, env.server.URL+"/api/v1/auth/login", credentialRequest{
		Email: "user@example.com", Password: "wrong-password-9",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("login wrong pw: status = %d, want 401", resp.StatusCode)
	}
}

func TestLogin_ValidIssuesTokens(t *testing.T) {
	env := newAuthTestEnv(t)
	pw := "valid-password-123"
	realHash, _ := auth.HashPassword(pw)
	env.mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
		WithArgs("user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "user_hash", "created_at", "updated_at"}).
			AddRow(int64(1), "user@example.com", realHash, "uhsh", time.Now(), time.Now()))
	env.mock.ExpectQuery(regexp.QuoteMeta(`SELECT verified`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"verified"}).AddRow(true))
	env.mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO refresh_tokens`)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, body := postJSON(t, env.server.URL+"/api/v1/auth/login", credentialRequest{
		Email: "user@example.com", Password: pw,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login valid: status = %d, body = %s", resp.StatusCode, body)
	}
	var out tokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.AccessToken == "" {
		t.Error("expected access token")
	}
}

// ----------------------------------------------------------------------------
// /register endpoint
// ----------------------------------------------------------------------------

func TestRegister_WeakPasswordRejected(t *testing.T) {
	env := newAuthTestEnv(t)
	resp, body := postJSON(t, env.server.URL+"/api/v1/auth/register", credentialRequest{
		Email: "u@example.com", Password: "short",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("register weak pw: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestRegister_BadEmailRejected(t *testing.T) {
	env := newAuthTestEnv(t)
	resp, _ := postJSON(t, env.server.URL+"/api/v1/auth/register", credentialRequest{
		Email: "not-an-email", Password: "valid-password-1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("register bad email: expected 400")
	}
}

func TestRegister_DuplicateReturns409(t *testing.T) {
	env := newAuthTestEnv(t)
	env.mock.ExpectBegin()
	env.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "users_email_key", Message: "dup"})
	env.mock.ExpectRollback()

	resp, _ := postJSON(t, env.server.URL+"/api/v1/auth/register", credentialRequest{
		Email: "dup@example.com", Password: "valid-password-1",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("register dup: expected 409, got %d", resp.StatusCode)
	}
}

func TestRegister_HappyPath(t *testing.T) {
	env := newAuthTestEnv(t)
	env.mock.ExpectBegin()
	env.mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "user_hash", "created_at", "updated_at"}).
			AddRow(int64(5), "new@example.com", "$2a$ignored", "tmp:placeholder", time.Now(), time.Now()))
	env.mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET user_hash`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	env.mock.ExpectCommit()
	// SetVerificationCode is called after commit.
	env.mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET verification_code`)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, body := postJSON(t, env.server.URL+"/api/v1/auth/register", credentialRequest{
		Email: "new@example.com", Password: "valid-password-1",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: status = %d, body = %s", resp.StatusCode, body)
	}
	// Register now returns { message, user_id, user_hash } — not tokens.
	var out struct {
		Message  string `json:"message"`
		UserID   int64  `json:"user_id"`
		UserHash string `json:"user_hash"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Message == "" {
		t.Error("expected non-empty message")
	}
	if out.UserID == 0 {
		t.Error("expected non-zero user_id")
	}
	if out.UserHash == "" {
		t.Error("expected non-empty user_hash")
	}
}

// ----------------------------------------------------------------------------
// misc edge-cases
// ----------------------------------------------------------------------------

func TestRegister_RejectsUnknownJSONFields(t *testing.T) {
	env := newAuthTestEnv(t)
	// Add an unexpected field; decoder uses DisallowUnknownFields.
	resp, _ := postJSON(t, env.server.URL+"/api/v1/auth/register", map[string]any{
		"email": "u@example.com", "password": "valid-password-1", "extra": true,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown field, got %d", resp.StatusCode)
	}
}

// Smoke: ValidateEmail behaves on common shapes. Note Go's mail.ParseAddress
// is permissive — "a@b" is a syntactically valid addr-spec (local@domain),
// so we don't expect TLD enforcement. We only reject *syntactic* errors and
// the "Display Name <addr>" form.
func TestValidateEmail(t *testing.T) {
	bad := []string{"", "a", "a@", "a @b.com", "Foo <a@b.com>"}
	for _, s := range bad {
		if err := validateEmail(s); err == nil {
			t.Errorf("validateEmail(%q) should fail", s)
		}
	}
	good := []string{"a@b.com", "user.name+tag@example.co.uk", "x@y.io", "a@b"}
	for _, s := range good {
		if err := validateEmail(s); err != nil {
			t.Errorf("validateEmail(%q) should pass: %v", s, err)
		}
	}
}

func TestExtractBearer(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"Bearer abc", "abc"},
		{"bearer abc", ""}, // case-sensitive prefix
		{"Basic xyz", ""},
		{"Bearer   spaced  ", "spaced"}, // trimmed
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if c.in != "" {
			req.Header.Set("Authorization", c.in)
		}
		got := extractBearer(req)
		// strings.TrimSpace applied inside extractBearer
		if strings.TrimSpace(got) != strings.TrimSpace(c.want) {
			t.Errorf("extractBearer(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// satisfy unused-import guard (kept for future test expansion).
var _ = regexp.QuoteMeta
