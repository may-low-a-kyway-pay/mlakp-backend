package notifications

import (
	"encoding/json"
	"time"
)

const (
	TypeExpenseCreated   = "expense.created"
	TypeDebtAccepted     = "debt.accepted"
	TypeDebtRejected     = "debt.rejected"
	TypeDebtResent       = "debt.resent"
	TypePaymentMarked    = "payment.marked"
	TypePaymentConfirmed = "payment.confirmed"
	TypePaymentRejected  = "payment.rejected"

	EntityExpense = "expense"
	EntityDebt    = "debt"
	EntityPayment = "payment"
)

type Notification struct {
	ID         string
	UserID     string
	Type       string
	Title      string
	Body       string
	EntityType string
	EntityID   string
	Metadata   []byte
	ReadAt     *time.Time
	CreatedAt  time.Time
}

func (n Notification) MarshalJSON() ([]byte, error) {
	type notificationJSON struct {
		ID         string          `json:"id"`
		UserID     string          `json:"user_id"`
		Type       string          `json:"type"`
		Title      string          `json:"title"`
		Body       string          `json:"body"`
		EntityType string          `json:"entity_type"`
		EntityID   string          `json:"entity_id"`
		Metadata   json.RawMessage `json:"metadata"`
		ReadAt     *string         `json:"read_at"`
		CreatedAt  string          `json:"created_at"`
	}

	metadata := json.RawMessage(n.Metadata)
	if len(metadata) == 0 {
		metadata = json.RawMessage("{}")
	}

	var readAt *string
	if n.ReadAt != nil {
		formatted := n.ReadAt.UTC().Format(time.RFC3339)
		readAt = &formatted
	}

	return json.Marshal(notificationJSON{
		ID:         n.ID,
		UserID:     n.UserID,
		Type:       n.Type,
		Title:      n.Title,
		Body:       n.Body,
		EntityType: n.EntityType,
		EntityID:   n.EntityID,
		Metadata:   metadata,
		ReadAt:     readAt,
		CreatedAt:  n.CreatedAt.UTC().Format(time.RFC3339),
	})
}

type CreateInput struct {
	UserID     string
	Type       string
	Title      string
	Body       string
	EntityType string
	EntityID   string
	Metadata   []byte
}

type ListInput struct {
	UserID string
	Limit  int32
}

type MarkReadInput struct {
	ID     string
	UserID string
}

type RealtimeEvent struct {
	Kind         string        `json:"kind"`
	Notification *Notification `json:"notification,omitempty"`
	UnreadCount  int64         `json:"unread_count"`
}
