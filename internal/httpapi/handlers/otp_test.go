package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
