package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadValidConfig(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppEnv != EnvLocal {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, EnvLocal)
	}
	if cfg.AppPort != "8080" {
		t.Fatalf("AppPort = %q, want %q", cfg.AppPort, "8080")
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Fatalf("AccessTokenTTL = %s, want 15m", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 720*time.Hour {
		t.Fatalf("RefreshTokenTTL = %s, want 720h", cfg.RefreshTokenTTL)
	}
}

func TestLoadReportsMissingRequiredValues(t *testing.T) {
	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want missing required values")
	}

	for _, want := range []string{"APP_ENV is required", "APP_PORT is required", "DATABASE_URL is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Load() error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_PORT", "70000")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want invalid port error")
	}
	if !strings.Contains(err.Error(), "APP_PORT is invalid") {
		t.Fatalf("Load() error = %q, want invalid port error", err.Error())
	}
}

func TestLoadRejectsShortProductionTokenSecret(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ENV", EnvProduction)
	t.Setenv("TOKEN_SECRET", "short")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want short production secret error")
	}
	if !strings.Contains(err.Error(), "TOKEN_SECRET must be at least 32 bytes in production") {
		t.Fatalf("Load() error = %q, want short production secret error", err.Error())
	}
}

func TestLoadParsesCORSOrigins(t *testing.T) {
	setValidEnv(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:8081, https://app.example.com ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.CORSOrigins) != 2 {
		t.Fatalf("len(cfg.CORSOrigins) = %d, want 2", len(cfg.CORSOrigins))
	}
	if cfg.CORSOrigins[0] != "http://localhost:8081" || cfg.CORSOrigins[1] != "https://app.example.com" {
		t.Fatalf("cfg.CORSOrigins = %#v, want parsed origins", cfg.CORSOrigins)
	}
}

func TestLoadRejectsInvalidCORSOrigin(t *testing.T) {
	setValidEnv(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "localhost:8081")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want invalid CORS origin error")
	}
	if !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
		t.Fatalf("Load() error = %q, want CORS_ALLOWED_ORIGINS validation error", err.Error())
	}
}

func setValidEnv(t *testing.T) {
	t.Helper()

	t.Setenv("APP_ENV", EnvLocal)
	t.Setenv("APP_PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/mlakp")
	t.Setenv("TOKEN_ISSUER", "mlakp-backend")
	t.Setenv("TOKEN_AUDIENCE", "mlakp-api")
	t.Setenv("TOKEN_SECRET", "local-development-secret")
	t.Setenv("ACCESS_TOKEN_TTL", "15m")
	t.Setenv("REFRESH_TOKEN_TTL", "720h")
	t.Setenv("READ_TIMEOUT", "5s")
	t.Setenv("WRITE_TIMEOUT", "10s")
	t.Setenv("IDLE_TIMEOUT", "60s")
	t.Setenv("SHUTDOWN_TIMEOUT", "10s")
	t.Setenv("POSTMARK_API_KEY", "test-api-key")
	t.Setenv("POSTMARK_FROM_EMAIL", "noreply@test.com")
	t.Setenv("POSTMARK_FROM_NAME", "Test App")
}
