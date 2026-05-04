package expenses

import "time"

const (
	SplitTypeEqual    = "equal"
	SplitTypeManual   = "manual"
	CurrencyTHB       = "THB"
	DebtStatusPending = "pending"
)

type CreateInput struct {
	GroupID      string
	Title        string
	Description  *string
	TotalAmount  string
	Currency     string
	PaidBy       string
	SplitType    string
	ReceiptURL   *string
	ExpenseDate  *string
	Participants []ParticipantInput
	CreatedBy    string
}

type GetInput struct {
	ExpenseID string
	UserID    string
}

type ListByGroupInput struct {
	GroupID string
	UserID  string
}

type ParticipantInput struct {
	UserID      string
	ShareAmount *string
}

type createParams struct {
	GroupID      string
	Title        string
	Description  *string
	TotalMinor   int64
	Currency     string
	PaidBy       string
	SplitType    string
	ReceiptURL   *string
	ExpenseDate  *time.Time
	Participants []participantShare
	CreatedBy    string
}

type participantShare struct {
	UserID     string
	ShareMinor int64
}

type CreatedExpense struct {
	Expense      Expense
	Participants []Participant
	Debts        []Debt
}

type ExpenseDetails struct {
	Expense      Expense
	Participants []Participant
	Debts        []Debt
}

type Expense struct {
	ID               string
	GroupID          string
	Title            string
	Description      *string
	TotalAmountMinor int64
	Currency         string
	PaidBy           string
	SplitType        string
	ReceiptURL       *string
	ExpenseDate      *time.Time
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Participant struct {
	ID               string
	ExpenseID        string
	UserID           string
	ShareAmountMinor int64
	CreatedAt        time.Time
}

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
