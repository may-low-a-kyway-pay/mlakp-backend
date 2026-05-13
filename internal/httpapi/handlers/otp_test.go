package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mlakp-backend/internal/otp"
	"mlakp-backend/internal/users"
)

func TestVerifyOTPRejectsPasswordResetPurpose(t *testing.T) {
	handler := NewOTPHandler(nil, nil, nil, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/verify-otp", strings.NewReader(`{
		"email": "thomas@example.com",
		"otp": "123456",
		"purpose": "password_reset"
	}`))
	response := httptest.NewRecorder()

	handler.VerifyOTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), `"code":"invalid_purpose"`) {
		t.Fatalf("response.Body = %q, want invalid_purpose error", response.Body.String())
	}
}

func TestSendPasswordResetOTPDoesNotRevealUnknownEmail(t *testing.T) {
	handler := NewOTPHandler(
		users.NewService(&authHandlerUserStore{getByEmailErr: users.ErrNotFound}, authHandlerHasher{}),
		otp.NewService(&otpHandlerStore{}, otp.Config{
			ExpiryMinutes:     10,
			RequestCooldown:   60,
			MaxAttempts:       5,
			RequestsPerWindow: 3,
			WindowMins:        10,
		}),
		nil,
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/send-otp", strings.NewReader(`{
		"email": "missing@example.com",
		"purpose": "password_reset"
	}`))
	response := httptest.NewRecorder()

	handler.SendOTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "email_not_found") {
		t.Fatalf("response.Body = %q, should not reveal whether the email exists", response.Body.String())
	}
}

type otpHandlerStore struct{}

func (s *otpHandlerStore) Create(ctx context.Context, params otp.CreateParams) (otp.Verification, error) {
	return otp.Verification{}, nil
}

func (s *otpHandlerStore) GetActiveByEmailAndPurpose(ctx context.Context, email, purpose string) (otp.Verification, error) {
	return otp.Verification{}, otp.ErrOTPNotFound
}

func (s *otpHandlerStore) IncrementAttempt(ctx context.Context, id string) (otp.Verification, error) {
	return otp.Verification{}, nil
}

func (s *otpHandlerStore) MarkVerified(ctx context.Context, id string) error {
	return nil
}

func (s *otpHandlerStore) ExpireOldVerifications(ctx context.Context, email, purpose string) error {
	return nil
}

func (s *otpHandlerStore) CountRecentRequests(ctx context.Context, email, purpose string, windowStart time.Time) (int, error) {
	return 0, nil
}
