package storage

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
)

// newMockPool builds a Pool whose *sql.DB is backed by sqlmock.
// The t.Cleanup closes the mock db automatically.
func newMockPool(t *testing.T) (*Pool, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	// Check expectations BEFORE closing: sqlmock's Close also walks the
	// expectation queue and would otherwise emit a misleading "Close was not
	// expected" message on already-fulfilled runs.
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled sqlmock expectations: %v", err)
		}
	})
	t.Cleanup(func() {
		_ = db.Close()
	})
	return &Pool{DB: db}, mock
}

// q collapses whitespace so the regex matcher is forgiving of formatting.
var wsRe = regexp.MustCompile(`\s+`)

func q(s string) string {
	return wsRe.ReplaceAllString(strings.TrimSpace(s), `\s+`)
}

// ----------------------------------------------------------------------------
// CreateUser
// ----------------------------------------------------------------------------

func TestCreateUser_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "user_hash", "created_at", "updated_at"}).
		AddRow(int64(1), "u@example.com", "$2a$hash", "uhsh", now, now)

	mock.ExpectQuery(q(`INSERT INTO users`)).
		WithArgs("u@example.com", "$2a$hash", "uhsh").
		WillReturnRows(rows)

	u, err := pool.CreateUser(ctx, "u@example.com", "$2a$hash", "uhsh")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID != 1 || u.Email != "u@example.com" || u.UserHash != "uhsh" {
		t.Errorf("unexpected user: %+v", u)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	// Simulate pg 23505 unique_violation on users_email_key.
	mock.ExpectQuery(q(`INSERT INTO users`)).
		WithArgs("dup@example.com", "h", "uh").
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "users_email_key", Message: "duplicate"})

	_, err := pool.CreateUser(ctx, "dup@example.com", "h", "uh")
	if !errors.Is(err, ErrUserExists) {
		t.Errorf("expected ErrUserExists, got %v", err)
	}
}

func TestCreateUser_OtherPgError(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`INSERT INTO users`)).
		WithArgs("u@e.com", "h", "u").
		WillReturnError(&pgconn.PgError{Code: "23502", Message: "not-null violation"})

	_, err := pool.CreateUser(ctx, "u@e.com", "h", "u")
	if errors.Is(err, ErrUserExists) {
		t.Errorf("must NOT map 23502 to ErrUserExists")
	}
	if err == nil {
		t.Error("expected non-nil error")
	}
}

// ----------------------------------------------------------------------------
// GetUserByEmail
// ----------------------------------------------------------------------------

func TestGetUserByEmail_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "user_hash", "created_at", "updated_at"}).
		AddRow(int64(7), "u@example.com", "$2a$hash", "uhsh", now, now)

	mock.ExpectQuery(q(`SELECT .* FROM users WHERE email =`)).
		WithArgs("u@example.com").
		WillReturnRows(rows)

	u, err := pool.GetUserByEmail(ctx, "u@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if u.ID != 7 {
		t.Errorf("id = %d", u.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT .* FROM users WHERE email =`)).
		WithArgs("ghost@example.com").
		WillReturnError(sql.ErrNoRows)

	_, err := pool.GetUserByEmail(ctx, "ghost@example.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`SELECT .* FROM users WHERE id =`)).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err := pool.GetUserByID(ctx, 999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// SaveRefreshToken / ConsumeRefreshToken / RevokeAllRefreshTokens
// ----------------------------------------------------------------------------

func TestSaveRefreshToken_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()
	exp := time.Now().Add(720 * time.Hour).UTC()

	mock.ExpectExec(q(`INSERT INTO refresh_tokens`)).
		WithArgs(int64(1), "tokhash", exp).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := pool.SaveRefreshToken(ctx, 1, "tokhash", exp); err != nil {
		t.Fatalf("SaveRefreshToken: %v", err)
	}
}

func TestConsumeRefreshToken_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectQuery(q(`UPDATE refresh_tokens SET revoked_at = NOW\(\)`)).
		WithArgs("tokhash").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(42)))

	uid, err := pool.ConsumeRefreshToken(ctx, "tokhash")
	if err != nil {
		t.Fatalf("ConsumeRefreshToken: %v", err)
	}
	if uid != 42 {
		t.Errorf("uid = %d", uid)
	}
}

func TestConsumeRefreshToken_AlreadyRevokedOrExpired(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	// Zero rows → already revoked, expired, or never existed.
	mock.ExpectQuery(q(`UPDATE refresh_tokens SET revoked_at = NOW\(\)`)).
		WithArgs("bogus").
		WillReturnError(sql.ErrNoRows)

	_, err := pool.ConsumeRefreshToken(ctx, "bogus")
	if !errors.Is(err, ErrRefreshTokenNotFound) {
		t.Errorf("expected ErrRefreshTokenNotFound, got %v", err)
	}
}

func TestRevokeAllRefreshTokens_Success(t *testing.T) {
	pool, mock := newMockPool(t)
	ctx := context.Background()

	mock.ExpectExec(q(`UPDATE refresh_tokens SET revoked_at = NOW\(\) WHERE user_id`)).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 3))

	if err := pool.RevokeAllRefreshTokens(ctx, 7); err != nil {
		t.Fatalf("RevokeAllRefreshTokens: %v", err)
	}
}
