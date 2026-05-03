package expenses

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"mlakp-backend/internal/money"
)

var (
	ErrInvalidGroupID       = errors.New("group id is invalid")
	ErrInvalidTitle         = errors.New("expense title must be between 1 and 160 characters")
	ErrInvalidAmount        = errors.New("expense amount is invalid")
	ErrInvalidCurrency      = errors.New("currency is invalid")
	ErrInvalidPayerID       = errors.New("payer id is invalid")
	ErrInvalidSplitType     = errors.New("split type is invalid")
	ErrInvalidParticipant   = errors.New("participant is invalid")
	ErrDuplicateParticipant = errors.New("participant is duplicated")
	ErrNoDebtorParticipant  = errors.New("at least one participant other than payer is required")
	ErrInvalidManualShare   = errors.New("manual share is invalid")
	ErrSplitMismatch        = errors.New("split total must equal amount")
	ErrInvalidReceiptURL    = errors.New("receipt url is invalid")
	ErrInvalidExpenseDate   = errors.New("expense date is invalid")
	ErrForbidden            = errors.New("expense action is forbidden")
	ErrPayerNotMember       = errors.New("payer is not a group member")
	ErrParticipantNotMember = errors.New("participant is not a group member")
)

type Store interface {
	Create(ctx context.Context, params createParams) (CreatedExpense, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreatedExpense, error) {
	params, err := validateCreateInput(input)
	if err != nil {
		return CreatedExpense{}, err
	}

	return s.store.Create(ctx, params)
}

func validateCreateInput(input CreateInput) (createParams, error) {
	groupID := strings.TrimSpace(input.GroupID)
	if groupID == "" {
		return createParams{}, ErrInvalidGroupID
	}

	title := strings.TrimSpace(input.Title)
	if len(title) < 1 || len(title) > 160 {
		return createParams{}, ErrInvalidTitle
	}

	totalMinor, err := money.ParseMinor(input.TotalAmount)
	if err != nil {
		return createParams{}, ErrInvalidAmount
	}
	if err := money.ValidatePositive(totalMinor); err != nil {
		return createParams{}, ErrInvalidAmount
	}

	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = CurrencyTHB
	}
	if currency != CurrencyTHB {
		return createParams{}, ErrInvalidCurrency
	}

	paidBy := strings.TrimSpace(input.PaidBy)
	if paidBy == "" {
		return createParams{}, ErrInvalidPayerID
	}

	splitType := strings.TrimSpace(input.SplitType)
	if splitType != SplitTypeEqual && splitType != SplitTypeManual {
		return createParams{}, ErrInvalidSplitType
	}

	createdBy := strings.TrimSpace(input.CreatedBy)
	if createdBy == "" {
		return createParams{}, ErrForbidden
	}

	description := normalizeOptionalString(input.Description)
	receiptURL, err := normalizeReceiptURL(input.ReceiptURL)
	if err != nil {
		return createParams{}, err
	}
	expenseDate, err := normalizeExpenseDate(input.ExpenseDate)
	if err != nil {
		return createParams{}, err
	}

	participants, err := buildParticipantShares(totalMinor, paidBy, splitType, input.Participants)
	if err != nil {
		return createParams{}, err
	}

	return createParams{
		GroupID:      groupID,
		Title:        title,
		Description:  description,
		TotalMinor:   totalMinor,
		Currency:     currency,
		PaidBy:       paidBy,
		SplitType:    splitType,
		ReceiptURL:   receiptURL,
		ExpenseDate:  expenseDate,
		Participants: participants,
		CreatedBy:    createdBy,
	}, nil
}

func buildParticipantShares(totalMinor int64, paidBy, splitType string, inputs []ParticipantInput) ([]participantShare, error) {
	if len(inputs) == 0 {
		return nil, ErrInvalidParticipant
	}

	userIDs := make([]string, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	hasDebtor := false
	for _, input := range inputs {
		userID := strings.TrimSpace(input.UserID)
		if userID == "" {
			return nil, ErrInvalidParticipant
		}
		if _, ok := seen[userID]; ok {
			return nil, ErrDuplicateParticipant
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
		if userID != paidBy {
			hasDebtor = true
		}
	}
	if !hasDebtor {
		return nil, ErrNoDebtorParticipant
	}

	switch splitType {
	case SplitTypeEqual:
		shares, err := money.SplitEqual(totalMinor, len(inputs))
		if err != nil {
			return nil, ErrInvalidParticipant
		}

		participants := make([]participantShare, 0, len(inputs))
		for i, userID := range userIDs {
			participants = append(participants, participantShare{
				UserID:     userID,
				ShareMinor: shares[i],
			})
		}
		return participants, nil
	case SplitTypeManual:
		participants := make([]participantShare, 0, len(inputs))
		shares := make([]int64, 0, len(inputs))
		for i, input := range inputs {
			if input.ShareAmount == nil {
				return nil, ErrInvalidManualShare
			}
			shareMinor, err := money.ParseMinor(*input.ShareAmount)
			if err != nil {
				return nil, ErrInvalidManualShare
			}
			if err := money.ValidatePositive(shareMinor); err != nil {
				return nil, ErrInvalidManualShare
			}
			shares = append(shares, shareMinor)
			participants = append(participants, participantShare{
				UserID:     userIDs[i],
				ShareMinor: shareMinor,
			})
		}
		if err := money.ValidateManualSplit(totalMinor, shares); err != nil {
			if errors.Is(err, money.ErrSplitMismatch) {
				return nil, ErrSplitMismatch
			}
			return nil, ErrInvalidManualShare
		}
		return participants, nil
	default:
		return nil, ErrInvalidSplitType
	}
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

func normalizeReceiptURL(value *string) (*string, error) {
	trimmed := normalizeOptionalString(value)
	if trimmed == nil {
		return nil, nil
	}

	parsed, err := url.ParseRequestURI(*trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidReceiptURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrInvalidReceiptURL
	}

	return trimmed, nil
}

func normalizeExpenseDate(value *string) (*time.Time, error) {
	trimmed := normalizeOptionalString(value)
	if trimmed == nil {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", *trimmed)
	if err != nil {
		return nil, ErrInvalidExpenseDate
	}

	return &parsed, nil
}
