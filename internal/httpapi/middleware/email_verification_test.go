package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mlakp-backend/internal/users"
)

func TestRequireVerifiedEmailRejectsUnverifiedUserInsideGracePeriod(t *testing.T) {
	deadline := time.Now().Add(24 * time.Hour)
	usersService := users.NewService(&emailVerificationUserStore{
		user: users.User{
			ID:                   "user-1",
			Email:                "thomas@example.com",
			VerificationDeadline: &deadline,
		},
	}, nil)
	middleware := NewEmailVerificationMiddleware(usersService)
	handler := middleware.RequireVerifiedEmail(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/v1/expenses", nil)
	request = request.WithContext(context.WithValue(request.Context(), userIDContextKey{}, "user-1"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusForbidden)
	}
	if !strings.Contains(response.Body.String(), `"code":"email_verification_required"`) {
		t.Fatalf("response.Body = %q, want email_verification_required error", response.Body.String())
	}
}

type emailVerificationUserStore struct {
	user users.User
}

func (s *emailVerificationUserStore) Create(ctx context.Context, name, username, email, passwordHash string) (users.PrivateUser, error) {
	return users.PrivateUser{}, nil
}

func (s *emailVerificationUserStore) GetByEmail(ctx context.Context, email string) (users.PrivateUser, error) {
	return users.PrivateUser{}, nil
}

func (s *emailVerificationUserStore) GetByID(ctx context.Context, id string) (users.User, error) {
	return s.user, nil
}

func (s *emailVerificationUserStore) GetByUsername(ctx context.Context, username string) (users.User, error) {
	return users.User{}, nil
}

func (s *emailVerificationUserStore) SearchByUsername(ctx context.Context, query string, limit int32) ([]users.User, error) {
	return nil, nil
}

func (s *emailVerificationUserStore) UpdateUsername(ctx context.Context, id, username string) (users.User, error) {
	return users.User{}, nil
}

func (s *emailVerificationUserStore) MarkEmailVerified(ctx context.Context, id string) (users.User, error) {
	return users.User{}, nil
}

func (s *emailVerificationUserStore) RevokeAllUserSessions(ctx context.Context, userID string) error {
	return nil
}

func (s *emailVerificationUserStore) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return nil
}
