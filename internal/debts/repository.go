package debts

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"mlakp-backend/internal/postgres/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Accept(ctx context.Context, debtID, debtorID string) (Debt, error) {
	debtUUID, debtorUUID, err := parseTransitionUUIDs(debtID, debtorID)
	if err != nil {
		return Debt{}, err
	}

	debt, err := r.queries.AcceptDebt(ctx, sqlc.AcceptDebtParams{
		ID:       debtUUID,
		DebtorID: debtorUUID,
	})
	if err != nil {
		return Debt{}, r.classifyTransitionError(ctx, err, debtUUID, debtorUUID)
	}

	return debtFromSQLC(debt), nil
}

func (r *Repository) Reject(ctx context.Context, debtID, debtorID string) (Debt, error) {
	debtUUID, debtorUUID, err := parseTransitionUUIDs(debtID, debtorID)
	if err != nil {
		return Debt{}, err
	}

	debt, err := r.queries.RejectDebt(ctx, sqlc.RejectDebtParams{
		ID:       debtUUID,
		DebtorID: debtorUUID,
	})
	if err != nil {
		return Debt{}, r.classifyTransitionError(ctx, err, debtUUID, debtorUUID)
	}

	return debtFromSQLC(debt), nil
}

func (r *Repository) ReviewRejected(ctx context.Context, params ReviewRejectedParams) (Debt, error) {
	debtUUID, reviewerUUID, err := parseTransitionUUIDs(params.DebtID, params.ReviewerID)
	if err != nil {
		return Debt{}, err
	}

	hasAdjustedAmount := params.AmountMinor != nil
	var amountMinor int64
	if hasAdjustedAmount {
		amountMinor = *params.AmountMinor
	}

	debt, err := r.queries.ReviewRejectedDebt(ctx, sqlc.ReviewRejectedDebtParams{
		ID:                debtUUID,
		UserID:            reviewerUUID,
		HasAdjustedAmount: hasAdjustedAmount,
		AmountMinor:       amountMinor,
	})
	if err != nil {
		return Debt{}, r.classifyReviewError(ctx, err, debtUUID, reviewerUUID)
	}

	return debtFromSQLC(debt), nil
}

func (r *Repository) ListForUser(ctx context.Context, filters ListFilters) ([]ListItem, error) {
	userUUID, err := parseUUID(filters.UserID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	rows, err := r.queries.ListDebtsForUser(ctx, sqlc.ListDebtsForUserParams{
		DebtorID: userUUID,
		Status: pgtype.Text{
			String: stringValue(filters.Status),
			Valid:  filters.Status != nil,
		},
		BalanceType: pgtype.Text{
			String: stringValue(filters.BalanceType),
			Valid:  filters.BalanceType != nil,
		},
	})
	if err != nil {
		return nil, err
	}

	debts := make([]ListItem, 0, len(rows))
	for _, row := range rows {
		debts = append(debts, debtListItemFromSQLC(row))
	}

	return debts, nil
}

func parseTransitionUUIDs(debtID, debtorID string) (pgtype.UUID, pgtype.UUID, error) {
	debtUUID, err := parseUUID(debtID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, ErrInvalidDebtID
	}
	debtorUUID, err := parseUUID(debtorID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, ErrInvalidUserID
	}

	return debtUUID, debtorUUID, nil
}

func (r *Repository) classifyTransitionError(ctx context.Context, err error, debtUUID, debtorUUID pgtype.UUID) error {
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	existing, lookupErr := r.queries.GetDebtByID(ctx, debtUUID)
	if lookupErr != nil {
		if errors.Is(lookupErr, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return lookupErr
	}
	if existing.DebtorID != debtorUUID {
		return ErrForbidden
	}

	return ErrInvalidState
}

func (r *Repository) classifyReviewError(ctx context.Context, err error, debtUUID, reviewerUUID pgtype.UUID) error {
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	reviewContext, lookupErr := r.queries.GetDebtReviewContext(ctx, sqlc.GetDebtReviewContextParams{
		ID:     debtUUID,
		UserID: reviewerUUID,
	})
	if lookupErr != nil {
		if errors.Is(lookupErr, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return lookupErr
	}
	if !reviewContext.IsGroupOwner {
		return ErrForbidden
	}

	return ErrInvalidState
}

func debtFromSQLC(debt sqlc.Debt) Debt {
	return Debt{
		ID:                   formatUUID(debt.ID),
		ExpenseID:            formatUUID(debt.ExpenseID),
		DebtorID:             formatUUID(debt.DebtorID),
		CreditorID:           formatUUID(debt.CreditorID),
		OriginalAmountMinor:  debt.OriginalAmountMinor,
		RemainingAmountMinor: debt.RemainingAmountMinor,
		Status:               debt.Status,
		AcceptedAt:           timestamptzPtr(debt.AcceptedAt),
		RejectedAt:           timestamptzPtr(debt.RejectedAt),
		SettledAt:            timestamptzPtr(debt.SettledAt),
		CreatedAt:            debt.CreatedAt.Time,
		UpdatedAt:            debt.UpdatedAt.Time,
	}
}

func debtListItemFromSQLC(row sqlc.ListDebtsForUserRow) ListItem {
	return ListItem{
		Debt: Debt{
			ID:                   formatUUID(row.ID),
			ExpenseID:            formatUUID(row.ExpenseID),
			DebtorID:             formatUUID(row.DebtorID),
			CreditorID:           formatUUID(row.CreditorID),
			OriginalAmountMinor:  row.OriginalAmountMinor,
			RemainingAmountMinor: row.RemainingAmountMinor,
			Status:               row.Status,
			AcceptedAt:           timestamptzPtr(row.AcceptedAt),
			RejectedAt:           timestamptzPtr(row.RejectedAt),
			SettledAt:            timestamptzPtr(row.SettledAt),
			CreatedAt:            row.CreatedAt.Time,
			UpdatedAt:            row.UpdatedAt.Time,
		},
		ExpenseTitle:     row.ExpenseTitle,
		DebtorName:       row.DebtorName,
		DebtorUsername:   row.DebtorUsername,
		CreditorName:     row.CreditorName,
		CreditorUsername: row.CreditorUsername,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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

func formatUUID(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}

	encoded := hex.EncodeToString(uuid.Bytes[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
