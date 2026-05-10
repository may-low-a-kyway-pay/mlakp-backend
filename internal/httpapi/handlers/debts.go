package handlers

import (
	"errors"
	"net/http"

	"mlakp-backend/internal/debts"
	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/money"
)

type DebtHandler struct {
	debts *debts.Service
}

type updateDebtRequest struct {
	Type string `json:"type"`
}

type reviewRejectedDebtRequest struct {
	Amount *string `json:"amount"`
}

func NewDebtHandler(debts *debts.Service) *DebtHandler {
	return &DebtHandler{debts: debts}
}

func (h *DebtHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	debtList, err := h.debts.List(r.Context(), debts.ListInput{
		UserID:      userID,
		Status:      r.URL.Query().Get("status"),
		BalanceType: r.URL.Query().Get("type"),
	})
	if err != nil {
		writeDebtError(w, err)
		return
	}

	debtsResponse := make([]debtResponse, 0, len(debtList))
	for _, debt := range debtList {
		debtsResponse = append(debtsResponse, toDebtListItemResponse(debt))
	}

	response.Success(w, http.StatusOK, map[string][]debtResponse{
		"debts": debtsResponse,
	})
}

func (h *DebtHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	var request updateDebtRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	debt, err := h.debts.Transition(r.Context(), r.PathValue("debtID"), userID, request.Type)
	if err != nil {
		writeDebtError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]debtResponse{
		"debt": toDebtResponse(debt),
	})
}

func (h *DebtHandler) ReviewRejected(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	var request reviewRejectedDebtRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	debt, err := h.debts.ReviewRejected(r.Context(), debts.ReviewRejectedInput{
		DebtID:     r.PathValue("debtID"),
		ReviewerID: userID,
		Amount:     request.Amount,
	})
	if err != nil {
		writeDebtError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]debtResponse{
		"debt": toDebtResponse(debt),
	})
}

func writeDebtError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, debts.ErrInvalidDebtID):
		response.Error(w, http.StatusBadRequest, "invalid_debt_id", "Debt ID is invalid")
	case errors.Is(err, debts.ErrInvalidUserID):
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
	case errors.Is(err, debts.ErrInvalidType):
		response.Error(w, http.StatusBadRequest, "invalid_debt_transition_type", "Debt transition type must be accept or reject")
	case errors.Is(err, debts.ErrInvalidStatus):
		response.Error(w, http.StatusBadRequest, "invalid_debt_status", "Debt status filter is invalid")
	case errors.Is(err, debts.ErrInvalidBalanceType):
		response.Error(w, http.StatusBadRequest, "invalid_debt_type", "Debt type filter must be owed or receivable")
	case errors.Is(err, debts.ErrInvalidAmount):
		response.Error(w, http.StatusBadRequest, "invalid_amount", "Debt amount is invalid")
	case errors.Is(err, debts.ErrNotFound):
		response.Error(w, http.StatusNotFound, "debt_not_found", "Debt was not found")
	case errors.Is(err, debts.ErrForbidden):
		response.Error(w, http.StatusForbidden, "debt_forbidden", "You are not allowed to update this debt")
	case errors.Is(err, debts.ErrInvalidState):
		response.Error(w, http.StatusConflict, "invalid_debt_state", "Debt cannot be updated from its current state")
	default:
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func toDebtListItemResponse(debt debts.ListItem) debtResponse {
	response := toDebtResponse(debt.Debt)
	response.ExpenseTitle = &debt.ExpenseTitle
	response.DebtorName = &debt.DebtorName
	response.DebtorUsername = &debt.DebtorUsername
	response.CreditorName = &debt.CreditorName
	response.CreditorUsername = &debt.CreditorUsername
	return response
}

func toDebtResponse(debt debts.Debt) debtResponse {
	return debtResponse{
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
	}
}
