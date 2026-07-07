package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name string
		pw   string
		want error
	}{
		{"empty", "", ErrPasswordTooShort},
		{"too_short", "1234567", ErrPasswordTooShort},
		{"boundary_min", "12345678", nil},
		{"normal", "correct horse battery staple", nil},
		{"cyrillic_8runes", "пароль123", nil}, // 9 runes — utf8 aware
		{"cyrillic_too_short", "пароль", ErrPasswordTooShort}, // 6 runes
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidatePassword(c.pw)
			if !errors.Is(err, c.want) {
				t.Errorf("ValidatePassword(%q) err = %v, want %v", c.pw, err, c.want)
			}
		})
	}
}

func TestValidatePassword_TooLong(t *testing.T) {
	// 73 runes — just over bcrypt's 72-byte limit (each rune 1 byte).
	pw := strings.Repeat("a", 73)
	if err := ValidatePassword(pw); !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("ValidatePassword(73 chars) err = %v, want ErrPasswordTooLong", err)
	}
}

func TestHashPassword_Roundtrip(t *testing.T) {
	pw := "supersecret-123"
	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("hash is empty")
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("expected bcrypt prefix, got %q", hash[:min(3, len(hash))])
	}
	if !CheckPassword(hash, pw) {
		t.Error("CheckPassword should accept the original password")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Error("CheckPassword should reject wrong password")
	}
}

func TestHashPassword_ShortRejectedBeforeBcrypt(t *testing.T) {
	if _, err := HashPassword("short"); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestHashPassword_UniqueSalts(t *testing.T) {
	pw := "samepassword-123"
	h1, _ := HashPassword(pw)
	h2, _ := HashPassword(pw)
	if h1 == h2 {
		t.Error("bcrypt should produce different hashes for same password (random salt)")
	}
	// Both must still validate against the same plaintext.
	if !CheckPassword(h1, pw) || !CheckPassword(h2, pw) {
		t.Error("both hashes must validate against original password")
	}
}

func TestCheckPassword_EmptyInputs(t *testing.T) {
	if CheckPassword("", "x") {
		t.Error("empty hash should not authenticate")
	}
	hash, _ := HashPassword("validpass-1")
	if CheckPassword(hash, "") {
		t.Error("empty password should not authenticate")
	}
}

func TestCheckPassword_MalformedHash(t *testing.T) {
	// A garbage string must never authenticate — must not panic either.
	if CheckPassword("not-a-real-hash", "whatever") {
		t.Error("malformed hash should not authenticate")
	}
}
