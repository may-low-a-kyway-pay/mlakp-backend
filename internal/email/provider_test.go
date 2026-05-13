package email

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestSendEmailUsesConfiguredSenderName(t *testing.T) {
	var body string
	provider := NewProvider(Config{
		APIKey:    "test-token",
		FromEmail: "noreply@example.com",
		FromName:  "PonyPigeon",
		Endpoint:  "https://postmark.test/email",
	})
	provider.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			data, _ := io.ReadAll(r.Body)
			body = string(data)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	if err := provider.SendEmail(context.Background(), "thomas@example.com", "Subject", "<p>Hello</p>"); err != nil {
		t.Fatalf("SendEmail returned error: %v", err)
	}
	if !strings.Contains(body, `"From":"PonyPigeon \u003cnoreply@example.com\u003e"`) {
		t.Fatalf("request body = %s, want configured sender name and email", body)
	}
}

func TestSendEmailIncludesPostmarkErrorBody(t *testing.T) {
	provider := NewProvider(Config{
		APIKey:    "bad-token",
		FromEmail: "noreply@example.com",
		Endpoint:  "https://postmark.test/email",
	})
	provider.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"ErrorCode":10,"Message":"Invalid API token"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := provider.SendEmail(context.Background(), "thomas@example.com", "Subject", "<p>Hello</p>")
	if err == nil {
		t.Fatal("SendEmail error = nil, want Postmark error")
	}
	if !strings.Contains(err.Error(), "Invalid API token") {
		t.Fatalf("SendEmail error = %q, want Postmark response body", err.Error())
	}
}
