package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// UserHash returns the anonymous identifier used in logs and analytics
// instead of PII (PRIVACY_RULES.md §1, CLAUDE.md principle 4).
//
// Construction: user_hash = SHA-256( decimal(user_id) || ":" || USER_HASH_SALT )
//
// We use ":" as a separator to prevent ambiguity — without it, user_id=12
// with salt "3x" would collide with user_id=1 with salt "23x".
//
// user_id is incorporated (not email) because email can change, and we want
// a user's hash to remain stable across email edits within the same account.
func UserHash(userID int64, salt string) string {
	if salt == "" {
		// This is a programming error (config layer should enforce non-empty
		// salt). Fail loudly rather than silently produce an unsalted hash.
		panic("auth: UserHash called with empty salt")
	}
	msg := strconv.FormatInt(userID, 10) + ":" + salt
	sum := sha256.Sum256([]byte(msg))
	return hex.EncodeToString(sum[:])
}

// HashToken returns SHA-256(token) in hex. We store only the hash of refresh
// tokens in the DB so that a DB read does not grant token theft — an attacker
// who only reads the refresh_tokens table cannot impersonate users.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// HashEmail returns a non-reversible, salted hex digest of an email.
//
// Why: a handful of flows (e.g. "is this email already registered?") must
// reason about email without storing it in plaintext indexes. When used, the
// caller should still prefer the actual unique constraint on users.email;
// this helper exists for analytics / audit entries that need a stable id
// but must never reveal the address.
//
// Salt is mixed in so that an attacker with a rainbow table of common
// emails cannot reverse the digest.
func HashEmail(email, salt string) string {
	if salt == "" {
		panic("auth: HashEmail called with empty salt")
	}
	msg := email + ":" + salt
	sum := sha256.Sum256([]byte(msg))
	return hex.EncodeToString(sum[:])
}

// FormatUserHashLogValue returns a short, log-friendly representation of a
// user_hash (first 12 hex chars). Used when logging: full hash is fine to
// persist but is verbose; truncating keeps log lines readable while still
// uniquely identifying a user in practice (48 bits of entropy).
func FormatUserHashLogValue(userHash string) string {
	if len(userHash) <= 12 {
		return userHash
	}
	return fmt.Sprintf("%s…", userHash[:12])
}
