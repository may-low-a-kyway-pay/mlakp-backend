package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{})

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
	router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), RouterDeps{})

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
