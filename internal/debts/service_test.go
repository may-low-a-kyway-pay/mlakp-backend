package debts

import (
	"context"
	"errors"
	"testing"
)

func TestServiceTransitionValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		debtID  string
		userID  string
		wantErr error
	}{
		{name: "missing debt", debtID: " ", userID: "user-1", wantErr: ErrInvalidDebtID},
		{name: "missing user", debtID: "debt-1", userID: " ", wantErr: ErrInvalidUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&fakeStore{})
			_, err := service.Transition(context.Background(), tt.debtID, tt.userID, TransitionAccept)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Transition() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceRejectTrimsInput(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	if _, err := service.Transition(context.Background(), " debt-1 ", " user-1 ", TransitionReject); err != nil {
		t.Fatalf("Transition() error = %v", err)
	}
	if store.debtID != "debt-1" {
		t.Fatalf("debtID = %q, want debt-1", store.debtID)
	}
	if store.userID != "user-1" {
		t.Fatalf("userID = %q, want user-1", store.userID)
	}
}

func TestServiceTransitionRejectsInvalidType(t *testing.T) {
	service := NewService(&fakeStore{})

	_, err := service.Transition(context.Background(), "debt-1", "user-1", "approve")
	if !errors.Is(err, ErrInvalidType) {
		t.Fatalf("Transition() error = %v, want %v", err, ErrInvalidType)
	}
}

func TestServiceReviewRejectedValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		input   ReviewRejectedInput
		wantErr error
	}{
		{name: "missing debt", input: ReviewRejectedInput{DebtID: " ", ReviewerID: "user-1"}, wantErr: ErrInvalidDebtID},
		{name: "missing reviewer", input: ReviewRejectedInput{DebtID: "debt-1", ReviewerID: " "}, wantErr: ErrInvalidUserID},
		{name: "invalid amount", input: ReviewRejectedInput{DebtID: "debt-1", ReviewerID: "user-1", Amount: stringPtr("1.234")}, wantErr: ErrInvalidAmount},
		{name: "zero amount", input: ReviewRejectedInput{DebtID: "debt-1", ReviewerID: "user-1", Amount: stringPtr("0")}, wantErr: ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&fakeStore{})
			_, err := service.ReviewRejected(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ReviewRejected() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceReviewRejectedTrimsAndParsesInput(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	if _, err := service.ReviewRejected(context.Background(), ReviewRejectedInput{
		DebtID:     " debt-1 ",
		ReviewerID: " owner-1 ",
		Amount:     stringPtr("12.50"),
	}); err != nil {
		t.Fatalf("ReviewRejected() error = %v", err)
	}
	if store.reviewParams.DebtID != "debt-1" {
		t.Fatalf("DebtID = %q, want debt-1", store.reviewParams.DebtID)
	}
	if store.reviewParams.ReviewerID != "owner-1" {
		t.Fatalf("ReviewerID = %q, want owner-1", store.reviewParams.ReviewerID)
	}
	if store.reviewParams.AmountMinor == nil || *store.reviewParams.AmountMinor != 1250 {
		t.Fatalf("AmountMinor = %v, want 1250", store.reviewParams.AmountMinor)
	}
}

func TestServiceListValidatesAndTrimsInput(t *testing.T) {
	tests := []struct {
		name    string
		input   ListInput
		wantErr error
	}{
		{name: "missing user", input: ListInput{UserID: " "}, wantErr: ErrInvalidUserID},
		{name: "valid", input: ListInput{UserID: " user-1 "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			service := NewService(store)
			_, err := service.List(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("List() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && store.userID != "user-1" {
				t.Fatalf("userID = %q, want user-1", store.userID)
			}
		})
	}
}

type fakeStore struct {
	debtID       string
	userID       string
	reviewParams ReviewRejectedParams
}

func (s *fakeStore) Accept(_ context.Context, debtID, debtorID string) (Debt, error) {
	s.debtID = debtID
	s.userID = debtorID
	return Debt{}, nil
}

func (s *fakeStore) Reject(_ context.Context, debtID, debtorID string) (Debt, error) {
	s.debtID = debtID
	s.userID = debtorID
	return Debt{}, nil
}

func (s *fakeStore) ReviewRejected(_ context.Context, params ReviewRejectedParams) (Debt, error) {
	s.reviewParams = params
	return Debt{}, nil
}

func (s *fakeStore) ListForUser(_ context.Context, userID string) ([]Debt, error) {
	s.userID = userID
	return nil, nil
}

func stringPtr(value string) *string {
	return &value
}
