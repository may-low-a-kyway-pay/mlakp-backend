package handlers

import (
	"errors"
	"net/http"
	"time"

	"mlakp-backend/internal/expenses"
	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/money"
)

type ExpenseHandler struct {
	expenses *expenses.Service
}

type createExpenseRequest struct {
	GroupID      string                      `json:"group_id"`
	Title        string                      `json:"title"`
	Description  *string                     `json:"description"`
	TotalAmount  string                      `json:"total_amount"`
	Currency     string                      `json:"currency"`
	PaidBy       string                      `json:"paid_by"`
	SplitType    string                      `json:"split_type"`
	ReceiptURL   *string                     `json:"receipt_url"`
	ExpenseDate  *string                     `json:"expense_date"`
	Participants []expenseParticipantRequest `json:"participants"`
}

type expenseParticipantRequest struct {
	UserID      string  `json:"user_id"`
	ShareAmount *string `json:"share_amount"`
}

type expenseResponse struct {
	ID               string  `json:"id"`
	GroupID          string  `json:"group_id"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	TotalAmount      string  `json:"total_amount"`
	TotalAmountMinor int64   `json:"total_amount_minor"`
	Currency         string  `json:"currency"`
	PaidBy           string  `json:"paid_by"`
	SplitType        string  `json:"split_type"`
	ReceiptURL       *string `json:"receipt_url"`
	ExpenseDate      *string `json:"expense_date"`
	CreatedBy        string  `json:"created_by"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type expenseParticipantResponse struct {
	ID               string `json:"id"`
	ExpenseID        string `json:"expense_id"`
	UserID           string `json:"user_id"`
	ShareAmount      string `json:"share_amount"`
	ShareAmountMinor int64  `json:"share_amount_minor"`
	CreatedAt        string `json:"created_at"`
}

type debtResponse struct {
	ID                   string  `json:"id"`
	ExpenseID            string  `json:"expense_id"`
	ExpenseTitle         *string `json:"expense_title,omitempty"`
	DebtorID             string  `json:"debtor_id"`
	DebtorName           *string `json:"debtor_name,omitempty"`
	DebtorUsername       *string `json:"debtor_username,omitempty"`
	CreditorID           string  `json:"creditor_id"`
	CreditorName         *string `json:"creditor_name,omitempty"`
	CreditorUsername     *string `json:"creditor_username,omitempty"`
	OriginalAmount       string  `json:"original_amount"`
	OriginalAmountMinor  int64   `json:"original_amount_minor"`
	RemainingAmount      string  `json:"remaining_amount"`
	RemainingAmountMinor int64   `json:"remaining_amount_minor"`
	Status               string  `json:"status"`
	AcceptedAt           *string `json:"accepted_at"`
	RejectedAt           *string `json:"rejected_at"`
	SettledAt            *string `json:"settled_at"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

func NewExpenseHandler(expenses *expenses.Service) *ExpenseHandler {
	return &ExpenseHandler{expenses: expenses}
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	var request createExpenseRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	created, err := h.expenses.Create(r.Context(), expenses.CreateInput{
		GroupID:      request.GroupID,
		Title:        request.Title,
		Description:  request.Description,
		TotalAmount:  request.TotalAmount,
		Currency:     request.Currency,
		PaidBy:       request.PaidBy,
		SplitType:    request.SplitType,
		ReceiptURL:   request.ReceiptURL,
		ExpenseDate:  request.ExpenseDate,
		Participants: toParticipantInputs(request.Participants),
		CreatedBy:    userID,
	})
	if err != nil {
		writeExpenseError(w, err)
		return
	}

	response.Success(w, http.StatusCreated, map[string]any{
		"expense":      toExpenseResponse(created.Expense),
		"participants": toExpenseParticipantResponses(created.Participants),
		"debts":        toDebtResponses(created.Debts),
	})
}

func (h *ExpenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	details, err := h.expenses.Get(r.Context(), expenses.GetInput{
		ExpenseID: r.PathValue("expenseID"),
		UserID:    userID,
	})
	if err != nil {
		writeExpenseError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]any{
		"expense":      toExpenseResponse(details.Expense),
		"participants": toExpenseParticipantResponses(details.Participants),
		"debts":        toDebtResponses(details.Debts),
	})
}

func (h *ExpenseHandler) ListByGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	expenseList, err := h.expenses.ListByGroup(r.Context(), expenses.ListByGroupInput{
		GroupID: r.PathValue("groupID"),
		UserID:  userID,
	})
	if err != nil {
		writeExpenseError(w, err)
		return
	}

	expensesResponse := make([]expenseResponse, 0, len(expenseList))
	for _, expense := range expenseList {
		expensesResponse = append(expensesResponse, toExpenseResponse(expense))
	}

	response.Success(w, http.StatusOK, map[string][]expenseResponse{
		"expenses": expensesResponse,
	})
}

func toParticipantInputs(participants []expenseParticipantRequest) []expenses.ParticipantInput {
	inputs := make([]expenses.ParticipantInput, 0, len(participants))
	for _, participant := range participants {
		inputs = append(inputs, expenses.ParticipantInput{
			UserID:      participant.UserID,
			ShareAmount: participant.ShareAmount,
		})
	}

	return inputs
}

func writeExpenseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, expenses.ErrInvalidGroupID):
		response.Error(w, http.StatusBadRequest, "invalid_group_id", "Group ID is invalid")
	case errors.Is(err, expenses.ErrInvalidExpenseID):
		response.Error(w, http.StatusBadRequest, "invalid_expense_id", "Expense ID is invalid")
	case errors.Is(err, expenses.ErrInvalidTitle):
		response.Error(w, http.StatusBadRequest, "invalid_expense_title", "Expense title must be between 1 and 160 characters")
	case errors.Is(err, expenses.ErrInvalidAmount):
		response.Error(w, http.StatusBadRequest, "invalid_amount", "Expense amount is invalid")
	case errors.Is(err, expenses.ErrInvalidCurrency):
		response.Error(w, http.StatusBadRequest, "invalid_currency", "Currency is invalid")
	case errors.Is(err, expenses.ErrInvalidPayerID):
		response.Error(w, http.StatusBadRequest, "invalid_payer_id", "Payer ID is invalid")
	case errors.Is(err, expenses.ErrInvalidSplitType):
		response.Error(w, http.StatusBadRequest, "invalid_split_type", "Split type is invalid")
	case errors.Is(err, expenses.ErrInvalidParticipant):
		response.Error(w, http.StatusBadRequest, "invalid_participant", "Participant is invalid")
	case errors.Is(err, expenses.ErrDuplicateParticipant):
		response.Error(w, http.StatusBadRequest, "duplicate_participant", "Participant is duplicated")
	case errors.Is(err, expenses.ErrNoDebtorParticipant):
		response.Error(w, http.StatusBadRequest, "missing_debtor_participant", "At least one participant other than payer is required")
	case errors.Is(err, expenses.ErrInvalidManualShare):
		response.Error(w, http.StatusBadRequest, "invalid_manual_share", "Manual share is invalid")
	case errors.Is(err, expenses.ErrSplitMismatch):
		response.Error(w, http.StatusBadRequest, "split_mismatch", "Split total must equal amount")
	case errors.Is(err, expenses.ErrInvalidReceiptURL):
		response.Error(w, http.StatusBadRequest, "invalid_receipt_url", "Receipt URL is invalid")
	case errors.Is(err, expenses.ErrInvalidExpenseDate):
		response.Error(w, http.StatusBadRequest, "invalid_expense_date", "Expense date is invalid")
	case errors.Is(err, expenses.ErrForbidden):
		response.Error(w, http.StatusForbidden, "expense_forbidden", "You are not allowed to access this expense or group")
	case errors.Is(err, expenses.ErrNotFound):
		response.Error(w, http.StatusNotFound, "expense_not_found", "Expense was not found")
	case errors.Is(err, expenses.ErrPayerNotMember):
		response.Error(w, http.StatusBadRequest, "payer_not_group_member", "Payer must be a group member")
	case errors.Is(err, expenses.ErrParticipantNotMember):
		response.Error(w, http.StatusBadRequest, "participant_not_group_member", "Every participant must be a group member")
	default:
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func toExpenseResponse(expense expenses.Expense) expenseResponse {
	return expenseResponse{
		ID:               expense.ID,
		GroupID:          expense.GroupID,
		Title:            expense.Title,
		Description:      expense.Description,
		TotalAmount:      money.FormatMinor(expense.TotalAmountMinor),
		TotalAmountMinor: expense.TotalAmountMinor,
		Currency:         expense.Currency,
		PaidBy:           expense.PaidBy,
		SplitType:        expense.SplitType,
		ReceiptURL:       expense.ReceiptURL,
		ExpenseDate:      dateStringPtr(expense.ExpenseDate),
		CreatedBy:        expense.CreatedBy,
		CreatedAt:        expense.CreatedAt.Format(timeFormatRFC3339),
		UpdatedAt:        expense.UpdatedAt.Format(timeFormatRFC3339),
	}
}

func toExpenseParticipantResponses(participants []expenses.Participant) []expenseParticipantResponse {
	responses := make([]expenseParticipantResponse, 0, len(participants))
	for _, participant := range participants {
		responses = append(responses, expenseParticipantResponse{
			ID:               participant.ID,
			ExpenseID:        participant.ExpenseID,
			UserID:           participant.UserID,
			ShareAmount:      money.FormatMinor(participant.ShareAmountMinor),
			ShareAmountMinor: participant.ShareAmountMinor,
			CreatedAt:        participant.CreatedAt.Format(timeFormatRFC3339),
		})
	}

	return responses
}

func toDebtResponses(debts []expenses.Debt) []debtResponse {
	responses := make([]debtResponse, 0, len(debts))
	for _, debt := range debts {
		responses = append(responses, debtResponse{
			ID:                   debt.ID,
			ExpenseID:            debt.ExpenseID,
			DebtorID:             debt.DebtorID,
			CreditorID:           debt.CreditorID,
			OriginalAmount:       money.FormatMinor(debt.OriginalAmountMinor),
			OriginalAmountMinor:  debt.OriginalAmountMinor,
			RemainingAmount:      money.FormatMinor(debt.RemainingAmountMinor),
			RemainingAmountMinor: debt.RemainingAmountMinor,
			Status:               debt.Status,
			AcceptedAt:           timeStringPtr(debt.AcceptedAt),
			RejectedAt:           timeStringPtr(debt.RejectedAt),
			SettledAt:            timeStringPtr(debt.SettledAt),
			CreatedAt:            debt.CreatedAt.Format(timeFormatRFC3339),
			UpdatedAt:            debt.UpdatedAt.Format(timeFormatRFC3339),
		})
	}

	return responses
}

func dateStringPtr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}

func timeStringPtr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(timeFormatRFC3339)
	return &formatted
}
