package payments

import "time"

const (
	StatusPendingConfirmation = "pending_confirmation"
	StatusConfirmed           = "confirmed"
	StatusRejected            = "rejected"

	ReviewTypeConfirm = "confirm"
	ReviewTypeReject  = "reject"

	DebtStatusAccepted         = "accepted"
	DebtStatusPartiallySettled = "partially_settled"
)

type Payment struct {
	ID          string
	DebtID      string
	PaidBy      string
	ReceivedBy  string
	AmountMinor int64
	Status      string
	Note        *string
	ConfirmedAt *time.Time
	RejectedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MarkInput struct {
	DebtID string
	UserID string
	Amount string
	Note   *string
}

type markParams struct {
	DebtID      string
	UserID      string
	AmountMinor int64
	Note        *string
}

type ReviewInput struct {
	PaymentID string
	UserID    string
	Type      string
}
