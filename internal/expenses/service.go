package expenses

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"mlakp-backend/internal/money"
	"mlakp-backend/internal/notifications"
)

var (
	ErrInvalidGroupID       = errors.New("group id is invalid")
	ErrInvalidExpenseID     = errors.New("expense id is invalid")
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
	ErrNotFound             = errors.New("expense not found")
	ErrPayerNotMember       = errors.New("payer is not a group member")
	ErrParticipantNotMember = errors.New("participant is not a group member")
)

type Store interface {
	Create(ctx context.Context, params createParams) (CreatedExpense, error)
	Get(ctx context.Context, expenseID, userID string) (ExpenseDetails, error)
	ListByGroup(ctx context.Context, groupID, userID string) ([]Expense, error)
}

type Notifier interface {
	Create(ctx context.Context, input notifications.CreateInput) (notifications.Notification, error)
}

type Service struct {
	store    Store
	notifier Notifier
}

func NewService(store Store, notifiers ...Notifier) *Service {
	service := &Service{store: store}
	if len(notifiers) > 0 {
		service.notifier = notifiers[0]
	}
	return service
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreatedExpense, error) {
	params, err := validateCreateInput(input)
	if err != nil {
		return CreatedExpense{}, err
	}

	created, err := s.store.Create(ctx, params)
	if err != nil {
		return CreatedExpense{}, err
	}

	for _, debt := range created.Debts {
		metadata, err := json.Marshal(map[string]string{
			"expense_id": created.Expense.ID,
		})
		if err != nil {
			metadata = []byte("{}")
		}

		s.createNotification(ctx, notifications.CreateInput{
			UserID:     debt.DebtorID,
			Type:       notifications.TypeExpenseCreated,
			Title:      "New expense to review",
			Body:       "A shared expense is waiting for your review.",
			EntityType: notifications.EntityDebt,
			EntityID:   debt.ID,
			Metadata:   metadata,
		})
	}

	return created, nil
}

func (s *Service) Get(ctx context.Context, input GetInput) (ExpenseDetails, error) {
	expenseID := strings.TrimSpace(input.ExpenseID)
	if expenseID == "" {
		return ExpenseDetails{}, ErrInvalidExpenseID
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return ExpenseDetails{}, ErrForbidden
	}

	return s.store.Get(ctx, expenseID, userID)
}

func (s *Service) ListByGroup(ctx context.Context, input ListByGroupInput) ([]Expense, error) {
	groupID := strings.TrimSpace(input.GroupID)
	if groupID == "" {
		return nil, ErrInvalidGroupID
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return nil, ErrForbidden
	}

	return s.store.ListByGroup(ctx, groupID, userID)
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

func (s *Service) createNotification(ctx context.Context, input notifications.CreateInput) {
	if s.notifier == nil {
		return
	}

	// Notification delivery must not turn an already committed expense into an API failure.
	_, _ = s.notifier.Create(ctx, input)
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
