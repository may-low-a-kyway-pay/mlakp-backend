package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid access token")

type TokenManager struct {
	issuer   string
	audience string
	secret   []byte
	ttl      time.Duration
	now      func() time.Time
}

type Claims struct {
	Subject   string `json:"sub"`
	SessionID string `json:"sid"`
	TokenID   string `json:"jti"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	Expires   int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

func NewTokenManager(issuer, audience, secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		issuer:   issuer,
		audience: audience,
		secret:   []byte(secret),
		ttl:      ttl,
		now:      time.Now,
	}
}

// IssueAccessToken creates the compact HMAC token format used by this service:
// base64url(JSON claims) + "." + base64url(HMAC-SHA256(payload)).
func (m *TokenManager) IssueAccessToken(_ context.Context, subject, sessionID string) (string, time.Time, error) {
	if strings.TrimSpace(subject) == "" || strings.TrimSpace(sessionID) == "" {
		return "", time.Time{}, ErrInvalidToken
	}

	now := m.now().UTC()
	expiresAt := now.Add(m.ttl)
	tokenID, err := randomTokenID()
	if err != nil {
		return "", time.Time{}, err
	}

	claims := Claims{
		Subject:   subject,
		SessionID: sessionID,
		TokenID:   tokenID,
		Issuer:    m.issuer,
		Audience:  m.audience,
		Expires:   expiresAt.Unix(),
		IssuedAt:  now.Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := m.sign(encodedPayload)

	return encodedPayload + "." + signature, expiresAt, nil
}

func (m *TokenManager) ValidateAccessToken(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Claims{}, ErrInvalidToken
	}

	// Compare signatures before decoding claims so tampered payloads are rejected.
	expectedSignature := m.sign(parts[0])
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSignature)) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}

	if claims.Subject == "" || claims.SessionID == "" || claims.TokenID == "" || claims.Issuer != m.issuer || claims.Audience != m.audience {
		return Claims{}, ErrInvalidToken
	}
	now := m.now().UTC().Unix()
	if claims.Expires <= now || claims.IssuedAt > now {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}

func (m *TokenManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = fmt.Fprint(mac, payload)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomTokenID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}
