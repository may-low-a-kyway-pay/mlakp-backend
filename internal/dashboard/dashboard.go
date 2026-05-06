package dashboard

import "time"

type Totals struct {
	YouOwe DashboardAmount
	YouGet DashboardAmount
}

type DashboardAmount struct {
	AmountMinor int64
	DebtCount   int64
}

type Snapshot struct {
	Totals            Totals
	UnsettledBalances []UnsettledBalance
}

type UnsettledBalance struct {
	ID                   string
	ExpenseID            string
	ExpenseTitle         string
	DebtorID             string
	DebtorName           string
	CreditorID           string
	CreditorName         string
	RemainingAmountMinor int64
	Status               string
	UpdatedAt            time.Time
}
