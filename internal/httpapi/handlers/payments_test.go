package handlers

import (
	"testing"
	"time"

	"mlakp-backend/internal/payments"
)

func TestToPaymentListItemResponseIncludesDisplayFields(t *testing.T) {
	now := time.Date(2026, 5, 8, 9, 15, 0, 0, time.UTC)
	note := "bank transfer"

	got := toPaymentListItemResponse(payments.ListItem{
		Payment: payments.Payment{
			ID:          "payment-1",
			DebtID:      "debt-1",
			PaidBy:      "debtor-1",
			ReceivedBy:  "creditor-1",
			AmountMinor: 1250,
			Status:      payments.StatusPendingConfirmation,
			Note:        &note,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		ExpenseID:                "expense-1",
		ExpenseTitle:             "Dinner",
		PaidByName:               "Thomas",
		ReceivedByName:           "Alice",
		DebtRemainingAmountMinor: 5000,
		DebtStatus:               payments.DebtStatusAccepted,
	})

	if got.ExpenseTitle != "Dinner" || got.PaidByName != "Thomas" || got.ReceivedByName != "Alice" {
		t.Fatalf("display fields = %+v, want expense and user names", got)
	}
	if got.Amount != "12.50" || got.DebtRemainingAmount != "50.00" {
		t.Fatalf("amounts = %s/%s, want 12.50/50.00", got.Amount, got.DebtRemainingAmount)
	}
	if got.Note == nil || *got.Note != "bank transfer" {
		t.Fatalf("Note = %v, want bank transfer", got.Note)
	}
}
