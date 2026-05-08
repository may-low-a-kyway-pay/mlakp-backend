package payments

import "time"

const (
	StatusPendingConfirmation = "pending_confirmation"
	StatusConfirmed           = "confirmed"
	StatusRejected            = "rejected"

	TypeReceived = "received"
	TypeSent     = "sent"
	TypeAll      = "all"

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

type ListItem struct {
	Payment
	ExpenseID                string
	ExpenseTitle             string
	PaidByName               string
	ReceivedByName           string
	DebtRemainingAmountMinor int64
	DebtStatus               string
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

type ListInput struct {
	UserID string
	Status string
	Type   string
}

type ListFilters struct {
	UserID string
	Status *string
	Type   *string
}

type ReviewInput struct {
	PaymentID string
	UserID    string
	Type      string
}
