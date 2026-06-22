package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// clearEnvVars removes the given env vars for the duration of the test.
func clearEnvVars(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestLoad_MissingRequiredSecrets(t *testing.T) {
	clearEnvVars(t, "DATABASE_URL", "JWT_ACCESS_SECRET", "JWT_REFRESH_SECRET", "USER_HASH_SALT")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when required env vars are missing")
	}
	// DATABASE_URL is optional at config-load time; main() handles the
	// missing case. So the error must mention only the secrets/salt.
	for _, want := range []string{"JWT_ACCESS_SECRET", "JWT_REFRESH_SECRET", "USER_HASH_SALT"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("DATABASE_URL should be optional at load time, got: %v", err)
	}
}

func TestLoad_DatabaseURLOptional(t *testing.T) {
	// No DATABASE_URL set — should still succeed at config load time.
	clearEnvVars(t, "DATABASE_URL")
	t.Setenv("JWT_ACCESS_SECRET", strings.Repeat("a", 32))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("b", 32))
	t.Setenv("USER_HASH_SALT", "somesalt")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config should load without DATABASE_URL, got: %v", err)
	}
	if cfg.Database.URL != "" {
		t.Errorf("expected empty DATABASE_URL, got %q", cfg.Database.URL)
	}
}

func TestLoad_ShortSecrets(t *testing.T) {
	clearEnvVars(t, "JWT_ACCESS_SECRET", "JWT_REFRESH_SECRET")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("USER_HASH_SALT", "salt")
	t.Setenv("JWT_ACCESS_SECRET", "tooshort")
	t.Setenv("JWT_REFRESH_SECRET", "alsoshort")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for short JWT secrets")
	}
	if !strings.Contains(err.Error(), "at least 32 characters") {
		t.Errorf("error should mention secret length, got: %v", err)
	}
}

func TestLoad_OK(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_ACCESS_SECRET", strings.Repeat("a", 32))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("b", 32))
	t.Setenv("USER_HASH_SALT", "somesalt")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("default HTTP_ADDR, got %q", cfg.HTTP.Addr)
	}
	if cfg.JWT.AccessTTL != 15*time.Minute {
		t.Errorf("default AccessTTL, got %v", cfg.JWT.AccessTTL)
	}
	if cfg.JWT.RefreshTTL != 30*24*time.Hour {
		t.Errorf("default RefreshTTL, got %v", cfg.JWT.RefreshTTL)
	}
	// Default CORS should include Vite dev server.
	if len(cfg.HTTP.CORSAllowedOrigins) != 1 || cfg.HTTP.CORSAllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("default CORS origins wrong, got %v", cfg.HTTP.CORSAllowedOrigins)
	}
}

func TestLoad_BadDuration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_ACCESS_SECRET", strings.Repeat("a", 32))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("b", 32))
	t.Setenv("USER_HASH_SALT", "somesalt")
	t.Setenv("JWT_ACCESS_TTL", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for malformed JWT_ACCESS_TTL")
	}
	if !strings.Contains(err.Error(), "JWT_ACCESS_TTL") {
		t.Errorf("error should mention JWT_ACCESS_TTL, got: %v", err)
	}
}

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"  a , b , c  ", []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		got := splitCSV(c.in)
		if len(got) != len(c.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("splitCSV(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}
