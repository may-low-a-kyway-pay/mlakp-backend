package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestServiceCreateValidatesControlledEventShape(t *testing.T) {
	service := NewService(&notificationStore{}, nil)

	tests := []struct {
		name    string
		input   CreateInput
		wantErr error
	}{
		{
			name: "unknown type",
			input: CreateInput{
				UserID:     "user-1",
				Type:       "unknown",
				Title:      "Title",
				Body:       "Body",
				EntityType: EntityDebt,
				EntityID:   "debt-1",
			},
			wantErr: ErrInvalidType,
		},
		{
			name: "unknown entity type",
			input: CreateInput{
				UserID:     "user-1",
				Type:       TypeDebtAccepted,
				Title:      "Title",
				Body:       "Body",
				EntityType: "unknown",
				EntityID:   "debt-1",
			},
			wantErr: ErrInvalidEntityType,
		},
		{
			name: "invalid metadata",
			input: CreateInput{
				UserID:     "user-1",
				Type:       TypeDebtAccepted,
				Title:      "Title",
				Body:       "Body",
				EntityType: EntityDebt,
				EntityID:   "debt-1",
				Metadata:   []byte("{"),
			},
			wantErr: ErrInvalidMetadata,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceCreatePublishesCreatedEvent(t *testing.T) {
	store := &notificationStore{
		notification: Notification{
			ID:         "notification-1",
			UserID:     "user-1",
			Type:       TypePaymentMarked,
			Title:      "Payment waiting for confirmation",
			Body:       "A payment was submitted and needs your review.",
			EntityType: EntityPayment,
			EntityID:   "payment-1",
			Metadata:   []byte("{}"),
			CreatedAt:  time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC),
		},
		unreadCount: 3,
	}
	publisher := &notificationPublisher{}
	service := NewService(store, publisher)

	notification, err := service.Create(context.Background(), CreateInput{
		UserID:     " user-1 ",
		Type:       TypePaymentMarked,
		Title:      " Payment waiting for confirmation ",
		Body:       " A payment was submitted and needs your review. ",
		EntityType: EntityPayment,
		EntityID:   " payment-1 ",
	})
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}

	if notification.ID != "notification-1" {
		t.Fatalf("Notification ID = %q, want notification-1", notification.ID)
	}
	if store.created.Type != TypePaymentMarked || store.created.Title != "Payment waiting for confirmation" {
		t.Fatalf("created input was not normalized: %+v", store.created)
	}
	if publisher.userID != "user-1" {
		t.Fatalf("published userID = %q, want user-1", publisher.userID)
	}
	if publisher.event.Kind != "notification.created" {
		t.Fatalf("event kind = %q, want notification.created", publisher.event.Kind)
	}
	if publisher.event.Notification == nil || publisher.event.Notification.ID != "notification-1" {
		t.Fatalf("event notification = %+v, want notification-1", publisher.event.Notification)
	}
	if publisher.event.UnreadCount != 3 {
		t.Fatalf("event unread count = %d, want 3", publisher.event.UnreadCount)
	}
}

func TestRealtimeEventMarshalUsesAPIShape(t *testing.T) {
	readAt := time.Date(2026, 5, 12, 10, 5, 0, 0, time.UTC)
	createdAt := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)

	payload, err := json.Marshal(RealtimeEvent{
		Kind: "notification.read",
		Notification: &Notification{
			ID:         "notification-1",
			UserID:     "user-1",
			Type:       TypeDebtAccepted,
			Title:      "Expense accepted",
			Body:       "A shared expense was accepted.",
			EntityType: EntityDebt,
			EntityID:   "debt-1",
			Metadata:   []byte(`{"debt_id":"debt-1"}`),
			ReadAt:     &readAt,
			CreatedAt:  createdAt,
		},
		UnreadCount: 2,
	})
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var got struct {
		Kind         string `json:"kind"`
		UnreadCount  int64  `json:"unread_count"`
		Notification struct {
			ID         string         `json:"id"`
			UserID     string         `json:"user_id"`
			EntityType string         `json:"entity_type"`
			EntityID   string         `json:"entity_id"`
			Metadata   map[string]any `json:"metadata"`
			ReadAt     string         `json:"read_at"`
			CreatedAt  string         `json:"created_at"`
		} `json:"notification"`
	}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("Unmarshal error = %v; payload=%s", err, payload)
	}

	if got.Kind != "notification.read" || got.UnreadCount != 2 {
		t.Fatalf("event envelope = %+v, want notification.read unread_count=2", got)
	}
	if got.Notification.ID != "notification-1" || got.Notification.UserID != "user-1" {
		t.Fatalf("notification identity = %+v, want snake_case id/user_id", got.Notification)
	}
	if got.Notification.EntityType != EntityDebt || got.Notification.EntityID != "debt-1" {
		t.Fatalf("notification entity = %+v, want debt/debt-1", got.Notification)
	}
	if got.Notification.Metadata["debt_id"] != "debt-1" {
		t.Fatalf("metadata = %+v, want debt_id object", got.Notification.Metadata)
	}
	if got.Notification.ReadAt != "2026-05-12T10:05:00Z" || got.Notification.CreatedAt != "2026-05-12T10:00:00Z" {
		t.Fatalf("timestamps = %s/%s, want RFC3339 UTC", got.Notification.ReadAt, got.Notification.CreatedAt)
	}
}

type notificationStore struct {
	created      CreateInput
	notification Notification
	unreadCount  int64
}

func (s *notificationStore) Create(_ context.Context, input CreateInput) (Notification, error) {
	s.created = input
	return s.notification, nil
}

func (s *notificationStore) ListForUser(_ context.Context, _ ListInput) ([]Notification, int64, error) {
	return []Notification{s.notification}, s.unreadCount, nil
}

func (s *notificationStore) MarkRead(_ context.Context, _ MarkReadInput) (Notification, int64, error) {
	return s.notification, s.unreadCount, nil
}

func (s *notificationStore) MarkAllRead(_ context.Context, _ string) (int64, error) {
	return s.unreadCount, nil
}

type notificationPublisher struct {
	userID string
	event  RealtimeEvent
}

func (p *notificationPublisher) Publish(userID string, event RealtimeEvent) {
	p.userID = userID
	p.event = event
}
