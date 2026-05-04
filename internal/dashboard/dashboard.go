package dashboard

type Totals struct {
	YouOwe DashboardAmount
	YouGet DashboardAmount
}

type DashboardAmount struct {
	AmountMinor int64
	DebtCount   int64
}
