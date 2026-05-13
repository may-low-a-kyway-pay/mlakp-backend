package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mlakp-backend/internal/auth"
	"mlakp-backend/internal/sessions"
	"mlakp-backend/internal/users"
)

func TestLoginRejectsUnverifiedUser(t *testing.T) {
	userStore := &authHandlerUserStore{
		user: users.PrivateUser{
			User: users.User{
				ID:       "user-1",
				Name:     "Thomas",
				Username: "thomas",
				Email:    "thomas@example.com",
			},
			PasswordHash: "hash:password123",
		},
	}
	sessionStore := &authHandlerSessionStore{}
	handler := NewAuthHandler(
		users.NewService(userStore, authHandlerHasher{}),
		auth.NewTokenManager("issuer", "audience", "secret", time.Minute),
		sessions.NewService(sessionStore, time.Hour),
	)

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{
		"email": "thomas@example.com",
		"password": "password123"
	}`))
	response := httptest.NewRecorder()

	handler.Login(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusForbidden)
	}
	if !strings.Contains(response.Body.String(), `"code":"email_not_verified"`) {
		t.Fatalf("response.Body = %q, want email_not_verified error", response.Body.String())
	}
	if sessionStore.created {
		t.Fatal("Login created a session for an unverified user")
	}
}

func TestLoginAllowsVerifiedUser(t *testing.T) {
	verifiedAt := time.Now()
	userStore := &authHandlerUserStore{
		user: users.PrivateUser{
			User: users.User{
				ID:              "user-1",
				Name:            "Thomas",
				Username:        "thomas",
				Email:           "thomas@example.com",
				EmailVerifiedAt: &verifiedAt,
			},
			PasswordHash: "hash:password123",
		},
	}
	sessionStore := &authHandlerSessionStore{}
	handler := NewAuthHandler(
		users.NewService(userStore, authHandlerHasher{}),
		auth.NewTokenManager("issuer", "audience", "secret", time.Minute),
		sessions.NewService(sessionStore, time.Hour),
	)

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{
		"email": "thomas@example.com",
		"password": "password123"
	}`))
	response := httptest.NewRecorder()

	handler.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !sessionStore.created {
		t.Fatal("Login did not create a session for a verified user")
	}
}

type authHandlerUserStore struct {
	user users.PrivateUser
}

func (s *authHandlerUserStore) Create(ctx context.Context, name, username, email, passwordHash string) (users.PrivateUser, error) {
	return users.PrivateUser{}, nil
}

func (s *authHandlerUserStore) GetByEmail(ctx context.Context, email string) (users.PrivateUser, error) {
	return s.user, nil
}

func (s *authHandlerUserStore) GetByID(ctx context.Context, id string) (users.User, error) {
	return users.User{}, nil
}

func (s *authHandlerUserStore) GetByUsername(ctx context.Context, username string) (users.User, error) {
	return users.User{}, nil
}

func (s *authHandlerUserStore) SearchByUsername(ctx context.Context, query string, limit int32) ([]users.User, error) {
	return nil, nil
}

func (s *authHandlerUserStore) UpdateUsername(ctx context.Context, id, username string) (users.User, error) {
	return users.User{}, nil
}

func (s *authHandlerUserStore) MarkEmailVerified(ctx context.Context, id string) (users.User, error) {
	return users.User{}, nil
}

func (s *authHandlerUserStore) RevokeAllUserSessions(ctx context.Context, userID string) error {
	return nil
}

func (s *authHandlerUserStore) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return nil
}

type authHandlerSessionStore struct {
	created bool
}

func (s *authHandlerSessionStore) Create(ctx context.Context, userID, refreshTokenHash string, expiresAt time.Time) (sessions.Session, error) {
	s.created = true
	return sessions.Session{
		ID:        "session-1",
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *authHandlerSessionStore) GetActiveByID(ctx context.Context, id string) (sessions.Session, error) {
	return sessions.Session{}, nil
}

func (s *authHandlerSessionStore) RotateRefreshToken(ctx context.Context, oldRefreshTokenHash, newRefreshTokenHash string) (sessions.Session, error) {
	return sessions.Session{}, nil
}

func (s *authHandlerSessionStore) Revoke(ctx context.Context, id string) error {
	return nil
}

func (s *authHandlerSessionStore) RevokeAllForUser(ctx context.Context, userID string) error {
	return nil
}

type authHandlerHasher struct{}

func (authHandlerHasher) HashPassword(password string) (string, error) {
	return "hash:" + password, nil
}

func (authHandlerHasher) ComparePassword(hash, password string) bool {
	return hash == "hash:"+password
}
