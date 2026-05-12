package notifications

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

func (r *Repository) Create(ctx context.Context, input CreateInput) (Notification, error) {
	userID, err := parseUUID(input.UserID)
	if err != nil {
		return Notification{}, ErrInvalidUserID
	}
	entityID, err := parseUUID(input.EntityID)
	if err != nil {
		return Notification{}, ErrInvalidEntityID
	}

	metadata := input.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}

	notification, err := r.queries.CreateNotification(ctx, sqlc.CreateNotificationParams{
		UserID:     userID,
		Type:       input.Type,
		Title:      input.Title,
		Body:       input.Body,
		EntityType: input.EntityType,
		EntityID:   entityID,
		Metadata:   metadata,
	})
	if err != nil {
		return Notification{}, err
	}

	return notificationFromSQLC(notification), nil
}

func (r *Repository) ListForUser(ctx context.Context, input ListInput) ([]Notification, int64, error) {
	userID, err := parseUUID(input.UserID)
	if err != nil {
		return nil, 0, ErrInvalidUserID
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := r.queries.ListNotificationsForUser(ctx, sqlc.ListNotificationsForUserParams{
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		return nil, 0, err
	}

	unreadCount, err := r.queries.CountUnreadNotificationsForUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	notifications := make([]Notification, 0, len(rows))
	for _, row := range rows {
		notifications = append(notifications, notificationFromSQLC(row))
	}

	return notifications, unreadCount, nil
}

func (r *Repository) MarkRead(ctx context.Context, input MarkReadInput) (Notification, int64, error) {
	notificationID, err := parseUUID(input.ID)
	if err != nil {
		return Notification{}, 0, ErrInvalidNotificationID
	}
	userID, err := parseUUID(input.UserID)
	if err != nil {
		return Notification{}, 0, ErrInvalidUserID
	}

	notification, err := r.queries.MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{
		ID:     notificationID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, 0, ErrNotFound
		}
		return Notification{}, 0, err
	}

	unreadCount, err := r.queries.CountUnreadNotificationsForUser(ctx, userID)
	if err != nil {
		return Notification{}, 0, err
	}

	return notificationFromSQLC(notification), unreadCount, nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return 0, ErrInvalidUserID
	}

	if err := r.queries.MarkAllNotificationsRead(ctx, userUUID); err != nil {
		return 0, err
	}

	return r.queries.CountUnreadNotificationsForUser(ctx, userUUID)
}

func notificationFromSQLC(notification sqlc.Notification) Notification {
	return Notification{
		ID:         formatUUID(notification.ID),
		UserID:     formatUUID(notification.UserID),
		Type:       notification.Type,
		Title:      notification.Title,
		Body:       notification.Body,
		EntityType: notification.EntityType,
		EntityID:   formatUUID(notification.EntityID),
		Metadata:   notification.Metadata,
		ReadAt:     timestamptzPtr(notification.ReadAt),
		CreatedAt:  notification.CreatedAt.Time,
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

func formatUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}

	encoded := hex.EncodeToString(value.Bytes[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
