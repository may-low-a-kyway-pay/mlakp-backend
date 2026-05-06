package handlers

import (
	"testing"
	"time"

	"mlakp-backend/internal/dashboard"
)

func TestToDashboardResponseIncludesUnsettledBalances(t *testing.T) {
	currentUserID := "00000000-0000-0000-0000-000000000001"
	creditorID := "00000000-0000-0000-0000-000000000002"
	debtorID := "00000000-0000-0000-0000-000000000003"
	updatedAt := time.Date(2026, 5, 6, 12, 30, 0, 0, time.UTC)

	got := toDashboardResponse(dashboard.Snapshot{
		Totals: dashboard.Totals{
			YouOwe: dashboard.DashboardAmount{AmountMinor: 1250, DebtCount: 1},
			YouGet: dashboard.DashboardAmount{AmountMinor: 2000, DebtCount: 2},
		},
		UnsettledBalances: []dashboard.UnsettledBalance{
			{
				ID:                   "debt-owed",
				ExpenseID:            "expense-owed",
				ExpenseTitle:         "Dinner",
				DebtorID:             currentUserID,
				DebtorName:           "Thomas",
				CreditorID:           creditorID,
				CreditorName:         "Alice",
				RemainingAmountMinor: 1250,
				Status:               "accepted",
				UpdatedAt:            updatedAt,
			},
			{
				ID:                   "debt-receivable",
				ExpenseID:            "expense-receivable",
				ExpenseTitle:         "Taxi",
				DebtorID:             debtorID,
				DebtorName:           "Bob",
				CreditorID:           currentUserID,
				CreditorName:         "Thomas",
				RemainingAmountMinor: 2000,
				Status:               "partially_settled",
				UpdatedAt:            updatedAt,
			},
		},
	}, currentUserID)

	if got.YouOwe.Amount != "12.50" || got.YouOwe.AmountMinor != 1250 || got.YouOwe.DebtCount != 1 {
		t.Fatalf("YouOwe = %+v, want formatted amount 12.50, minor 1250, count 1", got.YouOwe)
	}
	if got.YouGet.Amount != "20.00" || got.YouGet.AmountMinor != 2000 || got.YouGet.DebtCount != 2 {
		t.Fatalf("YouGet = %+v, want formatted amount 20.00, minor 2000, count 2", got.YouGet)
	}
	if len(got.UnsettledBalances) != 2 {
		t.Fatalf("len(UnsettledBalances) = %d, want 2", len(got.UnsettledBalances))
	}

	owed := got.UnsettledBalances[0]
	if owed.Type != "owed" {
		t.Fatalf("owed.Type = %q, want owed", owed.Type)
	}
	if owed.OtherUser.ID != creditorID || owed.OtherUser.Name != "Alice" {
		t.Fatalf("owed.OtherUser = %+v, want creditor Alice", owed.OtherUser)
	}
	if owed.RemainingAmount != "12.50" || owed.RemainingMinor != 1250 {
		t.Fatalf("owed remaining = %s/%d, want 12.50/1250", owed.RemainingAmount, owed.RemainingMinor)
	}

	receivable := got.UnsettledBalances[1]
	if receivable.Type != "receivable" {
		t.Fatalf("receivable.Type = %q, want receivable", receivable.Type)
	}
	if receivable.OtherUser.ID != debtorID || receivable.OtherUser.Name != "Bob" {
		t.Fatalf("receivable.OtherUser = %+v, want debtor Bob", receivable.OtherUser)
	}
	if receivable.UpdatedAt != "2026-05-06T12:30:00Z" {
		t.Fatalf("receivable.UpdatedAt = %q, want 2026-05-06T12:30:00Z", receivable.UpdatedAt)
	}
}
