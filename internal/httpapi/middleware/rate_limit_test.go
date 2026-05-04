package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterRejectsAfterLimit(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, newRateLimitRequest("/v1/auth/login", "192.0.2.10:1234"))
		if response.Code != http.StatusNoContent {
			t.Fatalf("request %d response.Code = %d, want %d", i+1, response.Code, http.StatusNoContent)
		}
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, newRateLimitRequest("/v1/auth/login", "192.0.2.10:1234"))

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
	if response.Header().Get("Retry-After") != "60" {
		t.Fatalf("Retry-After = %q, want %q", response.Header().Get("Retry-After"), "60")
	}
}

func TestRateLimiterKeysByPathAndClient(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), newRateLimitRequest("/v1/auth/login", "192.0.2.10:1234"))

	for _, request := range []*http.Request{
		newRateLimitRequest("/v1/auth/register", "192.0.2.10:1234"),
		newRateLimitRequest("/v1/auth/login", "192.0.2.11:1234"),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s %s response.Code = %d, want %d", request.URL.Path, request.RemoteAddr, response.Code, http.StatusNoContent)
		}
	}
}

func newRateLimitRequest(path, remoteAddr string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, path, nil)
	request.RemoteAddr = remoteAddr
	return request
}
