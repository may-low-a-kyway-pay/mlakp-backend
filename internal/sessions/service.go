package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidSession      = errors.New("invalid session")
)

type Store interface {
	Create(ctx context.Context, userID, refreshTokenHash string, expiresAt time.Time) (Session, error)
	GetActiveByID(ctx context.Context, id string) (Session, error)
	RotateRefreshToken(ctx context.Context, oldRefreshTokenHash, newRefreshTokenHash string) (Session, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

type AccessTokenResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	Session      Session
	RefreshToken string
}

type Service struct {
	store      Store
	refreshTTL time.Duration
	now        func() time.Time
}

func NewService(store Store, refreshTTL time.Duration) *Service {
	return &Service{
		store:      store,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

func (s *Service) Create(ctx context.Context, userID string) (Session, string, error) {
	if strings.TrimSpace(userID) == "" {
		return Session{}, "", ErrInvalidSession
	}

	refreshToken, refreshTokenHash, err := newRefreshToken()
	if err != nil {
		return Session{}, "", err
	}

	session, err := s.store.Create(ctx, userID, refreshTokenHash, s.now().UTC().Add(s.refreshTTL))
	if err != nil {
		return Session{}, "", err
	}

	return session, refreshToken, nil
}

func (s *Service) ValidateAccessSession(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return ErrInvalidSession
	}

	if _, err := s.store.GetActiveByID(ctx, sessionID); err != nil {
		return ErrInvalidSession
	}

	return nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, string, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return Session{}, "", ErrInvalidRefreshToken
	}

	newToken, newTokenHash, err := newRefreshToken()
	if err != nil {
		return Session{}, "", err
	}

	// Rotation makes refresh tokens single-use; the old hash must stop matching.
	session, err := s.store.RotateRefreshToken(ctx, hashRefreshToken(refreshToken), newTokenHash)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return Session{}, "", ErrInvalidRefreshToken
		}
		return Session{}, "", err
	}

	return session, newToken, nil
}

func (s *Service) Revoke(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return ErrInvalidSession
	}

	return s.store.Revoke(ctx, sessionID)
}

func (s *Service) RevokeAllForUser(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidSession
	}

	return s.store.RevokeAllForUser(ctx, userID)
}

func (s *Service) CreateForUser(ctx context.Context, userID string) (Session, string, error) {
	return s.Create(ctx, userID)
}

func newRefreshToken() (string, string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", "", err
	}

	token := base64.RawURLEncoding.EncodeToString(bytes[:])
	return token, hashRefreshToken(token), nil
}

// hashRefreshToken stores only a deterministic server-side fingerprint of the opaque token.
func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
