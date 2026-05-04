package handlers

import (
	"io"
	"net/http"
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
