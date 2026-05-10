package payments

import (
	"context"
	"encoding/hex"
	"errors"
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

func (r *Repository) Mark(ctx context.Context, params markParams) (Payment, error) {
	debtUUID, userUUID, err := parsePairUUIDs(params.DebtID, params.UserID, ErrInvalidDebtID, ErrInvalidUserID)
	if err != nil {
		return Payment{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	qtx := r.queries.WithTx(tx)
	debt, err := qtx.GetDebtForPaymentMark(ctx, debtUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, ErrNotFound
		}
		return Payment{}, err
	}
	if debt.DebtorID != userUUID {
		return Payment{}, ErrForbidden
	}
	if !canMarkPaymentForDebt(debt.Status) {
		return Payment{}, ErrInvalidDebtState
	}
	if debt.PendingAmountMinor > 0 {
		return Payment{}, ErrPendingPaymentExists
	}
	if params.AmountMinor > debt.RemainingAmountMinor-debt.PendingAmountMinor {
		return Payment{}, ErrAmountExceedsRemaining
	}

	payment, err := qtx.CreatePayment(ctx, sqlc.CreatePaymentParams{
		DebtID:      debtUUID,
		PaidBy:      userUUID,
		ReceivedBy:  debt.CreditorID,
		AmountMinor: params.AmountMinor,
		Note:        nullableText(params.Note),
	})
	if err != nil {
		return Payment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Payment{}, err
	}

	return paymentFromSQLC(payment), nil
}

func (r *Repository) BulkMark(ctx context.Context, params bulkMarkParams) ([]Payment, error) {
	userUUID, receivedByUUID, err := parsePairUUIDs(params.UserID, params.ReceivedBy, ErrInvalidUserID, ErrInvalidReceiverID)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	qtx := r.queries.WithTx(tx)
	debts, err := qtx.ListBulkPaymentDebtsForUpdate(ctx, sqlc.ListBulkPaymentDebtsForUpdateParams{
		DebtorID:   userUUID,
		CreditorID: receivedByUUID,
	})
	if err != nil {
		return nil, err
	}
	if len(debts) == 0 {
		return nil, ErrNotFound
	}

	totalAvailable := int64(0)
	for _, debt := range debts {
		totalAvailable += debt.RemainingAmountMinor
	}
	if params.AmountMinor > totalAvailable {
		return nil, ErrAmountExceedsRemaining
	}

	remaining := params.AmountMinor
	created := make([]Payment, 0, len(debts))
	for _, debt := range debts {
		if remaining == 0 {
			break
		}

		amount := debt.RemainingAmountMinor
		if amount > remaining {
			amount = remaining
		}

		payment, err := qtx.CreatePayment(ctx, sqlc.CreatePaymentParams{
			DebtID:      debt.ID,
			PaidBy:      userUUID,
			ReceivedBy:  receivedByUUID,
			AmountMinor: amount,
			Note:        nullableText(params.Note),
		})
		if err != nil {
			return nil, err
		}

		created = append(created, paymentFromSQLC(payment))
		remaining -= amount
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) ListForUser(ctx context.Context, filters ListFilters) ([]ListItem, error) {
	userUUID, err := parseUUID(filters.UserID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	rows, err := r.queries.ListPaymentsForUser(ctx, sqlc.ListPaymentsForUserParams{
		PaidBy:      userUUID,
		Status:      nullableText(filters.Status),
		PaymentType: nullableText(filters.Type),
	})
	if err != nil {
		return nil, err
	}

	payments := make([]ListItem, 0, len(rows))
	for _, row := range rows {
		payments = append(payments, paymentListItemFromSQLC(row))
	}

	return payments, nil
}

func (r *Repository) Confirm(ctx context.Context, paymentID, userID string) (Payment, error) {
	paymentUUID, userUUID, err := parsePairUUIDs(paymentID, userID, ErrInvalidPaymentID, ErrInvalidUserID)
	if err != nil {
		return Payment{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	qtx := r.queries.WithTx(tx)
	payment, err := qtx.GetPaymentWithDebtForUpdate(ctx, paymentUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, ErrNotFound
		}
		return Payment{}, err
	}
	if payment.ReceivedBy != userUUID {
		return Payment{}, ErrForbidden
	}
	if payment.Status != StatusPendingConfirmation {
		return Payment{}, ErrInvalidPaymentState
	}
	if !canConfirmPaymentForDebt(payment.DebtStatus) {
		return Payment{}, ErrInvalidDebtState
	}
	if payment.AmountMinor > payment.DebtRemainingAmountMinor {
		return Payment{}, ErrAmountExceedsRemaining
	}

	if _, err := qtx.ApplyConfirmedPaymentToDebt(ctx, sqlc.ApplyConfirmedPaymentToDebtParams{
		ID:                   payment.DebtID,
		RemainingAmountMinor: payment.AmountMinor,
	}); err != nil {
		return Payment{}, err
	}

	confirmed, err := qtx.ConfirmPayment(ctx, paymentUUID)
	if err != nil {
		return Payment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Payment{}, err
	}

	return paymentFromSQLC(confirmed), nil
}

func (r *Repository) Reject(ctx context.Context, paymentID, userID string) (Payment, error) {
	paymentUUID, userUUID, err := parsePairUUIDs(paymentID, userID, ErrInvalidPaymentID, ErrInvalidUserID)
	if err != nil {
		return Payment{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	qtx := r.queries.WithTx(tx)
	payment, err := qtx.GetPaymentWithDebtForUpdate(ctx, paymentUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, ErrNotFound
		}
		return Payment{}, err
	}
	if payment.ReceivedBy != userUUID {
		return Payment{}, ErrForbidden
	}
	if payment.Status != StatusPendingConfirmation {
		return Payment{}, ErrInvalidPaymentState
	}

	rejected, err := qtx.RejectPayment(ctx, paymentUUID)
	if err != nil {
		return Payment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Payment{}, err
	}

	return paymentFromSQLC(rejected), nil
}

func canMarkPaymentForDebt(status string) bool {
	return status == DebtStatusAccepted || status == DebtStatusPartiallySettled
}

func canConfirmPaymentForDebt(status string) bool {
	return status == DebtStatusAccepted || status == DebtStatusPartiallySettled
}

func paymentFromSQLC(payment sqlc.Payment) Payment {
	return Payment{
		ID:          formatUUID(payment.ID),
		DebtID:      formatUUID(payment.DebtID),
		PaidBy:      formatUUID(payment.PaidBy),
		ReceivedBy:  formatUUID(payment.ReceivedBy),
		AmountMinor: payment.AmountMinor,
		Status:      payment.Status,
		Note:        textPtr(payment.Note),
		ConfirmedAt: timestamptzPtr(payment.ConfirmedAt),
		RejectedAt:  timestamptzPtr(payment.RejectedAt),
		CreatedAt:   payment.CreatedAt.Time,
		UpdatedAt:   payment.UpdatedAt.Time,
	}
}

func paymentListItemFromSQLC(payment sqlc.ListPaymentsForUserRow) ListItem {
	return ListItem{
		Payment: Payment{
			ID:          formatUUID(payment.ID),
			DebtID:      formatUUID(payment.DebtID),
			PaidBy:      formatUUID(payment.PaidBy),
			ReceivedBy:  formatUUID(payment.ReceivedBy),
			AmountMinor: payment.AmountMinor,
			Status:      payment.Status,
			Note:        textPtr(payment.Note),
			ConfirmedAt: timestamptzPtr(payment.ConfirmedAt),
			RejectedAt:  timestamptzPtr(payment.RejectedAt),
			CreatedAt:   payment.CreatedAt.Time,
			UpdatedAt:   payment.UpdatedAt.Time,
		},
		ExpenseID:                formatUUID(payment.ExpenseID),
		ExpenseTitle:             payment.ExpenseTitle,
		PaidByName:               payment.PaidByName,
		ReceivedByName:           payment.ReceivedByName,
		DebtRemainingAmountMinor: payment.DebtRemainingAmountMinor,
		DebtStatus:               payment.DebtStatus,
	}
}

func parsePairUUIDs(first, second string, firstErr, secondErr error) (pgtype.UUID, pgtype.UUID, error) {
	firstUUID, err := parseUUID(first)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, firstErr
	}
	secondUUID, err := parseUUID(second)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, secondErr
	}

	return firstUUID, secondUUID, nil
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

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
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
