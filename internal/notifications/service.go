package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrInvalidNotificationID = errors.New("notification id is invalid")
	ErrInvalidUserID         = errors.New("notification user id is invalid")
	ErrInvalidType           = errors.New("notification type is invalid")
	ErrInvalidTitle          = errors.New("notification title is invalid")
	ErrInvalidBody           = errors.New("notification body is invalid")
	ErrInvalidEntityType     = errors.New("notification entity type is invalid")
	ErrInvalidEntityID       = errors.New("notification entity id is invalid")
	ErrInvalidMetadata       = errors.New("notification metadata is invalid")
	ErrNotFound              = errors.New("notification not found")
)

type Store interface {
	Create(ctx context.Context, input CreateInput) (Notification, error)
	ListForUser(ctx context.Context, input ListInput) ([]Notification, int64, error)
	MarkRead(ctx context.Context, input MarkReadInput) (Notification, int64, error)
	MarkAllRead(ctx context.Context, userID string) (int64, error)
}

type Publisher interface {
	Publish(userID string, event RealtimeEvent)
}

type Service struct {
	store     Store
	publisher Publisher
}

func NewService(store Store, publisher Publisher) *Service {
	return &Service{store: store, publisher: publisher}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Notification, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return Notification{}, err
	}

	notification, err := s.store.Create(ctx, normalized)
	if err != nil {
		return Notification{}, err
	}

	_, unreadCount, err := s.store.ListForUser(ctx, ListInput{UserID: notification.UserID, Limit: 1})
	if err != nil {
		return Notification{}, err
	}

	s.publish(notification.UserID, RealtimeEvent{
		Kind:         "notification.created",
		Notification: &notification,
		UnreadCount:  unreadCount,
	})

	return notification, nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]Notification, int64, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return nil, 0, ErrInvalidUserID
	}

	return s.store.ListForUser(ctx, ListInput{UserID: userID, Limit: input.Limit})
}

func (s *Service) MarkRead(ctx context.Context, input MarkReadInput) (Notification, int64, error) {
	notificationID := strings.TrimSpace(input.ID)
	if notificationID == "" {
		return Notification{}, 0, ErrInvalidNotificationID
	}
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return Notification{}, 0, ErrInvalidUserID
	}

	notification, unreadCount, err := s.store.MarkRead(ctx, MarkReadInput{ID: notificationID, UserID: userID})
	if err != nil {
		return Notification{}, 0, err
	}

	s.publish(userID, RealtimeEvent{
		Kind:         "notification.read",
		Notification: &notification,
		UnreadCount:  unreadCount,
	})

	return notification, unreadCount, nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, ErrInvalidUserID
	}

	unreadCount, err := s.store.MarkAllRead(ctx, userID)
	if err != nil {
		return 0, err
	}

	s.publish(userID, RealtimeEvent{Kind: "notifications.read_all", UnreadCount: unreadCount})
	return unreadCount, nil
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	normalized := CreateInput{
		UserID:     strings.TrimSpace(input.UserID),
		Type:       strings.TrimSpace(input.Type),
		Title:      strings.TrimSpace(input.Title),
		Body:       strings.TrimSpace(input.Body),
		EntityType: strings.TrimSpace(input.EntityType),
		EntityID:   strings.TrimSpace(input.EntityID),
		Metadata:   input.Metadata,
	}

	switch {
	case normalized.UserID == "":
		return CreateInput{}, ErrInvalidUserID
	case !validNotificationType(normalized.Type):
		return CreateInput{}, ErrInvalidType
	case normalized.Title == "" || len(normalized.Title) > 160:
		return CreateInput{}, ErrInvalidTitle
	case normalized.Body == "" || len(normalized.Body) > 500:
		return CreateInput{}, ErrInvalidBody
	case !validEntityType(normalized.EntityType):
		return CreateInput{}, ErrInvalidEntityType
	case normalized.EntityID == "":
		return CreateInput{}, ErrInvalidEntityID
	}

	if len(normalized.Metadata) == 0 {
		normalized.Metadata = []byte("{}")
	}
	if !json.Valid(normalized.Metadata) {
		return CreateInput{}, ErrInvalidMetadata
	}

	return normalized, nil
}

func validNotificationType(value string) bool {
	switch value {
	case TypeExpenseCreated, TypeDebtAccepted, TypeDebtRejected, TypeDebtResent, TypePaymentMarked, TypePaymentConfirmed, TypePaymentRejected:
		return true
	default:
		return false
	}
}

func validEntityType(value string) bool {
	switch value {
	case EntityExpense, EntityDebt, EntityPayment:
		return true
	default:
		return false
	}
}

func (s *Service) publish(userID string, event RealtimeEvent) {
	if s.publisher != nil {
		s.publisher.Publish(userID, event)
	}
}
