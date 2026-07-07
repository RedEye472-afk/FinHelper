package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// User is the persisted user record. We expose it through storage (not a
// separate domain type) because at this stage it only mirrors the DB row and
// has no invariants worth a dedicated domain package. When richer behavior
// emerges, promote to domain.User.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	UserHash     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Sentinel errors. Callers use errors.Is to map them to HTTP responses
// without type assertions.
var (
	// ErrUserExists is returned by CreateUser when the email is already taken.
	ErrUserExists = errors.New("storage: user already exists")
	// ErrUserNotFound is returned by GetUserByEmail when no row matches.
	ErrUserNotFound = errors.New("storage: user not found")
	// ErrRefreshTokenNotFound is returned when a refresh token row does not
	// exist, is already revoked, or is expired.
	ErrRefreshTokenNotFound = errors.New("storage: refresh token not found")
)

// CreateUser inserts a new user. userHash must already be computed by the
// caller (auth.UserHash) — storage is intentionally agnostic of the salt.
//
// On a duplicate email we translate the underlying pg 23505 (unique_violation)
// to ErrUserExists so handlers can return 409 without importing pgconn.
func (p *Pool) CreateUser(ctx context.Context, email, passwordHash, userHash string) (User, error) {
	const q = `
		INSERT INTO users (email, password_hash, user_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, user_hash, created_at, updated_at
	`
	var u User
	err := p.DB.QueryRowContext(ctx, q, email, passwordHash, userHash).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.UserHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return User{}, translatePgError(err, "email", ErrUserExists)
	}
	return u, nil
}

// GetUserByEmail returns the user with the given email, or ErrUserNotFound.
// We do NOT filter by deleted_at IS NULL here — a soft-deleted user still
// owns resources that may need reconciliation. The service layer decides.
func (p *Pool) GetUserByEmail(ctx context.Context, email string) (User, error) {
	const q = `
		SELECT id, email, password_hash, user_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var u User
	err := p.DB.QueryRowContext(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.UserHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("storage: get user by email: %w", err)
	}
	return u, nil
}

// GetUserByID returns the user with the given id, or ErrUserNotFound.
// Used by AuthMiddleware when reconstructing identity from a JWT.
func (p *Pool) GetUserByID(ctx context.Context, id int64) (User, error) {
	const q = `
		SELECT id, email, password_hash, user_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u User
	err := p.DB.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.UserHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("storage: get user by id: %w", err)
	}
	return u, nil
}

// SaveRefreshToken records a refresh token (already hashed by the caller) for
// the given user. Inserting rather than upserting preserves history for audit
// and lets us detect token reuse (a stolen-then-revoked token reappearing).
func (p *Pool) SaveRefreshToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := p.DB.ExecContext(ctx, q, userID, tokenHash, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("storage: save refresh token: %w", err)
	}
	return nil
}

// ConsumeRefreshToken atomically marks a refresh token as revoked and returns
// the associated user_id. Returns ErrRefreshTokenNotFound if the token does
// not exist, is already revoked, or has expired — all three cases are
// indistinguishable to the caller and all must result in 401.
//
// The single UPDATE … RETURNING guarantees atomicity: two concurrent refresh
// requests with the same token cannot both succeed (one will see revoked_at
// already set and get zero rows).
func (p *Pool) ConsumeRefreshToken(ctx context.Context, tokenHash string) (userID int64, err error) {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		RETURNING user_id
	`
	err = p.DB.QueryRowContext(ctx, q, tokenHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrRefreshTokenNotFound
		}
		return 0, fmt.Errorf("storage: consume refresh token: %w", err)
	}
	return userID, nil
}

// SetVerificationCode stores a 6-digit verification code with a 10-minute
// expiry for the given user. Returns ErrUserNotFound if the user does not exist.
func (p *Pool) SetVerificationCode(ctx context.Context, userID int64, code string, expiresAt time.Time) error {
	const q = `UPDATE users SET verification_code = $1, verification_expires = $2 WHERE id = $3`
	res, err := p.DB.ExecContext(ctx, q, code, expiresAt.UTC(), userID)
	if err != nil {
		return fmt.Errorf("storage: set verification code: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// VerifyEmail checks the 6-digit code for a user. If the code matches and is
// not expired, sets verified = TRUE and clears the code fields.
// Returns ErrUserNotFound if no user with that id exists, or an error message
// if the code is wrong/expired.
func (p *Pool) VerifyEmail(ctx context.Context, userID int64, code string) error {
	const q = `SELECT verification_code, verification_expires FROM users WHERE id = $1`
	var storedCode *string
	var expiresAt *time.Time
	if err := p.DB.QueryRowContext(ctx, q, userID).Scan(&storedCode, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("storage: verify email lookup: %w", err)
	}
	if storedCode == nil || *storedCode == "" {
		return fmt.Errorf("no verification code set")
	}
	if *storedCode != code {
		return fmt.Errorf("invalid verification code")
	}
	if expiresAt == nil || time.Now().After(*expiresAt) {
		return fmt.Errorf("verification code expired")
	}
	const updateQ = `UPDATE users SET verified = TRUE, verification_code = NULL, verification_expires = NULL WHERE id = $1`
	_, err := p.DB.ExecContext(ctx, updateQ, userID)
	if err != nil {
		return fmt.Errorf("storage: verify email update: %w", err)
	}
	return nil
}

// FindUserByVerificationCode finds a user by their 6-digit verification code.
// Returns the user if found AND the code has not expired. Returns ErrUserNotFound
// if no matching unexpired code exists.
// This is used for the public verify-email endpoint (no auth required).
func (p *Pool) FindUserByVerificationCode(ctx context.Context, code string) (User, error) {
	const q = `
		SELECT id, email, password_hash, user_hash, created_at, updated_at
		FROM users
		WHERE verification_code = $1
		  AND verification_expires > NOW()
	`
	var u User
	err := p.DB.QueryRowContext(ctx, q, code).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.UserHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("storage: find user by verification code: %w", err)
	}
	return u, nil
}

// MarkUserVerified sets verified=true and clears verification_code/expires for the given user.
func (p *Pool) MarkUserVerified(ctx context.Context, userID int64) error {
	const q = `UPDATE users SET verified = TRUE, verification_code = NULL, verification_expires = NULL WHERE id = $1`
	res, err := p.DB.ExecContext(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("storage: mark user verified: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// IsVerified returns whether a user has verified their email.
func (p *Pool) IsVerified(ctx context.Context, userID int64) (bool, error) {
	const q = `SELECT verified FROM users WHERE id = $1`
	var verified bool
	err := p.DB.QueryRowContext(ctx, q, userID).Scan(&verified)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrUserNotFound
		}
		return false, fmt.Errorf("storage: is verified: %w", err)
	}
	return verified, nil
}

// SetPasswordResetToken stores a UUID token (SHA-256 hashed by caller) for
// password reset, valid for 1 hour.
func (p *Pool) SetPasswordResetToken(ctx context.Context, email, tokenHash string, expiresAt time.Time) error {
	const q = `UPDATE users SET password_reset_token = $1, password_reset_expires = $2 WHERE email = $3`
	res, err := p.DB.ExecContext(ctx, q, tokenHash, expiresAt.UTC(), email)
	if err != nil {
		return fmt.Errorf("storage: set password reset token: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// ConsumePasswordResetToken checks the token for a user. If valid, atomically
// updates the password and clears the token fields. Returns ErrUserNotFound
// if the email does not exist, or an error if the token is wrong/expired.
func (p *Pool) ConsumePasswordResetToken(ctx context.Context, email, tokenHash, newPasswordHash string) error {
	const q = `SELECT password_reset_token, password_reset_expires FROM users WHERE email = $1`
	var storedToken *string
	var expiresAt *time.Time
	if err := p.DB.QueryRowContext(ctx, q, email).Scan(&storedToken, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("storage: consume password reset lookup: %w", err)
	}
	if storedToken == nil || *storedToken == "" {
		return fmt.Errorf("no password reset token set")
	}
	if *storedToken != tokenHash {
		return fmt.Errorf("invalid password reset token")
	}
	if expiresAt == nil || time.Now().After(*expiresAt) {
		return fmt.Errorf("password reset token expired")
	}
	const updateQ = `UPDATE users SET password_hash = $1, password_reset_token = NULL, password_reset_expires = NULL WHERE email = $2`
	_, err := p.DB.ExecContext(ctx, updateQ, newPasswordHash, email)
	if err != nil {
		return fmt.Errorf("storage: consume password reset update: %w", err)
	}
	return nil
}

// RevokeAllRefreshTokens marks every active refresh token for a user as
// revoked. Called on logout-all and password change.
func (p *Pool) RevokeAllRefreshTokens(ctx context.Context, userID int64) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := p.DB.ExecContext(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("storage: revoke all refresh tokens: %w", err)
	}
	return nil
}
