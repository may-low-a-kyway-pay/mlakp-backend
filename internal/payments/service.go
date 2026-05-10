package payments

import (
	"context"
	"errors"
	"strings"

	"mlakp-backend/internal/money"
)

var (
	ErrInvalidDebtID          = errors.New("debt id is invalid")
	ErrInvalidPaymentID       = errors.New("payment id is invalid")
	ErrInvalidUserID          = errors.New("user id is invalid")
	ErrInvalidReceiverID      = errors.New("receiver id is invalid")
	ErrInvalidAmount          = errors.New("payment amount is invalid")
	ErrNotFound               = errors.New("payment or debt not found")
	ErrForbidden              = errors.New("payment action is forbidden")
	ErrInvalidDebtState       = errors.New("debt state is invalid for payment")
	ErrInvalidPaymentState    = errors.New("payment state transition is invalid")
	ErrInvalidReviewType      = errors.New("payment review type is invalid")
	ErrInvalidStatus          = errors.New("payment status is invalid")
	ErrInvalidType            = errors.New("payment type is invalid")
	ErrPendingPaymentExists   = errors.New("pending payment already exists for debt")
	ErrAmountExceedsRemaining = errors.New("payment amount exceeds remaining debt amount")
)

type Store interface {
	Mark(ctx context.Context, params markParams) (Payment, error)
	BulkMark(ctx context.Context, params bulkMarkParams) ([]Payment, error)
	ListForUser(ctx context.Context, filters ListFilters) ([]ListItem, error)
	Confirm(ctx context.Context, paymentID, userID string) (Payment, error)
	Reject(ctx context.Context, paymentID, userID string) (Payment, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Mark(ctx context.Context, input MarkInput) (Payment, error) {
	params, err := validateMarkInput(input)
	if err != nil {
		return Payment{}, err
	}

	return s.store.Mark(ctx, params)
}

func (s *Service) BulkMark(ctx context.Context, input BulkMarkInput) ([]Payment, error) {
	params, err := validateBulkMarkInput(input)
	if err != nil {
		return nil, err
	}

	return s.store.BulkMark(ctx, params)
}

func (s *Service) List(ctx context.Context, input ListInput) ([]ListItem, error) {
	filters, err := validateListInput(input)
	if err != nil {
		return nil, err
	}

	return s.store.ListForUser(ctx, filters)
}

func (s *Service) Confirm(ctx context.Context, input ReviewInput) (Payment, error) {
	paymentID, userID, err := validateReviewIdentity(input)
	if err != nil {
		return Payment{}, err
	}

	return s.store.Confirm(ctx, paymentID, userID)
}

func (s *Service) Reject(ctx context.Context, input ReviewInput) (Payment, error) {
	paymentID, userID, err := validateReviewIdentity(input)
	if err != nil {
		return Payment{}, err
	}

	return s.store.Reject(ctx, paymentID, userID)
}

func (s *Service) Review(ctx context.Context, input ReviewInput) (Payment, error) {
	paymentID, userID, reviewType, err := validateReviewInput(input)
	if err != nil {
		return Payment{}, err
	}

	switch reviewType {
	case ReviewTypeConfirm:
		return s.store.Confirm(ctx, paymentID, userID)
	case ReviewTypeReject:
		return s.store.Reject(ctx, paymentID, userID)
	default:
		return Payment{}, ErrInvalidReviewType
	}
}

func validateListInput(input ListInput) (ListFilters, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return ListFilters{}, ErrInvalidUserID
	}

	filters := ListFilters{UserID: userID}
	if status := strings.TrimSpace(input.Status); status != "" {
		switch status {
		case StatusPendingConfirmation, StatusConfirmed, StatusRejected:
			filters.Status = &status
		default:
			return ListFilters{}, ErrInvalidStatus
		}
	}

	if paymentType := strings.TrimSpace(input.Type); paymentType != "" && paymentType != TypeAll {
		switch paymentType {
		case TypeReceived, TypeSent:
			filters.Type = &paymentType
		default:
			return ListFilters{}, ErrInvalidType
		}
	}

	return filters, nil
}

func validateMarkInput(input MarkInput) (markParams, error) {
	debtID := strings.TrimSpace(input.DebtID)
	if debtID == "" {
		return markParams{}, ErrInvalidDebtID
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return markParams{}, ErrInvalidUserID
	}

	amountMinor, err := money.ParseMinor(input.Amount)
	if err != nil {
		return markParams{}, ErrInvalidAmount
	}
	if err := money.ValidatePositive(amountMinor); err != nil {
		return markParams{}, ErrInvalidAmount
	}

	return markParams{
		DebtID:      debtID,
		UserID:      userID,
		AmountMinor: amountMinor,
		Note:        normalizeOptionalString(input.Note),
	}, nil
}

func validateBulkMarkInput(input BulkMarkInput) (bulkMarkParams, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return bulkMarkParams{}, ErrInvalidUserID
	}

	receivedBy := strings.TrimSpace(input.ReceivedBy)
	if receivedBy == "" {
		return bulkMarkParams{}, ErrInvalidReceiverID
	}

	if receivedBy == userID {
		return bulkMarkParams{}, ErrForbidden
	}

	amountMinor, err := money.ParseMinor(input.Amount)
	if err != nil {
		return bulkMarkParams{}, ErrInvalidAmount
	}
	if err := money.ValidatePositive(amountMinor); err != nil {
		return bulkMarkParams{}, ErrInvalidAmount
	}

	return bulkMarkParams{
		UserID:      userID,
		ReceivedBy:  receivedBy,
		AmountMinor: amountMinor,
		Note:        normalizeOptionalString(input.Note),
	}, nil
}

func validateReviewInput(input ReviewInput) (string, string, string, error) {
	paymentID, userID, err := validateReviewIdentity(input)
	if err != nil {
		return "", "", "", err
	}

	reviewType := strings.TrimSpace(input.Type)
	if reviewType != ReviewTypeConfirm && reviewType != ReviewTypeReject {
		return "", "", "", ErrInvalidReviewType
	}

	return paymentID, userID, reviewType, nil
}

func validateReviewIdentity(input ReviewInput) (string, string, error) {
	paymentID := strings.TrimSpace(input.PaymentID)
	if paymentID == "" {
		return "", "", ErrInvalidPaymentID
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return "", "", ErrInvalidUserID
	}

	return paymentID, userID, nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
