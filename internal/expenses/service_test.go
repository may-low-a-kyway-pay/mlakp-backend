package expenses

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"mlakp-backend/internal/notifications"
)

func TestServiceCreateEqualSplit(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	_, err := service.Create(context.Background(), CreateInput{
		GroupID:     "group-1",
		Title:       " Dinner ",
		TotalAmount: "100.00",
		PaidBy:      "payer-1",
		SplitType:   SplitTypeEqual,
		Participants: []ParticipantInput{
			{UserID: "payer-1"},
			{UserID: "user-2"},
			{UserID: "user-3"},
		},
		CreatedBy: "creator-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if store.params.Title != "Dinner" {
		t.Fatalf("Title = %q, want Dinner", store.params.Title)
	}
	if store.params.Currency != CurrencyTHB {
		t.Fatalf("Currency = %q, want %q", store.params.Currency, CurrencyTHB)
	}
	if store.params.TotalMinor != 10000 {
		t.Fatalf("TotalMinor = %d, want 10000", store.params.TotalMinor)
	}
	wantShares := []int64{3333, 3333, 3334}
	for i, participant := range store.params.Participants {
		if participant.ShareMinor != wantShares[i] {
			t.Fatalf("participant[%d].ShareMinor = %d, want %d", i, participant.ShareMinor, wantShares[i])
		}
	}
}

func TestServiceCreateManualSplit(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	_, err := service.Create(context.Background(), CreateInput{
		GroupID:     "group-1",
		Title:       "Dinner",
		TotalAmount: "100.00",
		PaidBy:      "payer-1",
		SplitType:   SplitTypeManual,
		Participants: []ParticipantInput{
			{UserID: "payer-1", ShareAmount: stringPtr("40.00")},
			{UserID: "user-2", ShareAmount: stringPtr("60.00")},
		},
		CreatedBy: "creator-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got := store.params.Participants[1].ShareMinor; got != 6000 {
		t.Fatalf("manual share = %d, want 6000", got)
	}
}

func TestServiceCreateNotifiesDebtorsWithDebtActionTarget(t *testing.T) {
	store := &fakeStore{
		created: CreatedExpense{
			Expense: Expense{ID: "expense-1"},
			Debts: []Debt{
				{ID: "debt-1", DebtorID: "user-2"},
				{ID: "debt-2", DebtorID: "user-3"},
			},
		},
	}
	notifier := &fakeNotifier{}
	service := NewService(store, notifier)

	_, err := service.Create(context.Background(), validCreateInput(func(input *CreateInput) {
		input.Participants = []ParticipantInput{
			{UserID: "payer-1"},
			{UserID: "user-2"},
			{UserID: "user-3"},
		}
	}))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(notifier.inputs) != 2 {
		t.Fatalf("notifications = %d, want 2", len(notifier.inputs))
	}

	for i, input := range notifier.inputs {
		if input.Type != notifications.TypeExpenseCreated {
			t.Fatalf("notification[%d].Type = %q, want %q", i, input.Type, notifications.TypeExpenseCreated)
		}
		if input.EntityType != notifications.EntityDebt {
			t.Fatalf("notification[%d].EntityType = %q, want %q", i, input.EntityType, notifications.EntityDebt)
		}
		if input.EntityID != store.created.Debts[i].ID {
			t.Fatalf("notification[%d].EntityID = %q, want debt id %q", i, input.EntityID, store.created.Debts[i].ID)
		}

		var metadata map[string]string
		if err := json.Unmarshal(input.Metadata, &metadata); err != nil {
			t.Fatalf("notification[%d].Metadata invalid JSON: %v", i, err)
		}
		if metadata["expense_id"] != "expense-1" {
			t.Fatalf("notification[%d].Metadata expense_id = %q, want expense-1", i, metadata["expense_id"])
		}
	}
}

func TestServiceCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateInput
		wantErr error
	}{
		{name: "missing group", input: validCreateInput(func(input *CreateInput) { input.GroupID = " " }), wantErr: ErrInvalidGroupID},
		{name: "invalid title", input: validCreateInput(func(input *CreateInput) { input.Title = " " }), wantErr: ErrInvalidTitle},
		{name: "invalid amount", input: validCreateInput(func(input *CreateInput) { input.TotalAmount = "1.234" }), wantErr: ErrInvalidAmount},
		{name: "invalid currency", input: validCreateInput(func(input *CreateInput) { input.Currency = "USD" }), wantErr: ErrInvalidCurrency},
		{name: "missing payer", input: validCreateInput(func(input *CreateInput) { input.PaidBy = " " }), wantErr: ErrInvalidPayerID},
		{name: "invalid split type", input: validCreateInput(func(input *CreateInput) { input.SplitType = "weighted" }), wantErr: ErrInvalidSplitType},
		{name: "duplicate participant", input: validCreateInput(func(input *CreateInput) {
			input.Participants = append(input.Participants, ParticipantInput{UserID: "user-2"})
		}), wantErr: ErrDuplicateParticipant},
		{name: "no debtor participant", input: validCreateInput(func(input *CreateInput) {
			input.Participants = []ParticipantInput{{UserID: "payer-1"}}
		}), wantErr: ErrNoDebtorParticipant},
		{name: "manual missing share", input: validCreateInput(func(input *CreateInput) {
			input.SplitType = SplitTypeManual
		}), wantErr: ErrInvalidManualShare},
		{name: "manual mismatch", input: validCreateInput(func(input *CreateInput) {
			input.SplitType = SplitTypeManual
			input.Participants = []ParticipantInput{
				{UserID: "payer-1", ShareAmount: stringPtr("40.00")},
				{UserID: "user-2", ShareAmount: stringPtr("50.00")},
			}
		}), wantErr: ErrSplitMismatch},
		{name: "invalid receipt url", input: validCreateInput(func(input *CreateInput) {
			input.ReceiptURL = stringPtr("ftp://example.com/receipt")
		}), wantErr: ErrInvalidReceiptURL},
		{name: "invalid expense date", input: validCreateInput(func(input *CreateInput) {
			input.ExpenseDate = stringPtr("2026/05/03")
		}), wantErr: ErrInvalidExpenseDate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&fakeStore{})
			_, err := service.Create(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceGetValidatesAndTrimsInput(t *testing.T) {
	tests := []struct {
		name    string
		input   GetInput
		wantErr error
	}{
		{name: "missing expense", input: GetInput{ExpenseID: " ", UserID: "user-1"}, wantErr: ErrInvalidExpenseID},
		{name: "missing user", input: GetInput{ExpenseID: "expense-1", UserID: " "}, wantErr: ErrForbidden},
		{name: "valid", input: GetInput{ExpenseID: " expense-1 ", UserID: " user-1 "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			service := NewService(store)
			_, err := service.Get(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (store.expenseID != "expense-1" || store.userID != "user-1") {
				t.Fatalf("store IDs = (%q, %q), want (expense-1, user-1)", store.expenseID, store.userID)
			}
		})
	}
}

func TestServiceListByGroupValidatesAndTrimsInput(t *testing.T) {
	tests := []struct {
		name    string
		input   ListByGroupInput
		wantErr error
	}{
		{name: "missing group", input: ListByGroupInput{GroupID: " ", UserID: "user-1"}, wantErr: ErrInvalidGroupID},
		{name: "missing user", input: ListByGroupInput{GroupID: "group-1", UserID: " "}, wantErr: ErrForbidden},
		{name: "valid", input: ListByGroupInput{GroupID: " group-1 ", UserID: " user-1 "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			service := NewService(store)
			_, err := service.ListByGroup(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ListByGroup() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (store.groupID != "group-1" || store.userID != "user-1") {
				t.Fatalf("store IDs = (%q, %q), want (group-1, user-1)", store.groupID, store.userID)
			}
		})
	}
}

type fakeStore struct {
	params    createParams
	created   CreatedExpense
	expenseID string
	groupID   string
	userID    string
}

func (s *fakeStore) Create(_ context.Context, params createParams) (CreatedExpense, error) {
	s.params = params
	return s.created, nil
}

func (s *fakeStore) Get(_ context.Context, expenseID, userID string) (ExpenseDetails, error) {
	s.expenseID = expenseID
	s.userID = userID
	return ExpenseDetails{}, nil
}

func (s *fakeStore) ListByGroup(_ context.Context, groupID, userID string) ([]Expense, error) {
	s.groupID = groupID
	s.userID = userID
	return nil, nil
}

func validCreateInput(mutators ...func(*CreateInput)) CreateInput {
	input := CreateInput{
		GroupID:     "group-1",
		Title:       "Dinner",
		TotalAmount: "100.00",
		Currency:    CurrencyTHB,
		PaidBy:      "payer-1",
		SplitType:   SplitTypeEqual,
		Participants: []ParticipantInput{
			{UserID: "payer-1"},
			{UserID: "user-2"},
		},
		CreatedBy: "creator-1",
	}

	for _, mutate := range mutators {
		mutate(&input)
	}

	return input
}

func stringPtr(value string) *string {
	return &value
}

type fakeNotifier struct {
	inputs []notifications.CreateInput
}

func (n *fakeNotifier) Create(_ context.Context, input notifications.CreateInput) (notifications.Notification, error) {
	n.inputs = append(n.inputs, input)
	return notifications.Notification{}, nil
}
