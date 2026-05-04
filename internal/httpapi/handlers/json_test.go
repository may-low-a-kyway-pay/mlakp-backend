package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsTrailingJSON(t *testing.T) {
	request := &http.Request{
		Body: io.NopCloser(strings.NewReader(`{"name":"Thomas"} {"extra":true}`)),
	}

	var payload struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(request, &payload); err == nil {
		t.Fatal("decodeJSON returned nil, want error for trailing JSON")
	}
}

func TestDecodeJSONAllowsTrailingWhitespace(t *testing.T) {
	request := &http.Request{
		Body: io.NopCloser(strings.NewReader("{\"name\":\"Thomas\"}\n\t ")),
	}

	var payload struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(request, &payload); err != nil {
		t.Fatalf("decodeJSON returned error: %v", err)
	}
	if payload.Name != "Thomas" {
		t.Fatalf("payload.Name = %q, want %q", payload.Name, "Thomas")
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	request := &http.Request{
		Body: io.NopCloser(strings.NewReader(`{"name":"Thomas","role":"admin"}`)),
	}

	var payload struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(request, &payload); err == nil {
		t.Fatal("decodeJSON returned nil, want error for unknown field")
	}
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	request := &http.Request{
		Body: io.NopCloser(strings.NewReader(`{"name":"` + strings.Repeat("a", int(maxRequestBodyBytes)) + `"}`)),
	}

	var payload struct {
		Name string `json:"name"`
	}

	if err := decodeJSON(request, &payload); !errors.Is(err, errRequestBodyTooLarge) {
		t.Fatalf("decodeJSON error = %v, want %v", err, errRequestBodyTooLarge)
	}
}

func TestWriteDecodeErrorHandlesOversizedBody(t *testing.T) {
	response := httptest.NewRecorder()

	writeDecodeError(response, errRequestBodyTooLarge)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("response.Code = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(response.Body.String(), `"code":"request_body_too_large"`) {
		t.Fatalf("response.Body = %q, want request_body_too_large error", response.Body.String())
	}
}

func TestWriteAuthNoStoreHeaders(t *testing.T) {
	response := httptest.NewRecorder()

	writeAuthNoStoreHeaders(response)

	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", response.Header().Get("Cache-Control"), "no-store")
	}
	if response.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("Pragma = %q, want %q", response.Header().Get("Pragma"), "no-cache")
	}
}
