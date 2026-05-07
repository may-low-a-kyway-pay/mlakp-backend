package debts

import (
	"context"
	"errors"
	"strings"

	"mlakp-backend/internal/money"
)

var (
	ErrInvalidDebtID      = errors.New("debt id is invalid")
	ErrInvalidUserID      = errors.New("user id is invalid")
	ErrInvalidType        = errors.New("debt transition type is invalid")
	ErrInvalidStatus      = errors.New("debt status is invalid")
	ErrInvalidBalanceType = errors.New("debt balance type is invalid")
	ErrInvalidAmount      = errors.New("debt amount is invalid")
	ErrNotFound           = errors.New("debt not found")
	ErrForbidden          = errors.New("debt action is forbidden")
	ErrInvalidState       = errors.New("debt state transition is invalid")
)

type Store interface {
	Accept(ctx context.Context, debtID, debtorID string) (Debt, error)
	Reject(ctx context.Context, debtID, debtorID string) (Debt, error)
	ReviewRejected(ctx context.Context, params ReviewRejectedParams) (Debt, error)
	ListForUser(ctx context.Context, filters ListFilters) ([]ListItem, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Transition(ctx context.Context, debtID, debtorID, transitionType string) (Debt, error) {
	debtID, debtorID, err := validateTransitionInput(debtID, debtorID)
	if err != nil {
		return Debt{}, err
	}

	switch strings.TrimSpace(transitionType) {
	case TransitionAccept:
		return s.store.Accept(ctx, debtID, debtorID)
	case TransitionReject:
		return s.store.Reject(ctx, debtID, debtorID)
	default:
		return Debt{}, ErrInvalidType
	}
}

func (s *Service) ReviewRejected(ctx context.Context, input ReviewRejectedInput) (Debt, error) {
	params, err := validateReviewRejectedInput(input)
	if err != nil {
		return Debt{}, err
	}

	return s.store.ReviewRejected(ctx, params)
}

func (s *Service) List(ctx context.Context, input ListInput) ([]ListItem, error) {
	filters, err := validateListInput(input)
	if err != nil {
		return nil, err
	}

	return s.store.ListForUser(ctx, filters)
}

func validateTransitionInput(debtID, debtorID string) (string, string, error) {
	debtID = strings.TrimSpace(debtID)
	if debtID == "" {
		return "", "", ErrInvalidDebtID
	}

	debtorID = strings.TrimSpace(debtorID)
	if debtorID == "" {
		return "", "", ErrInvalidUserID
	}

	return debtID, debtorID, nil
}

func validateListInput(input ListInput) (ListFilters, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return ListFilters{}, ErrInvalidUserID
	}

	filters := ListFilters{UserID: userID}
	if status := strings.TrimSpace(input.Status); status != "" {
		switch status {
		case StatusPending, StatusAccepted, StatusRejected, StatusPartiallySettled, StatusSettled:
			filters.Status = &status
		default:
			return ListFilters{}, ErrInvalidStatus
		}
	}

	if balanceType := strings.TrimSpace(input.BalanceType); balanceType != "" {
		switch balanceType {
		case BalanceTypeOwed, BalanceTypeReceivable:
			filters.BalanceType = &balanceType
		default:
			return ListFilters{}, ErrInvalidBalanceType
		}
	}

	return filters, nil
}

func validateReviewRejectedInput(input ReviewRejectedInput) (ReviewRejectedParams, error) {
	debtID := strings.TrimSpace(input.DebtID)
	if debtID == "" {
		return ReviewRejectedParams{}, ErrInvalidDebtID
	}

	reviewerID := strings.TrimSpace(input.ReviewerID)
	if reviewerID == "" {
		return ReviewRejectedParams{}, ErrInvalidUserID
	}

	params := ReviewRejectedParams{
		DebtID:     debtID,
		ReviewerID: reviewerID,
	}
	if input.Amount != nil {
		amountMinor, err := money.ParseMinor(*input.Amount)
		if err != nil {
			return ReviewRejectedParams{}, ErrInvalidAmount
		}
		if err := money.ValidatePositive(amountMinor); err != nil {
			return ReviewRejectedParams{}, ErrInvalidAmount
		}
		params.AmountMinor = &amountMinor
	}

	return params, nil
}
