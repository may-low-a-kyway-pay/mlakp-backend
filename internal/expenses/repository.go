package expenses

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"mlakp-backend/internal/postgres/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool, queries *sqlc.Queries) *Repository {
	return &Repository{
		pool:    pool,
		queries: queries,
	}
}

func (r *Repository) Create(ctx context.Context, params createParams) (CreatedExpense, error) {
	groupID, err := parseUUID(params.GroupID)
	if err != nil {
		return CreatedExpense{}, ErrInvalidGroupID
	}
	createdBy, err := parseUUID(params.CreatedBy)
	if err != nil {
		return CreatedExpense{}, ErrForbidden
	}
	paidBy, err := parseUUID(params.PaidBy)
	if err != nil {
		return CreatedExpense{}, ErrInvalidPayerID
	}

	participantIDs := make(map[string]pgtype.UUID, len(params.Participants))
	for _, participant := range params.Participants {
		userID, err := parseUUID(participant.UserID)
		if err != nil {
			return CreatedExpense{}, ErrInvalidParticipant
		}
		participantIDs[participant.UserID] = userID
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return CreatedExpense{}, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	qtx := r.queries.WithTx(tx)
	if err := requireGroupMember(ctx, qtx, groupID, createdBy, ErrForbidden); err != nil {
		return CreatedExpense{}, err
	}
	if err := requireGroupMember(ctx, qtx, groupID, paidBy, ErrPayerNotMember); err != nil {
		return CreatedExpense{}, err
	}
	for _, userID := range participantIDs {
		if err := requireGroupMember(ctx, qtx, groupID, userID, ErrParticipantNotMember); err != nil {
			return CreatedExpense{}, err
		}
	}

	expense, err := qtx.CreateExpense(ctx, sqlc.CreateExpenseParams{
		GroupID:          groupID,
		Title:            params.Title,
		Description:      nullableText(params.Description),
		TotalAmountMinor: params.TotalMinor,
		Currency:         params.Currency,
		PaidBy:           paidBy,
		SplitType:        params.SplitType,
		ReceiptUrl:       nullableText(params.ReceiptURL),
		ExpenseDate:      nullableDate(params.ExpenseDate),
		CreatedBy:        createdBy,
	})
	if err != nil {
		return CreatedExpense{}, err
	}

	participants := make([]Participant, 0, len(params.Participants))
	debts := make([]Debt, 0, len(params.Participants)-1)
	for _, participant := range params.Participants {
		userID := participantIDs[participant.UserID]
		createdParticipant, err := qtx.CreateExpenseParticipant(ctx, sqlc.CreateExpenseParticipantParams{
			ExpenseID:        expense.ID,
			UserID:           userID,
			ShareAmountMinor: participant.ShareMinor,
		})
		if err != nil {
			return CreatedExpense{}, err
		}
		participants = append(participants, participantFromSQLC(createdParticipant))

		if participant.UserID == params.PaidBy {
			continue
		}
		debt, err := qtx.CreateDebt(ctx, sqlc.CreateDebtParams{
			ExpenseID:           expense.ID,
			DebtorID:            userID,
			CreditorID:          paidBy,
			OriginalAmountMinor: participant.ShareMinor,
		})
		if err != nil {
			return CreatedExpense{}, err
		}
		debts = append(debts, debtFromSQLC(debt))
	}

	if err := tx.Commit(ctx); err != nil {
		return CreatedExpense{}, err
	}

	return CreatedExpense{
		Expense:      expenseFromSQLC(expense),
		Participants: participants,
		Debts:        debts,
	}, nil
}

func requireGroupMember(ctx context.Context, queries *sqlc.Queries, groupID, userID pgtype.UUID, errIfMissing error) error {
	isMember, err := queries.IsGroupMember(ctx, sqlc.IsGroupMemberParams{
		GroupID: groupID,
		UserID:  userID,
	})
	if err != nil {
		return err
	}
	if !isMember {
		return errIfMissing
	}

	return nil
}

func expenseFromSQLC(expense sqlc.Expense) Expense {
	return Expense{
		ID:               formatUUID(expense.ID),
		GroupID:          formatUUID(expense.GroupID),
		Title:            expense.Title,
		Description:      textPtr(expense.Description),
		TotalAmountMinor: expense.TotalAmountMinor,
		Currency:         expense.Currency,
		PaidBy:           formatUUID(expense.PaidBy),
		SplitType:        expense.SplitType,
		ReceiptURL:       textPtr(expense.ReceiptUrl),
		ExpenseDate:      datePtr(expense.ExpenseDate),
		CreatedBy:        formatUUID(expense.CreatedBy),
		CreatedAt:        expense.CreatedAt.Time,
		UpdatedAt:        expense.UpdatedAt.Time,
	}
}

func participantFromSQLC(participant sqlc.ExpenseParticipant) Participant {
	return Participant{
		ID:               formatUUID(participant.ID),
		ExpenseID:        formatUUID(participant.ExpenseID),
		UserID:           formatUUID(participant.UserID),
		ShareAmountMinor: participant.ShareAmountMinor,
		CreatedAt:        participant.CreatedAt.Time,
	}
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

func nullableText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *value, Valid: true}
}

func nullableDate(value *time.Time) pgtype.Date {
	if value == nil {
		return pgtype.Date{}
	}

	return pgtype.Date{Time: *value, Valid: true}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func datePtr(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func rollbackUnlessCommitted(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
