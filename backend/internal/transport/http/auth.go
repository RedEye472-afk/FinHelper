package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RedEye472-afk/FinHelper/internal/auth"
	"github.com/RedEye472-afk/FinHelper/internal/email"
	applog "github.com/RedEye472-afk/FinHelper/internal/log"
	"github.com/RedEye472-afk/FinHelper/internal/service/categorization"
	"github.com/RedEye472-afk/FinHelper/internal/storage"
)

// AuthDeps bundles collaborators an AuthHandler needs.
type AuthDeps struct {
	Pool       *storage.Pool
	Issuer     *auth.JWTIssuer
	Salt       string // USER_HASH_SALT, mixed into user_hash
	Logger     *slog.Logger
	Mailer     *email.Sender
	FrontendURL string // URL for password reset links
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

type verifyEmailRequest struct {
	Code string `json:"code"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	Email    string `json:"email"` // required to look up the user
}

// Register → POST /api/v1/auth/register.
//
// Two-phase insert inside a transaction: user_hash depends on the assigned
// user_id, which only exists after INSERT. We insert with a unique
// placeholder ("tmp:" + email-hash) then UPDATE to the real id-derived hash.
// This keeps the UNIQUE(user_hash) constraint satisfiable without a separate
// "create then fixup" call path outside the tx.
//
// After registration, sends a 6-digit verification code to the user's email.
// The user must call POST /api/v1/auth/verify-email before they can fully log in.
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
		INSERT INTO users (email, password_hash, user_hash, verified)
		VALUES ($1, $2, $3, FALSE)
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

	// Seed default categories + keyword rules (same as before).
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

	// Generate and send verification code.
	code := generate6DigitCode()
	if err := h.deps.Pool.SetVerificationCode(ctx, id, code, time.Now().Add(10*time.Minute)); err != nil {
		h.deps.Logger.Error("register: set verification code", "user_hash", userHash, "error", err.Error())
		// Non-fatal — user can retry via send-code.
	}
	if h.deps.Mailer != nil {
		if err := h.deps.Mailer.SendVerificationCode(req.Email, code); err != nil {
			h.deps.Logger.Error("register: send verification email", "user_hash", userHash, "error", err.Error())
		}
	}

	ctx = applog.WithUserHash(ctx, userHash)
	applog.Info(ctx, h.deps.Logger, "user registered; verification code sent")
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":   "Verification code sent to email",
		"user_id":   id,
		"user_hash": userHash,
	})
}

// VerifyEmail → POST /api/v1/auth/verify-email.
//
// PUBLIC endpoint (no auth required). Finds the user by their 6-digit
// verification code, marks them as verified, and returns a token pair.
// Formula: lookup user WHERE verification_code = $1 AND verification_expires > NOW()
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req verifyEmailRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "auth.invalid_body", err.Error())
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "auth.invalid_code", "code is required")
		return
	}

	// Find user by the verification code (public — no auth required).
	user, err := h.deps.Pool.FindUserByVerificationCode(ctx, req.Code)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			writeError(w, http.StatusBadRequest, "auth.invalid_code", "invalid or expired verification code")
		} else {
			h.deps.Logger.Error("verify-email: lookup", "error", err.Error())
			writeError(w, http.StatusInternalServerError, "internal", "")
		}
		return
	}

	if err := h.deps.Pool.MarkUserVerified(ctx, user.ID); err != nil {
		h.deps.Logger.Error("verify-email: mark verified", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}

	// Issue tokens so the user is immediately logged in after verification.
	resp, err := h.issueTokens(ctx, user.ID, user.UserHash)
	if err != nil {
		h.deps.Logger.Error("verify-email: issue tokens", "user_hash", user.UserHash, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	applog.Info(ctx, h.deps.Logger, "email verified")
	writeJSON(w, http.StatusOK, resp)
}

// SendCode → POST /api/v1/auth/send-code.
//
// Public endpoint. Generates a new 6-digit verification code and emails it.
// Used for: resend code, or to trigger verification for existing unverified users.
// Returns empty 200 on success (no PII in response).
func (h *AuthHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "auth.invalid_body", err.Error())
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.deps.Pool.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Don't reveal whether the email exists.
		writeJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a code has been sent"})
		return
	}

	// Check if already verified.
	verified, err := h.deps.Pool.IsVerified(ctx, user.ID)
	if err == nil && verified {
		writeJSON(w, http.StatusOK, map[string]string{"message": "Email already verified"})
		return
	}

	code := generate6DigitCode()
	if err := h.deps.Pool.SetVerificationCode(ctx, user.ID, code, time.Now().Add(10*time.Minute)); err != nil {
		h.deps.Logger.Error("send-code: set code", "user_hash", user.UserHash, "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	if h.deps.Mailer != nil {
		if err := h.deps.Mailer.SendVerificationCode(req.Email, code); err != nil {
			h.deps.Logger.Error("send-code: send email", "user_hash", user.UserHash, "error", err.Error())
		}
	}

	applog.Info(ctx, h.deps.Logger, "verification code resent")
	writeJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a code has been sent"})
}

// Login → POST /api/v1/auth/login.
//
// Identical 401 for "no such email" and "wrong password" so the endpoint
// is not an email-existence oracle.
//
// If the user's email is not verified, sends a new verification code and
// returns requiresVerification=true instead of tokens.
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

	// Check if email is verified.
	verified, err := h.deps.Pool.IsVerified(ctx, u.ID)
	if err != nil {
		h.deps.Logger.Error("login: check verified", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	if !verified {
		// Send verification code.
		code := generate6DigitCode()
		if err := h.deps.Pool.SetVerificationCode(ctx, u.ID, code, time.Now().Add(10*time.Minute)); err != nil {
			h.deps.Logger.Error("login: set code", "user_hash", u.UserHash, "error", err.Error())
		}
		if h.deps.Mailer != nil {
			if err := h.deps.Mailer.SendVerificationCode(req.Email, code); err != nil {
				h.deps.Logger.Error("login: send code", "user_hash", u.UserHash, "error", err.Error())
			}
		}
		applog.Info(ctx, h.deps.Logger, "login blocked: email not verified")
		writeJSON(w, http.StatusOK, map[string]any{
			"requires_verification": true,
			"email":                 maskEmail(req.Email),
			"message":               "Email not verified. Verification code sent.",
		})
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

// ForgotPassword → POST /api/v1/auth/forgot-password.
//
// Generates a UUID reset token, stores its SHA-256 hash, and sends a
// password reset link to the user's email. Always returns 200 (don't reveal
// whether the email exists).
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req forgotPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "auth.invalid_body", err.Error())
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Always return 200 — don't reveal email existence.
	if req.Email == "" {
		writeJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a reset link has been sent"})
		return
	}

	// Generate UUID token.
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		h.deps.Logger.Error("forgot-password: generate token", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal", "")
		return
	}
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := auth.HashToken(rawToken)

	if err := h.deps.Pool.SetPasswordResetToken(ctx, req.Email, tokenHash, time.Now().Add(1*time.Hour)); err != nil {
		// User not found or DB error — still return 200 (security).
		h.deps.Logger.Warn("forgot-password: set token", "error", err.Error())
		writeJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a reset link has been sent"})
		return
	}

	frontendURL := h.deps.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, rawToken)

	if h.deps.Mailer != nil {
		if err := h.deps.Mailer.SendPasswordReset(req.Email, resetURL); err != nil {
			h.deps.Logger.Error("forgot-password: send email", "error", err.Error())
		}
	}

	applog.Info(ctx, h.deps.Logger, "password reset email sent")
	writeJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a reset link has been sent"})
}

// ResetPassword → POST /api/v1/auth/reset-password.
//
// Consumes the reset token and updates the user's password. All tokens are
// single-use: reusing a consumed or expired token returns an error.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req resetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "auth.invalid_body", err.Error())
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Token == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "auth.invalid_request", "token and email are required")
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

	tokenHash := auth.HashToken(req.Token)
	if err := h.deps.Pool.ConsumePasswordResetToken(ctx, req.Email, tokenHash, pwHash); err != nil {
		h.deps.Logger.Warn("reset-password: consume token", "error", err.Error())
		writeError(w, http.StatusBadRequest, "auth.invalid_token", "Invalid or expired reset token")
		return
	}

	// Revoke all refresh tokens so existing sessions must re-login.
	u, err := h.deps.Pool.GetUserByEmail(ctx, req.Email)
	if err == nil {
		if revokeErr := h.deps.Pool.RevokeAllRefreshTokens(ctx, u.ID); revokeErr != nil {
			h.deps.Logger.Error("reset-password: revoke tokens", "user_hash", u.UserHash, "error", revokeErr.Error())
		}
	}

	applog.Info(ctx, h.deps.Logger, "password reset successful")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
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

	// 2) DB-side single-use check.
	userID, err := h.deps.Pool.ConsumeRefreshToken(ctx, auth.HashToken(req.RefreshToken))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth.invalid_refresh", "token revoked or expired")
		return
	}
	if userID != claims.UserID {
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// issueTokens mints access+refresh and persists the refresh hash.
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

// generate6DigitCode returns a cryptographically random 6-digit number.
func generate6DigitCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		// Fallback: should never happen on a healthy system.
		return "123456"
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

// maskEmail returns "a***@domain.com" for logging/display.
func maskEmail(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return email
	}
	return email[:1] + "***" + email[at:]
}

// decodeJSON reads & decodes a JSON body with a strict size cap.
func decodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("unexpected trailing data in JSON body")
	}
	return nil
}

// rollbackUnlessCommitted rolls back unless the tx was already committed.
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
