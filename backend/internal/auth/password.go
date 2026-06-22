package auth

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLength is the minimum acceptable password length.
// Matches Plane from GLM.md → Задача 2: "Password policy: ≥ 8 символов".
const MinPasswordLength = 8

// MaxPasswordLength caps input size to bound bcrypt's 72-byte limit and
// protect against denial-of-service via very long inputs.
const MaxPasswordLength = 72

// ErrPasswordTooShort is returned when a password is shorter than MinPasswordLength.
var ErrPasswordTooShort = errors.New("auth: password must be at least 8 characters")

// ErrPasswordTooLong is returned when a password exceeds MaxPasswordLength.
var ErrPasswordTooLong = errors.New("auth: password must be at most 72 characters")

// ValidatePassword enforces the password policy before hashing.
// Policy (Plane from GLM.md, Задача 2): length only, ≥ 8 chars.
// We deliberately avoid "must contain digit/symbol" rules — NIST 800-63B
// recommends length + checking against breach corpora over composition rules.
func ValidatePassword(pw string) error {
	n := utf8.RuneCountInString(pw)
	if n < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if n > MaxPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}

// HashPassword bcrypts the plaintext password at cost 12.
// Cost 12 ≈ ~250 ms on a modern CPU — a deliberate slowdown that makes
// offline brute-force expensive without making login unusable.
//
// Caller MUST have run ValidatePassword first; we still defend in depth by
// rejecting >72-byte inputs (bcrypt's hard limit) with a wrapped error.
func HashPassword(pw string) (string, error) {
	if err := ValidatePassword(pw); err != nil {
		return "", err
	}
	// bcrypt.MaxCost is 31; 12 is our policy. Using DefaultCost would silently
	// change over time as x/crypto bumps it, so we pin the cost explicitly.
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword reports whether the plaintext matches the stored bcrypt hash.
// Returns true on match, false on any mismatch (wrong password, malformed hash).
// We never surface the underlying bcrypt error to the caller: a malformed hash
// and a wrong password are both "not authenticated" from the caller's POV;
// distinguishing them would leak information and complicates handlers.
func CheckPassword(hash, pw string) bool {
	if hash == "" || pw == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
