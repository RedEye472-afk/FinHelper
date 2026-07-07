package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testAccessSecret  = "test-access-secret-must-be-at-least-32-chars"
	testRefreshSecret = "test-refresh-secret-must-be-at-least-32-chars"
)

func testIssuer(t *testing.T) *JWTIssuer {
	t.Helper()
	iss, err := NewJWTIssuer(testAccessSecret, testRefreshSecret, time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewJWTIssuer: %v", err)
	}
	return iss
}

func TestNewJWTIssuer_RejectsShortSecrets(t *testing.T) {
	cases := []struct {
		name         string
		accessSecret string
		refreshSec   string
	}{
		{"short_access", "too-short", testRefreshSecret},
		{"short_refresh", testAccessSecret, "too-short"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewJWTIssuer(c.accessSecret, c.refreshSec, time.Minute, time.Hour); err == nil {
				t.Error("expected error for short secret")
			}
		})
	}
}

func TestNewJWTIssuer_RejectsNonPositiveTTL(t *testing.T) {
	if _, err := NewJWTIssuer(testAccessSecret, testRefreshSecret, 0, time.Hour); err == nil {
		t.Error("expected error for zero access TTL")
	}
	if _, err := NewJWTIssuer(testAccessSecret, testRefreshSecret, time.Minute, 0); err == nil {
		t.Error("expected error for zero refresh TTL")
	}
	if _, err := NewJWTIssuer(testAccessSecret, testRefreshSecret, -time.Second, time.Hour); err == nil {
		t.Error("expected error for negative access TTL")
	}
}

func TestIssueAndVerifyAccess_Roundtrip(t *testing.T) {
	iss := testIssuer(t)
	tok, exp, err := iss.IssueAccess(42, "user-hash-abc", "sub-42")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}
	if !exp.After(time.Now()) {
		t.Errorf("expiry should be in the future, got %v", exp)
	}

	claims, err := iss.VerifyAccess(tok)
	if err != nil {
		t.Fatalf("VerifyAccess: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("uid = %d, want 42", claims.UserID)
	}
	if claims.UserHash != "user-hash-abc" {
		t.Errorf("uhsh = %q", claims.UserHash)
	}
	if claims.Kind != string(KindAccess) {
		t.Errorf("kind = %q, want %q", claims.Kind, KindAccess)
	}
}

func TestIssueAndVerifyRefresh_Roundtrip(t *testing.T) {
	iss := testIssuer(t)
	tok, _, err := iss.IssueRefresh(42, "h", "s")
	if err != nil {
		t.Fatalf("IssueRefresh: %v", err)
	}
	claims, err := iss.VerifyRefresh(tok)
	if err != nil {
		t.Fatalf("VerifyRefresh: %v", err)
	}
	if claims.Kind != string(KindRefresh) {
		t.Errorf("kind = %q", claims.Kind)
	}
}

func TestVerify_KindCrossUseRejected(t *testing.T) {
	iss := testIssuer(t)
	accessTok, _, _ := iss.IssueAccess(1, "h", "s")
	refreshTok, _, _ := iss.IssueRefresh(1, "h", "s")

	// With separate access/refresh secrets (the production configuration),
	// cross-use fails at signature verification — kind check is pure
	// defense-in-depth. Either ErrTokenInvalid or ErrWrongKind is acceptable;
	// the user-facing guarantee is "the token must not be accepted".
	if err := isAcceptableKindErr(iss.VerifyRefresh(accessTok)); err != nil {
		t.Errorf("access-as-refresh: %v", err)
	}
	if err := isAcceptableKindErr(iss.VerifyAccess(refreshTok)); err != nil {
		t.Errorf("refresh-as-access: %v", err)
	}
}

func TestVerify_KindCheckTriggersWithSharedSecret(t *testing.T) {
	// Defense-in-depth: even if a misconfigured deployment used the SAME
	// secret for access and refresh, the embedded "kind" claim still
	// prevents cross-use. VerifyRefresh must reject an access-kind token.
	iss, err := NewJWTIssuer(testAccessSecret, testAccessSecret, time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewJWTIssuer: %v", err)
	}
	accessTok, _, _ := iss.IssueAccess(1, "h", "s")
	refreshTok, _, _ := iss.IssueRefresh(1, "h", "s")

	if _, err := iss.VerifyRefresh(accessTok); !errors.Is(err, ErrWrongKind) {
		t.Errorf("access-as-refresh with shared secret: err = %v, want ErrWrongKind", err)
	}
	if _, err := iss.VerifyAccess(refreshTok); !errors.Is(err, ErrWrongKind) {
		t.Errorf("refresh-as-access with shared secret: err = %v, want ErrWrongKind", err)
	}
}

// isAcceptableKindErr returns nil if err is one of the rejection errors we
// accept for cross-kind use, or a descriptive error otherwise.
func isAcceptableKindErr(_ *Claims, err error) error {
	if err == nil {
		return errors.New("expected rejection, got nil")
	}
	if errors.Is(err, ErrTokenInvalid) || errors.Is(err, ErrWrongKind) {
		return nil
	}
	return err
}

func TestVerify_WrongSecret(t *testing.T) {
	iss1, _ := NewJWTIssuer(testAccessSecret, testRefreshSecret, time.Minute, time.Hour)
	iss2, _ := NewJWTIssuer("another-secret-that-is-also-32+chars-long!", testRefreshSecret, time.Minute, time.Hour)

	tok, _, _ := iss1.IssueAccess(1, "h", "s")
	if _, err := iss2.VerifyAccess(tok); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("wrong-secret verify: err = %v, want ErrTokenInvalid", err)
	}
}

func TestVerify_EmptyToken(t *testing.T) {
	iss := testIssuer(t)
	if _, err := iss.VerifyAccess(""); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("empty token: err = %v, want ErrTokenInvalid", err)
	}
}

func TestVerify_Expired(t *testing.T) {
	// 1ns TTL — issues a token already effectively expired.
	iss, err := NewJWTIssuer(testAccessSecret, testRefreshSecret, time.Nanosecond, time.Nanosecond)
	if err != nil {
		t.Fatalf("NewJWTIssuer: %v", err)
	}
	tok, _, _ := iss.IssueAccess(1, "h", "s")
	// Sleep long enough for the token to be expired by VerifyExpiresAt leeway.
	time.Sleep(15 * time.Millisecond)
	_, err = iss.VerifyAccess(tok)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expired token: err = %v, want ErrTokenExpired", err)
	}
}

func TestVerify_RejectsAlgNone(t *testing.T) {
	// alg=none is unsigned and the canonical JWT bypass. Our keyFunc must
	// reject it by requiring HMAC signing method. Forge a none token by hand.
	iss := testIssuer(t)
	header := `{"alg":"none","typ":"JWT"}`
	payload := `{"uid":1,"uhsh":"h","kind":"access"}`
	headerB := base64url(header)
	payloadB := base64url(payload)
	forged := headerB + "." + payloadB + "." // empty signature
	if _, err := iss.VerifyAccess(forged); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("alg=none: err = %v, want ErrTokenInvalid", err)
	}
}

func TestVerify_Garbage(t *testing.T) {
	iss := testIssuer(t)
	for _, bad := range []string{"not-a-jwt", "a.b", "a.b.c.d", "...", "x.y.z"} {
		if _, err := iss.VerifyAccess(bad); !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("VerifyAccess(%q): err = %v, want ErrTokenInvalid", bad, err)
		}
	}
}

func TestIssue_ClaimStability(t *testing.T) {
	// The same user must always produce verifiable claims; subject/iat/exp
	// don't need to be byte-identical across calls (exp advances with now),
	// but uid/uhsh/kind must.
	iss := testIssuer(t)
	tok1, _, _ := iss.IssueAccess(7, "hash-7", "s")
	tok2, _, _ := iss.IssueAccess(7, "hash-7", "s")
	c1, _ := iss.VerifyAccess(tok1)
	c2, _ := iss.VerifyAccess(tok2)
	if c1.UserID != c2.UserID || c1.UserHash != c2.UserHash || c1.Kind != c2.Kind {
		t.Error("identity claims must be stable across issues")
	}
}

func TestAccessTTLReflectsConfig(t *testing.T) {
	iss, _ := NewJWTIssuer(testAccessSecret, testRefreshSecret, 7*time.Hour, 30*24*time.Hour)
	if iss.AccessTTL() != 7*time.Hour {
		t.Errorf("AccessTTL = %v", iss.AccessTTL())
	}
	if iss.RefreshTTL() != 30*24*time.Hour {
		t.Errorf("RefreshTTL = %v", iss.RefreshTTL())
	}
}

// base64url is a tiny helper for crafting forged tokens in tests without
// pulling in encoding/json. It only handles ASCII payloads (sufficient here).
func base64url(s string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	// Use std encoding then translate; simpler: hand-roll for ASCII.
	var b strings.Builder
	buf := []byte(s)
	for i := 0; i < len(buf); i += 3 {
		var n uint32
		var cnt int
		for j := 0; j < 3 && i+j < len(buf); j++ {
			n |= uint32(buf[i+j]) << (16 - 8*j)
			cnt++
		}
		out := [4]byte{'=', '=', '=', '='}
		out[0] = alphabet[(n>>18)&63]
		out[1] = alphabet[(n>>12)&63]
		if cnt >= 2 {
			out[2] = alphabet[(n>>6)&63]
		}
		if cnt >= 3 {
			out[3] = alphabet[n&63]
		}
		b.Write(out[:])
	}
	// JWT uses raw URL encoding without padding.
	out := strings.TrimRight(b.String(), "=")
	return out
}
