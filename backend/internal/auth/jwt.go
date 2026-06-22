// Package auth implements authentication primitives for FinHelper:
// password hashing, anonymous user_hash derivation, and JWT issuance/verify.
//
// JWT design (Plane from GLM.md → Задача 2):
//   - access token:  short-lived (15m default), signed with AccessSecret
//   - refresh token: long-lived (30d default), signed with RefreshSecret,
//                    stored ONLY as SHA-256 in the DB (see HashToken)
//   - separate secrets so leakage of one class never forges the other
//
// Claims carry user_id (int64, stable across email edits) and user_hash
// (anonymous id for logs/analytics — never email).
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenKind distinguishes the two token roles. We embed it in the token via
// a custom claim "kind", and check it on verify — so an access token can
// never be replayed against the refresh endpoint and vice versa.
type TokenKind string

const (
	KindAccess  TokenKind = "access"
	KindRefresh TokenKind = "refresh"
)

// Standard claim keys we depend on.
const (
	claimKind = "kind"
)

// Errors surfaced by Verify. Handlers map these to 401/400 uniformly.
var (
	ErrTokenExpired = errors.New("auth: token expired")
	ErrTokenInvalid = errors.New("auth: token invalid")
	ErrWrongKind    = errors.New("auth: token kind mismatch")
)

// Claims is the body of every FinHelper JWT.
//
// Why a custom struct instead of just jwt.RegisteredClaims:
//   - we need user_id and user_hash as typed fields, not untyped map entries;
//   - embedding RegisteredClaims gives us iss/exp/iat/sub for free, plus
//     standardized validation via jwt's own VerifyExpiresAt etc.
type Claims struct {
	UserID   int64  `json:"uid"`
	UserHash string `json:"uhsh"`
	Kind     string `json:"kind"`
	jwt.RegisteredClaims
}

// JWTIssuer signs and verifies access/refresh tokens. Construct one per
// process via NewJWTIssuer; it's safe for concurrent use because jwt's
// signing internals are stateless.
type JWTIssuer struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewJWTIssuer constructs a JWTIssuer. Secrets are copied so the caller can
// wipe its own buffers if desired. TTLs must be positive — we enforce that
// here so a misconfigured env ("JWT_ACCESS_TTL=0") fails loudly at boot.
func NewJWTIssuer(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) (*JWTIssuer, error) {
	if len(accessSecret) < 32 {
		return nil, fmt.Errorf("auth: access secret must be ≥32 bytes, got %d", len(accessSecret))
	}
	if len(refreshSecret) < 32 {
		return nil, fmt.Errorf("auth: refresh secret must be ≥32 bytes, got %d", len(refreshSecret))
	}
	if accessTTL <= 0 {
		return nil, fmt.Errorf("auth: access TTL must be positive, got %v", accessTTL)
	}
	if refreshTTL <= 0 {
		return nil, fmt.Errorf("auth: refresh TTL must be positive, got %v", refreshTTL)
	}
	return &JWTIssuer{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}, nil
}

// IssueAccess produces a signed access token for the given user.
func (j *JWTIssuer) IssueAccess(userID int64, userHash, subject string) (string, time.Time, error) {
	return j.issue(j.accessSecret, j.accessTTL, KindAccess, userID, userHash, subject)
}

// IssueRefresh produces a signed refresh token. The opaque "subject" field is
// unused but kept for future extensibility (e.g. device/session id).
func (j *JWTIssuer) IssueRefresh(userID int64, userHash, subject string) (string, time.Time, error) {
	return j.issue(j.refreshSecret, j.refreshTTL, KindRefresh, userID, userHash, subject)
}

func (j *JWTIssuer) issue(secret []byte, ttl time.Duration, kind TokenKind, userID int64, userHash, subject string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(ttl)
	claims := Claims{
		UserID:   userID,
		UserHash: userHash,
		Kind:     string(kind),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign %s token: %w", kind, err)
	}
	return signed, exp, nil
}

// VerifyAccess validates an access token and returns its claims.
// On any failure it returns a wrapped error from {ErrTokenExpired, ErrTokenInvalid, ErrWrongKind}.
func (j *JWTIssuer) VerifyAccess(token string) (*Claims, error) {
	return j.verify(j.accessSecret, token, KindAccess)
}

// VerifyRefresh validates a refresh token and returns its claims.
func (j *JWTIssuer) VerifyRefresh(token string) (*Claims, error) {
	return j.verify(j.refreshSecret, token, KindRefresh)
}

func (j *JWTIssuer) verify(secret []byte, token string, want TokenKind) (*Claims, error) {
	if token == "" {
		return nil, ErrTokenInvalid
	}
	parsed := &Claims{}
	// We pin the algorithm to HS256 explicitly in keyFunc — this prevents the
	// classic JWT alg-confusion attack where an attacker swaps HS256 for "none"
	// or RS256 to bypass verification.
	parsedToken, err := jwt.ParseWithClaims(token, parsed, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		// Distinguish "expired" from other failures so handlers can return
		// the right HTTP semantics (401 either way, but logs differ).
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
	if parsedToken == nil || !parsedToken.Valid {
		return nil, ErrTokenInvalid
	}
	// Defense-in-depth: even if a token is well-signed, reject if it's the
	// wrong kind. Prevents using an access token to refresh and vice versa.
	if parsed.Kind != string(want) {
		return nil, ErrWrongKind
	}
	return parsed, nil
}

// AccessTTL / RefreshTTL expose the configured TTLs for callers that need to
// set cookie expiry or schedule background rotation.
func (j *JWTIssuer) AccessTTL() time.Duration  { return j.accessTTL }
func (j *JWTIssuer) RefreshTTL() time.Duration { return j.refreshTTL }

// JWTVerifier is the read-only surface of an issuer (verify-only). The HTTP
// middleware depends on this interface rather than the concrete *JWTIssuer
// so tests can swap in a fake.
type JWTVerifier interface {
	Verify(token string) (*Claims, error)
}

// Verify satisfies JWTVerifier by validating an access-kind token — the form
// AuthMiddleware calls. Refresh verification keeps its own name (VerifyRefresh)
// because the two token classes use different secrets.
func (j *JWTIssuer) Verify(token string) (*Claims, error) {
	return j.verify(j.accessSecret, token, KindAccess)
}
