package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	applog "github.com/RedEye472-afk/FinHelper/internal/log"
	"github.com/RedEye472-afk/FinHelper/internal/service/categorization"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// AuthDeps bundles collaborators an AuthHandler needs.
type AuthDeps struct {
	Pool   *storage.Pool
	Issuer *auth.JWTIssuer
	Salt   string // USER_HASH_SALT, mixed into user_hash
	Logger *slog.Logger
}

// AuthHandler wires register/login/refresh HTTP endpoints.
type AuthHandler struct {
	deps AuthDeps
}

// NewAuthHandler rejects nil deps at boot so a misconfigured main() fails
// loudly instead of panicking mid-request.
func NewAuthHandler(deps AuthDeps) *AuthHandler {
	if deps.Pool == nil || deps.Issuer == nil || deps.Salt == "" || deps.Logger == nil {
		panic("http: NewAuthHandler requires all deps non-nil/non-empty")
	}
	return &AuthHandler{deps: deps}
}

// tokenResponse is the JSON body returned on register/login/refresh.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // access token seconds
}

type credentialRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register → POST /api/v1/auth/register.
//
// Two-phase insert inside a transaction: user_hash depends on the assigned
// user_id, which only exists after INSERT. We insert with a unique
// placeholder ("tmp:" + email-hash) then UPDATE to the real id-derived hash.
// This keeps the UNIQUE(user_hash) constraint satisfiable without a separate
// "create then fixup" call path outside the tx.
//
// PRIVACY_RULES.md §1: no email in logs, only user_hash.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req credentialRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "auth.invalid_body", err.Error())
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if err := validateEmail(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, "auth.invalid_email", err.Error())
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, "auth.weak_password", err.Error())
		return
	}

	pwHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "auth.weak_password", err.Error())
		return
	}

	tx, err := h.deps.Pool.DB.BeginTx(ctx, nil)
	if err != nil {
		h.deps.Logger.Error("register: begin tx", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	defer rollbackUnlessCommitted(tx) // noop if Commit ran

	placeholder := "tmp:" + auth.HashEmail(req.Email, h.deps.Salt)
	var (
		id           int64
		createdAt    string
		updatedAt    string
		storedEmail  string
		passwordHash string
	)
	const insertQ = `
		INSERT INTO users (email, password_hash, user_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, user_hash, created_at, updated_at
	`
	var placeholderOut string
	err = tx.QueryRowContext(ctx, insertQ, req.Email, pwHash, placeholder).Scan(
		&id, &storedEmail, &passwordHash, &placeholderOut, &createdAt, &updatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "auth.user_exists", "email already registered")
			return
		}
		h.deps.Logger.Error("register: insert user", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}

	userHash := auth.UserHash(id, h.deps.Salt)
	const updateHashQ = `UPDATE users SET user_hash = $1 WHERE id = $2`
	if _, err := tx.ExecContext(ctx, updateHashQ, userHash, id); err != nil {
		h.deps.Logger.Error("register: set user_hash", "user_hash", userHash, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}

	// Seed default categories + keyword rules for ф.2 (auto-categorization),
	// inside the same tx so a new user is immediately categorizable. Both
	// helpers are idempotent (ON CONFLICT DO NOTHING), so a retry after a
	// partial failure won't double-insert. Seed failures are logged but do
	// NOT abort registration — a user without categories is degraded but
	// functional; blocking signup on a seed issue would be worse.
	if err := h.deps.Pool.SeedSystemCategories(ctx, tx, id, categorization.SystemCategories); err != nil {
		h.deps.Logger.Error("register: seed categories", "user_hash", userHash, "error", err.Error())
	}
	ruleSeeds := make([]storage.KeywordRuleSeed, 0, len(categorization.SystemKeywordRules))
	for _, r := range categorization.SystemKeywordRules {
		ruleSeeds = append(ruleSeeds, storage.KeywordRuleSeed{Keyword: r.Keyword, CategoryName: r.Category})
	}
	if err := h.deps.Pool.SeedDefaultsForUser(ctx, tx, id, ruleSeeds); err != nil {
		h.deps.Logger.Error("register: seed rules", "user_hash", userHash, "error", err.Error())
	}

	if err := tx.Commit(); err != nil {
		h.deps.Logger.Error("register: commit", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}

	ctx = applog.WithUserHash(ctx, userHash)
	resp, err := h.issueTokens(ctx, id, userHash)
	if err != nil {
		// User exists; tell client to log in.
		h.deps.Logger.Error("register: issue tokens", "user_hash", userHash, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "user created; please log in")
		return
	}
	applog.Info(ctx, h.deps.Logger, "user registered")
	writeJSON(w, http.StatusCreated, resp)
}

// Login → POST /api/v1/auth/login.
//
// Identical 401 for "no such email" and "wrong password" so the endpoint
// is not an email-existence oracle.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req credentialRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "auth.invalid_body", err.Error())
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	u, err := h.deps.Pool.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Don't distinguish not-found from storage error to the client.
		if !errors.Is(err, storage.ErrUserNotFound) {
			h.deps.Logger.Error("login: lookup", "error", err.Error())
		}
		writeError(w, http.StatusUnauthorized, "auth.invalid_credentials", "invalid email or password")
		return
	}

	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "auth.invalid_credentials", "invalid email or password")
		return
	}

	ctx = applog.WithUserHash(ctx, u.UserHash)
	resp, err := h.issueTokens(ctx, u.ID, u.UserHash)
	if err != nil {
		h.deps.Logger.Error("login: issue tokens", "user_hash", u.UserHash, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.deps.Logger, "user logged in")
	writeJSON(w, http.StatusOK, resp)
}

// Refresh → POST /api/v1/auth/refresh.
//
// Rotation policy (Plane from GLM.md, Задача 2): refresh is single-use.
// ConsumeRefreshToken atomically revokes the row; a replayed token returns
// 401 because the row is already revoked.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "auth.invalid_body", "refresh_token required")
		return
	}

	// 1) Cheap JWT signature/kind/expiry reject first.
	claims, err := h.deps.Issuer.VerifyRefresh(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth.invalid_refresh", "token invalid or expired")
		return
	}

	// 2) DB-side single-use check. Even a correctly signed token must be
	// marked active; this is what makes logout/rotation effective.
	userID, err := h.deps.Pool.ConsumeRefreshToken(ctx, auth.HashToken(req.RefreshToken))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth.invalid_refresh", "token revoked or expired")
		return
	}
	if userID != claims.UserID { // defense-in-depth
		h.deps.Logger.Error("refresh: user_id mismatch", "user_hash", claims.UserHash)
		writeError(w, http.StatusUnauthorized, "auth.invalid_refresh", "token mismatch")
		return
	}

	ctx = applog.WithUserHash(ctx, claims.UserHash)
	resp, err := h.issueTokens(ctx, claims.UserID, claims.UserHash)
	if err != nil {
		h.deps.Logger.Error("refresh: issue tokens", "user_hash", claims.UserHash, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.deps.Logger, "tokens refreshed")
	writeJSON(w, http.StatusOK, resp)
}

// issueTokens mints access+refresh and persists the refresh hash. Shared by
// register / login / refresh.
func (h *AuthHandler) issueTokens(ctx context.Context, userID int64, userHash string) (tokenResponse, error) {
	accessTok, _, err := h.deps.Issuer.IssueAccess(userID, userHash, "")
	if err != nil {
		return tokenResponse{}, err
	}
	refreshTok, refreshExp, err := h.deps.Issuer.IssueRefresh(userID, userHash, "")
	if err != nil {
		return tokenResponse{}, err
	}
	if err := h.deps.Pool.SaveRefreshToken(ctx, userID, auth.HashToken(refreshTok), refreshExp); err != nil {
		return tokenResponse{}, err
	}
	return tokenResponse{
		AccessToken:  accessTok,
		RefreshToken: refreshTok,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.deps.Issuer.AccessTTL().Seconds()),
	}, nil
}

// decodeJSON reads & decodes a JSON body with a strict size cap. Returns a
// plain error whose message is safe to surface as the "detail" field.
func decodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	// Reject trailing junk after the single JSON object.
	if dec.More() {
		return errors.New("unexpected trailing data in JSON body")
	}
	return nil
}

// rollbackUnlessCommitted rolls back unless the tx was already committed.
// Calling Rollback on a committed tx returns sql.ErrTxDone which we ignore.
func rollbackUnlessCommitted(tx interface {
	Rollback() error
}) {
	_ = tx.Rollback()
}

// isUniqueViolation reports whether err is pg 23505 (unique_violation).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
