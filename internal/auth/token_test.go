package auth

import (
	"context"
	"testing"
	"time"
)

func TestTokenManagerIssuesAndValidatesAccessToken(t *testing.T) {
	manager := NewTokenManager("mlakp-backend", "mlakp-api", "test-secret", 15*time.Minute)
	manager.now = func() time.Time {
		return time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	}

	token, expiresAt, err := manager.IssueAccessToken(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}
	if expiresAt != manager.now().Add(15*time.Minute) {
		t.Fatalf("expiresAt = %s, want 15 minute ttl", expiresAt)
	}

	claims, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("claims.Subject = %q, want user-1", claims.Subject)
	}
	if claims.Issuer != "mlakp-backend" || claims.Audience != "mlakp-api" {
		t.Fatalf("claims issuer/audience = %q/%q, want configured values", claims.Issuer, claims.Audience)
	}
}

func TestTokenManagerRejectsTamperedToken(t *testing.T) {
	manager := NewTokenManager("mlakp-backend", "mlakp-api", "test-secret", 15*time.Minute)
	token, _, err := manager.IssueAccessToken(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	if _, err := manager.ValidateAccessToken(token + "x"); err == nil {
		t.Fatal("ValidateAccessToken() error = nil, want tampered token rejection")
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	manager := NewTokenManager("mlakp-backend", "mlakp-api", "test-secret", time.Minute)
	manager.now = func() time.Time {
		return now
	}

	token, _, err := manager.IssueAccessToken(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	manager.now = func() time.Time {
		return now.Add(2 * time.Minute)
	}
	if _, err := manager.ValidateAccessToken(token); err == nil {
		t.Fatal("ValidateAccessToken() error = nil, want expired token rejection")
	}
}
