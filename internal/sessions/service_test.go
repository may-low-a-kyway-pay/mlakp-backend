package sessions

import (
	"context"
	"testing"
	"time"
)

func TestServiceCreateStoresRefreshTokenHash(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, 30*24*time.Hour)
	service.now = func() time.Time {
		return time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	}

	session, refreshToken, err := service.Create(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if session.ID != "session-1" {
		t.Fatalf("session.ID = %q, want session-1", session.ID)
	}
	if refreshToken == "" {
		t.Fatal("refreshToken is empty")
	}
	if store.createdRefreshTokenHash == "" {
		t.Fatal("createdRefreshTokenHash is empty")
	}
	if store.createdRefreshTokenHash == refreshToken {
		t.Fatal("refresh token was stored without hashing")
	}
	if store.createdExpiresAt != service.now().Add(30*24*time.Hour) {
		t.Fatalf("createdExpiresAt = %s, want refresh ttl", store.createdExpiresAt)
	}
}

func TestServiceRefreshRotatesRefreshToken(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, 30*24*time.Hour)

	session, refreshToken, err := service.Create(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	refreshedSession, rotatedRefreshToken, err := service.Refresh(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshedSession.ID != session.ID {
		t.Fatalf("refreshedSession.ID = %q, want %q", refreshedSession.ID, session.ID)
	}
	if rotatedRefreshToken == refreshToken {
		t.Fatal("Refresh() returned the same refresh token")
	}
	if store.rotatedOldHash == "" || store.rotatedNewHash == "" {
		t.Fatal("refresh token hashes were not recorded")
	}
	if store.rotatedOldHash == store.rotatedNewHash {
		t.Fatal("refresh token hash did not rotate")
	}
}

func TestServiceRefreshRejectsRotatedRefreshToken(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, 30*24*time.Hour)

	_, refreshToken, err := service.Create(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, _, err := service.Refresh(context.Background(), refreshToken); err != nil {
		t.Fatalf("Refresh() first error = %v", err)
	}
	if _, _, err := service.Refresh(context.Background(), refreshToken); err != ErrInvalidRefreshToken {
		t.Fatalf("Refresh() second error = %v, want %v", err, ErrInvalidRefreshToken)
	}
}

func TestServiceRevokePreventsAccessValidation(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, 30*24*time.Hour)

	session, _, err := service.Create(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := service.Revoke(context.Background(), session.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if err := service.ValidateAccessSession(context.Background(), session.ID); err != ErrInvalidSession {
		t.Fatalf("ValidateAccessSession() error = %v, want %v", err, ErrInvalidSession)
	}
}

type fakeStore struct {
	sessions                map[string]Session
	refreshHashToSessionID  map[string]string
	createdRefreshTokenHash string
	createdExpiresAt        time.Time
	rotatedOldHash          string
	rotatedNewHash          string
}

func (s *fakeStore) Create(_ context.Context, userID, refreshTokenHash string, expiresAt time.Time) (Session, error) {
	s.init()
	session := Session{
		ID:        "session-1",
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,
	}
	s.sessions[session.ID] = session
	s.refreshHashToSessionID[refreshTokenHash] = session.ID
	s.createdRefreshTokenHash = refreshTokenHash
	s.createdExpiresAt = expiresAt
	return session, nil
}

func (s *fakeStore) GetActiveByID(_ context.Context, id string) (Session, error) {
	s.init()
	session, ok := s.sessions[id]
	if !ok || session.RevokedAt != nil {
		return Session{}, ErrInvalidSession
	}

	return session, nil
}

func (s *fakeStore) RotateRefreshToken(_ context.Context, oldRefreshTokenHash, newRefreshTokenHash string) (Session, error) {
	s.init()
	sessionID, ok := s.refreshHashToSessionID[oldRefreshTokenHash]
	if !ok {
		return Session{}, ErrInvalidRefreshToken
	}
	delete(s.refreshHashToSessionID, oldRefreshTokenHash)
	s.refreshHashToSessionID[newRefreshTokenHash] = sessionID
	s.rotatedOldHash = oldRefreshTokenHash
	s.rotatedNewHash = newRefreshTokenHash
	return s.sessions[sessionID], nil
}

func (s *fakeStore) Revoke(_ context.Context, id string) error {
	s.init()
	session := s.sessions[id]
	now := time.Now().UTC()
	session.RevokedAt = &now
	s.sessions[id] = session
	return nil
}

func (s *fakeStore) init() {
	if s.sessions == nil {
		s.sessions = make(map[string]Session)
	}
	if s.refreshHashToSessionID == nil {
		s.refreshHashToSessionID = make(map[string]string)
	}
}
