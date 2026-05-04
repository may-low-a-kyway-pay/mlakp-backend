package debts

import "time"

const (
	TransitionAccept = "accept"
	TransitionReject = "reject"

	StatusPending  = "pending"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
)

type Debt struct {
	ID                   string
	ExpenseID            string
	DebtorID             string
	CreditorID           string
	OriginalAmountMinor  int64
	RemainingAmountMinor int64
	Status               string
	AcceptedAt           *time.Time
	RejectedAt           *time.Time
	SettledAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ReviewRejectedInput struct {
	DebtID     string
	ReviewerID string
	Amount     *string
}

type ListInput struct {
	UserID string
}

type ReviewRejectedParams struct {
	DebtID      string
	ReviewerID  string
	AmountMinor *int64
}
