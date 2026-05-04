package dashboard

import (
	"context"
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

func (r *Repository) GetTotals(ctx context.Context, userID string) (Totals, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return Totals{}, ErrInvalidUserID
	}

	row, err := r.queries.GetDashboardTotalsForUser(ctx, userUUID)
	if err != nil {
		return Totals{}, err
	}

	return Totals{
		YouOwe: DashboardAmount{
			AmountMinor: row.YouOweMinor,
			DebtCount:   row.YouOweCount,
		},
		YouGet: DashboardAmount{
			AmountMinor: row.YouGetMinor,
			DebtCount:   row.YouGetCount,
		},
	}, nil
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
