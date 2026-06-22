package auth

import (
	"strings"
	"testing"
)

func TestUserHash_StableAndUnique(t *testing.T) {
	salt := "static-salt"
	h1 := UserHash(1, salt)
	h2 := UserHash(1, salt)
	h3 := UserHash(2, salt)

	if h1 != h2 {
		t.Errorf("UserHash must be deterministic: %s != %s", h1, h2)
	}
	if h1 == h3 {
		t.Errorf("different user_ids must produce different hashes: %s == %s", h1, h3)
	}
	// SHA-256 hex = 64 chars.
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex digest, got %d", len(h1))
	}
}

func TestUserHash_Salted(t *testing.T) {
	// Same user_id with different salts must differ.
	if UserHash(1, "salt-a") == UserHash(1, "salt-b") {
		t.Error("salt must affect user_hash")
	}
}

func TestUserHash_SeparatorPreventsCollision(t *testing.T) {
	// Without separator, "12" + "3x" salt == "1" + "23x" salt.
	// Our ":" separator makes them distinct.
	a := UserHash(12, "3x")
	b := UserHash(1, "23x")
	if a == b {
		t.Error("separator must prevent prefix-collision")
	}
}

func TestUserHash_PanicsOnEmptySalt(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty salt")
		}
	}()
	_ = UserHash(1, "")
}

func TestHashToken_DeterministicAndHex(t *testing.T) {
	h := HashToken("some-refresh-token")
	if len(h) != 64 {
		t.Errorf("expected 64-char hex, got %d", len(h))
	}
	if HashToken("some-refresh-token") != h {
		t.Error("HashToken must be deterministic")
	}
	if HashToken("different") == h {
		t.Error("different inputs must hash differently")
	}
}

func TestHashEmail_Salted(t *testing.T) {
	a := HashEmail("user@example.com", "salt")
	b := HashEmail("user@example.com", "salt")
	c := HashEmail("user@example.com", "other")
	if a != b {
		t.Error("HashEmail must be deterministic")
	}
	if a == c {
		t.Error("salt must affect hash")
	}
}

func TestFormatUserHashLogValue(t *testing.T) {
	full := strings.Repeat("a", 64)
	short := FormatUserHashLogValue(full)
	if !strings.HasSuffix(short, "…") {
		t.Errorf("expected ellipsis suffix, got %q", short)
	}
	// 12 ASCII chars + "…" (3 bytes in UTF-8) = 15 bytes; 13 runes.
	if got := len([]rune(short)); got != 13 {
		t.Errorf("expected 13-rune truncation, got %d (%q)", got, short)
	}
	// Short input returned as-is.
	if got := FormatUserHashLogValue("abc"); got != "abc" {
		t.Errorf("short input should pass through, got %q", got)
	}
}
