package payments

import (
	"context"
	"errors"
	"testing"
)

func TestServiceMarkValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		input   MarkInput
		wantErr error
	}{
		{name: "missing debt", input: MarkInput{DebtID: " ", UserID: "user-1", Amount: "10.00"}, wantErr: ErrInvalidDebtID},
		{name: "missing user", input: MarkInput{DebtID: "debt-1", UserID: " ", Amount: "10.00"}, wantErr: ErrInvalidUserID},
		{name: "invalid amount", input: MarkInput{DebtID: "debt-1", UserID: "user-1", Amount: "1.234"}, wantErr: ErrInvalidAmount},
		{name: "zero amount", input: MarkInput{DebtID: "debt-1", UserID: "user-1", Amount: "0"}, wantErr: ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&fakeStore{})
			_, err := service.Mark(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Mark() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceMarkTrimsAndParsesInput(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)
	note := " rent "

	if _, err := service.Mark(context.Background(), MarkInput{
		DebtID: " debt-1 ",
		UserID: " user-1 ",
		Amount: "12.50",
		Note:   &note,
	}); err != nil {
		t.Fatalf("Mark() error = %v", err)
	}
	if store.markParams.DebtID != "debt-1" {
		t.Fatalf("DebtID = %q, want debt-1", store.markParams.DebtID)
	}
	if store.markParams.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", store.markParams.UserID)
	}
	if store.markParams.AmountMinor != 1250 {
		t.Fatalf("AmountMinor = %d, want 1250", store.markParams.AmountMinor)
	}
	if store.markParams.Note == nil || *store.markParams.Note != "rent" {
		t.Fatalf("Note = %v, want rent", store.markParams.Note)
	}
}

func TestServiceReviewValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		input   ReviewInput
		call    func(*Service, context.Context, ReviewInput) (Payment, error)
		wantErr error
	}{
		{name: "confirm missing payment", input: ReviewInput{PaymentID: " ", UserID: "user-1"}, call: (*Service).Confirm, wantErr: ErrInvalidPaymentID},
		{name: "confirm missing user", input: ReviewInput{PaymentID: "payment-1", UserID: " "}, call: (*Service).Confirm, wantErr: ErrInvalidUserID},
		{name: "reject missing payment", input: ReviewInput{PaymentID: " ", UserID: "user-1"}, call: (*Service).Reject, wantErr: ErrInvalidPaymentID},
		{name: "reject missing user", input: ReviewInput{PaymentID: "payment-1", UserID: " "}, call: (*Service).Reject, wantErr: ErrInvalidUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&fakeStore{})
			_, err := tt.call(service, context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("review call error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceReviewDispatchesByType(t *testing.T) {
	tests := []struct {
		name     string
		input    ReviewInput
		wantCall string
		wantErr  error
	}{
		{name: "confirm", input: ReviewInput{PaymentID: " payment-1 ", UserID: " user-1 ", Type: " confirm "}, wantCall: ReviewTypeConfirm},
		{name: "reject", input: ReviewInput{PaymentID: " payment-1 ", UserID: " user-1 ", Type: " reject "}, wantCall: ReviewTypeReject},
		{name: "invalid type", input: ReviewInput{PaymentID: "payment-1", UserID: "user-1", Type: "accept"}, wantErr: ErrInvalidReviewType},
		{name: "missing type", input: ReviewInput{PaymentID: "payment-1", UserID: "user-1"}, wantErr: ErrInvalidReviewType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			service := NewService(store)
			_, err := service.Review(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Review() error = %v, want %v", err, tt.wantErr)
			}
			if store.call != tt.wantCall {
				t.Fatalf("store call = %q, want %q", store.call, tt.wantCall)
			}
			if tt.wantErr == nil && (store.paymentID != "payment-1" || store.userID != "user-1") {
				t.Fatalf("store identity = (%q, %q), want (payment-1, user-1)", store.paymentID, store.userID)
			}
		})
	}
}

type fakeStore struct {
	markParams markParams
	paymentID  string
	userID     string
	call       string
}

func (s *fakeStore) Mark(_ context.Context, params markParams) (Payment, error) {
	s.markParams = params
	return Payment{}, nil
}

func (s *fakeStore) Confirm(_ context.Context, paymentID, userID string) (Payment, error) {
	s.paymentID = paymentID
	s.userID = userID
	s.call = ReviewTypeConfirm
	return Payment{}, nil
}

func (s *fakeStore) Reject(_ context.Context, paymentID, userID string) (Payment, error) {
	s.paymentID = paymentID
	s.userID = userID
	s.call = ReviewTypeReject
	return Payment{}, nil
}
