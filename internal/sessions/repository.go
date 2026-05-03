package sessions

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"mlakp-backend/internal/postgres/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, userID, refreshTokenHash string, expiresAt time.Time) (Session, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return Session{}, ErrInvalidSession
	}

	session, err := r.queries.CreateAuthSession(ctx, sqlc.CreateAuthSessionParams{
		UserID:           userUUID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        pgtype.Timestamptz{Time: expiresAt.UTC(), Valid: true},
	})
	if err != nil {
		return Session{}, err
	}

	return fromSQLC(session), nil
}

func (r *Repository) GetActiveByID(ctx context.Context, id string) (Session, error) {
	sessionID, err := parseUUID(id)
	if err != nil {
		return Session{}, ErrInvalidSession
	}

	session, err := r.queries.GetActiveAuthSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrInvalidSession
		}
		return Session{}, err
	}

	return fromSQLC(session), nil
}

func (r *Repository) RotateRefreshToken(ctx context.Context, oldRefreshTokenHash, newRefreshTokenHash string) (Session, error) {
	// The WHERE clause enforces active-session and old-token checks atomically.
	session, err := r.queries.RotateRefreshToken(ctx, sqlc.RotateRefreshTokenParams{
		RefreshTokenHash:   oldRefreshTokenHash,
		RefreshTokenHash_2: newRefreshTokenHash,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrInvalidRefreshToken
		}
		return Session{}, err
	}

	return fromSQLC(session), nil
}

func (r *Repository) Revoke(ctx context.Context, id string) error {
	sessionID, err := parseUUID(id)
	if err != nil {
		return ErrInvalidSession
	}

	return r.queries.RevokeAuthSession(ctx, sessionID)
}

func fromSQLC(session sqlc.AuthSession) Session {
	var revokedAt *time.Time
	if session.RevokedAt.Valid {
		revokedAt = &session.RevokedAt.Time
	}

	var lastUsedAt *time.Time
	if session.LastUsedAt.Valid {
		lastUsedAt = &session.LastUsedAt.Time
	}

	return Session{
		ID:         formatUUID(session.ID),
		UserID:     formatUUID(session.UserID),
		CreatedAt:  session.CreatedAt.Time,
		ExpiresAt:  session.ExpiresAt.Time,
		RevokedAt:  revokedAt,
		LastUsedAt: lastUsedAt,
	}
}

func parseUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	if !uuid.Valid {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid")
	}

	return uuid, nil
}

func formatUUID(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}

	encoded := hex.EncodeToString(uuid.Bytes[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}
