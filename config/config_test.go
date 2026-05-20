package config

import "testing"

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
