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
	ErrInvalidAmount          = errors.New("payment amount is invalid")
	ErrNotFound               = errors.New("payment or debt not found")
	ErrForbidden              = errors.New("payment action is forbidden")
	ErrInvalidDebtState       = errors.New("debt state is invalid for payment")
	ErrInvalidPaymentState    = errors.New("payment state transition is invalid")
	ErrInvalidReviewType      = errors.New("payment review type is invalid")
	ErrAmountExceedsRemaining = errors.New("payment amount exceeds remaining debt amount")
)

type Store interface {
	Mark(ctx context.Context, params markParams) (Payment, error)
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
