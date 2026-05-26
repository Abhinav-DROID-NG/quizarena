package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCORSOrigins(t *testing.T) {
	origins := CORSOrigins("https://a.com, https://b.com")
	if len(origins) != 2 || origins[0] != "https://a.com" || origins[1] != "https://b.com" {
		t.Fatalf("unexpected origins: %#v", origins)
	}

	fallback := CORSOrigins(" , ")
	if len(fallback) != 1 || fallback[0] != "*" {
		t.Fatalf("expected wildcard fallback got %#v", fallback)
	}
}

func TestGetEnvAsInt32(t *testing.T) {
	t.Setenv("DB_MAX_CONNS", "40")
	if got := getEnvAsInt32("DB_MAX_CONNS", 30); got != 40 {
		t.Fatalf("expected 40 got %d", got)
	}

	t.Setenv("DB_MAX_CONNS", "not-a-number")
	if got := getEnvAsInt32("DB_MAX_CONNS", 30); got != 30 {
		t.Fatalf("expected default for invalid value, got %d", got)
	}

	t.Setenv("DB_MAX_CONNS", "2147483648")
	if got := getEnvAsInt32("DB_MAX_CONNS", 30); got != 30 {
		t.Fatalf("expected default for overflow value, got %d", got)
	}
}

func TestLoadReadsDotEnv(t *testing.T) {
	key := "QUIZARENA_TEST_GOOGLE_CLIENT_ID"
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv(key)
	})

	dotEnvPath := filepath.Join(t.TempDir(), ".env")
	content := []byte(key + "=test-client-id\n")
	if err := os.WriteFile(dotEnvPath, content, 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	loadDotEnv(dotEnvPath)
	if got := os.Getenv(key); got != "test-client-id" {
		t.Fatalf("expected env from .env, got %q", got)
	}
}

func TestSplitCSV(t *testing.T) {
	values := splitCSV("admin@example.com, ops@example.com, ,")
	if len(values) != 2 || values[0] != "admin@example.com" || values[1] != "ops@example.com" {
		t.Fatalf("unexpected csv split: %#v", values)
	}
}

func TestValidateConfig(t *testing.T) {
	cfg := Config{
		Port:               "8080",
		FrontendOrigin:     "http://localhost:5500",
		JWTSecret:          "super-secret",
		JWTExpiration:      24 * time.Hour,
		GoogleClientID:     "client-id",
		AdminEmails:        []string{"admin@example.com"},
		DatabaseURL:        "db-url",
		DBMaxConns:         30,
		ShutdownTimeoutSec: 10,
	}
	warnings, err := Validate(cfg)
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
}
