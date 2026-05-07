package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mlakp-backend/internal/httpapi/handlers"
	"mlakp-backend/internal/httpapi/middleware"
)

func TestHealthz(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "{\"success\":true,\"data\":{\"status\":\"ok\"}}\n" {
		t.Fatalf("response.Body = %q, want health response", response.Body.String())
	}
}

func TestReadyz(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{})

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "{\"success\":true,\"data\":{\"status\":\"ready\"}}\n" {
		t.Fatalf("response.Body = %q, want ready response", response.Body.String())
	}
}

func TestDocs(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{AppEnv: "local"})

	request := httptest.NewRequest(http.MethodGet, "/docs", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("response.Body does not contain SwaggerUIBundle")
	}
}

func TestOpenAPIYAML(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{AppEnv: "test"})

	request := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "openapi: 3.0.3") {
		t.Fatalf("response.Body does not contain OpenAPI document")
	}
}

func TestDocsDisabledInProduction(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{AppEnv: "production"})

	for _, path := range []string{"/docs", "/docs/openapi.yaml"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code != http.StatusNotFound {
			t.Fatalf("%s response.Code = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func TestAuthRoutesUseRateLimiter(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{
		AuthHandler:     handlers.NewAuthHandler(nil, nil, nil),
		AuthRateLimiter: middleware.NewRateLimiter(0, time.Minute),
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"email":"thomas@example.com","password":"password123"}`))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}

func TestProductionSecurityHeaders(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{AppEnv: "production"})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Header().Get("Strict-Transport-Security") != "max-age=31536000; includeSubDomains" {
		t.Fatalf("Strict-Transport-Security = %q, want production HSTS", response.Header().Get("Strict-Transport-Security"))
	}
	if response.Header().Get("Content-Security-Policy") != "default-src 'none'; frame-ancestors 'none'; base-uri 'none'" {
		t.Fatalf("Content-Security-Policy = %q, want production CSP", response.Header().Get("Content-Security-Policy"))
	}
}

func TestLocalSecurityHeadersDoNotSetHSTS(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{AppEnv: "local"})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("Strict-Transport-Security = %q, want empty for local", response.Header().Get("Strict-Transport-Security"))
	}
}

func TestCORSAllowedOrigin(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{
		AppEnv:      "production",
		CORSOrigins: []string{"http://localhost:8081"},
	})

	request := httptest.NewRequest(http.MethodOptions, "/v1/auth/login", nil)
	request.Header.Set("Origin", "http://localhost:8081")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusNoContent)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8081" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want allowed origin", response.Header().Get("Access-Control-Allow-Origin"))
	}
	if response.Header().Get("Strict-Transport-Security") != "max-age=31536000; includeSubDomains" {
		t.Fatalf("Strict-Transport-Security = %q, want production HSTS", response.Header().Get("Strict-Transport-Security"))
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{
		CORSOrigins: []string{"http://localhost:8081"},
	})

	request := httptest.NewRequest(http.MethodOptions, "/v1/auth/login", nil)
	request.Header.Set("Origin", "http://malicious.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", response.Header().Get("Access-Control-Allow-Origin"))
	}
}
