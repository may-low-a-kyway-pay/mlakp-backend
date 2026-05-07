package handlers

import (
	"testing"
	"time"

	"mlakp-backend/internal/debts"
)

func TestToDebtListItemResponseIncludesDisplayFields(t *testing.T) {
	now := time.Date(2026, 5, 7, 9, 15, 0, 0, time.UTC)

	got := toDebtListItemResponse(debts.ListItem{
		Debt: debts.Debt{
			ID:                   "debt-1",
			ExpenseID:            "expense-1",
			DebtorID:             "debtor-1",
			CreditorID:           "creditor-1",
			OriginalAmountMinor:  1250,
			RemainingAmountMinor: 750,
			Status:               debts.StatusAccepted,
			CreatedAt:            now,
			UpdatedAt:            now,
		},
		ExpenseTitle: "Dinner",
		DebtorName:   "Thomas",
		CreditorName: "Alice",
	})

	if got.ExpenseTitle == nil || *got.ExpenseTitle != "Dinner" {
		t.Fatalf("ExpenseTitle = %v, want Dinner", got.ExpenseTitle)
	}
	if got.DebtorName == nil || *got.DebtorName != "Thomas" {
		t.Fatalf("DebtorName = %v, want Thomas", got.DebtorName)
	}
	if got.CreditorName == nil || *got.CreditorName != "Alice" {
		t.Fatalf("CreditorName = %v, want Alice", got.CreditorName)
	}
	if got.OriginalAmount != "12.50" || got.RemainingAmount != "7.50" {
		t.Fatalf("amounts = %s/%s, want 12.50/7.50", got.OriginalAmount, got.RemainingAmount)
	}
}
