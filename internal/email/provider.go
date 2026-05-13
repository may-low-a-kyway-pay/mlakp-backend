package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
	Endpoint  string
}

type Provider struct {
	config Config
	client *http.Client
}

func NewProvider(config Config) *Provider {
	return &Provider{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Provider) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	payload := map[string]interface{}{
		"From":       p.fromAddress(),
		"To":         to,
		"Subject":    subject,
		"HtmlBody":   htmlBody,
		"TrackOpens": true,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Server-Token", p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("postmark request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("postmark api error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (p *Provider) fromAddress() string {
	fromEmail := strings.TrimSpace(p.config.FromEmail)
	fromName := strings.TrimSpace(p.config.FromName)
	if fromName == "" {
		return fromEmail
	}

	return fmt.Sprintf("%s <%s>", fromName, fromEmail)
}

func (p *Provider) SendOTPEmail(ctx context.Context, to, name, otp, purpose string) error {
	tmpl, err := template.New("otp_email").Parse(otpEmailTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Name    string
		OTP     string
		Purpose string
	}{
		Name:    name,
		OTP:     otp,
		Purpose: formatPurpose(purpose),
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	return p.SendEmail(ctx, to, "Verify your email - PonyPigeon", body.String())
}

func formatPurpose(purpose string) string {
	switch purpose {
	case "signup":
		return "Email Verification"
	case "password_reset":
		return "Password Reset"
	default:
		return purpose
	}
}
