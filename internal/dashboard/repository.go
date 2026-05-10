package dashboard

import (
	"context"
	"encoding/hex"
	"fmt"

	"mlakp-backend/internal/postgres/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetSnapshot(ctx context.Context, userID string) (Snapshot, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return Snapshot{}, ErrInvalidUserID
	}

	totalsRow, err := r.queries.GetDashboardTotalsForUser(ctx, userUUID)
	if err != nil {
		return Snapshot{}, err
	}

	balanceRows, err := r.queries.ListDashboardUnsettledBalances(ctx, userUUID)
	if err != nil {
		return Snapshot{}, err
	}

	personRows, err := r.queries.ListDashboardPersonBalances(ctx, userUUID)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		Totals: Totals{
			YouOwe: DashboardAmount{
				AmountMinor: totalsRow.YouOweMinor,
				DebtCount:   totalsRow.YouOweCount,
			},
			YouGet: DashboardAmount{
				AmountMinor: totalsRow.YouGetMinor,
				DebtCount:   totalsRow.YouGetCount,
			},
		},
		UnsettledBalances: unsettledBalancesFromSQLC(balanceRows),
		PersonBalances:    personBalancesFromSQLC(personRows),
	}, nil
}

func unsettledBalancesFromSQLC(rows []sqlc.ListDashboardUnsettledBalancesRow) []UnsettledBalance {
	balances := make([]UnsettledBalance, 0, len(rows))
	for _, row := range rows {
		balances = append(balances, UnsettledBalance{
			ID:                   formatUUID(row.ID),
			ExpenseID:            formatUUID(row.ExpenseID),
			ExpenseTitle:         row.ExpenseTitle,
			DebtorID:             formatUUID(row.DebtorID),
			DebtorName:           row.DebtorName,
			DebtorUsername:       row.DebtorUsername,
			CreditorID:           formatUUID(row.CreditorID),
			CreditorName:         row.CreditorName,
			CreditorUsername:     row.CreditorUsername,
			RemainingAmountMinor: row.RemainingAmountMinor,
			Status:               row.Status,
			UpdatedAt:            row.UpdatedAt.Time,
		})
	}

	return balances
}

func personBalancesFromSQLC(rows []sqlc.ListDashboardPersonBalancesRow) []PersonBalance {
	balances := make([]PersonBalance, 0, len(rows))
	for _, row := range rows {
		balances = append(balances, PersonBalance{
			Type:                 row.BalanceType,
			OtherUserID:          formatUUID(row.OtherUserID),
			OtherUserName:        row.OtherUserName,
			OtherUserUsername:    row.OtherUserUsername,
			RemainingAmountMinor: row.RemainingAmountMinor,
			DebtCount:            row.DebtCount,
			HasPendingPayment:    row.HasPendingPayment,
		})
	}

	return balances
}

func formatUUID(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}

	encoded := hex.EncodeToString(uuid.Bytes[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}

func parseUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	if !uuid.Valid {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid")
	}

	return uuid, nil
}
