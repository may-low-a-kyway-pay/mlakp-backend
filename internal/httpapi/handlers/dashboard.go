package handlers

import (
	"errors"
	"net/http"

	"mlakp-backend/internal/dashboard"
	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/money"
)

type DashboardHandler struct {
	dashboard *dashboard.Service
}

type dashboardAmountResponse struct {
	Amount      string `json:"amount"`
	AmountMinor int64  `json:"amount_minor"`
	DebtCount   int64  `json:"debt_count"`
}

type dashboardResponse struct {
	YouOwe            dashboardAmountResponse    `json:"you_owe"`
	YouGet            dashboardAmountResponse    `json:"you_get"`
	UnsettledBalances []unsettledBalanceResponse `json:"unsettled_balances"`
	PersonBalances    []personBalanceResponse    `json:"person_balances"`
}

type dashboardUserResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type unsettledBalanceResponse struct {
	ID              string                `json:"id"`
	ExpenseID       string                `json:"expense_id"`
	ExpenseTitle    string                `json:"expense_title"`
	Type            string                `json:"type"`
	OtherUser       dashboardUserResponse `json:"other_user"`
	RemainingAmount string                `json:"remaining_amount"`
	RemainingMinor  int64                 `json:"remaining_amount_minor"`
	Status          string                `json:"status"`
	UpdatedAt       string                `json:"updated_at"`
}

type personBalanceResponse struct {
	Type            string                `json:"type"`
	OtherUser       dashboardUserResponse `json:"other_user"`
	RemainingAmount string                `json:"remaining_amount"`
	RemainingMinor  int64                 `json:"remaining_amount_minor"`
	DebtCount       int64                 `json:"debt_count"`
}

func NewDashboardHandler(dashboard *dashboard.Service) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	snapshot, err := h.dashboard.Get(r.Context(), userID)
	if err != nil {
		writeDashboardError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]dashboardResponse{
		"dashboard": toDashboardResponse(snapshot, userID),
	})
}

func writeDashboardError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dashboard.ErrInvalidUserID):
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
	default:
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func toDashboardResponse(snapshot dashboard.Snapshot, userID string) dashboardResponse {
	return dashboardResponse{
		YouOwe:            toDashboardAmountResponse(snapshot.Totals.YouOwe),
		YouGet:            toDashboardAmountResponse(snapshot.Totals.YouGet),
		UnsettledBalances: toUnsettledBalanceResponses(snapshot.UnsettledBalances, userID),
		PersonBalances:    toPersonBalanceResponses(snapshot.PersonBalances),
	}
}

func toDashboardAmountResponse(amount dashboard.DashboardAmount) dashboardAmountResponse {
	return dashboardAmountResponse{
		Amount:      money.FormatMinor(amount.AmountMinor),
		AmountMinor: amount.AmountMinor,
		DebtCount:   amount.DebtCount,
	}
}

func toUnsettledBalanceResponses(balances []dashboard.UnsettledBalance, userID string) []unsettledBalanceResponse {
	responses := make([]unsettledBalanceResponse, 0, len(balances))
	for _, balance := range balances {
		balanceType := "receivable"
		otherUser := dashboardUserResponse{
			ID:   balance.DebtorID,
			Name: balance.DebtorName,
		}

		if balance.DebtorID == userID {
			balanceType = "owed"
			otherUser = dashboardUserResponse{
				ID:   balance.CreditorID,
				Name: balance.CreditorName,
			}
		}

		responses = append(responses, unsettledBalanceResponse{
			ID:              balance.ID,
			ExpenseID:       balance.ExpenseID,
			ExpenseTitle:    balance.ExpenseTitle,
			Type:            balanceType,
			OtherUser:       otherUser,
			RemainingAmount: money.FormatMinor(balance.RemainingAmountMinor),
			RemainingMinor:  balance.RemainingAmountMinor,
			Status:          balance.Status,
			UpdatedAt:       balance.UpdatedAt.Format(timeFormatRFC3339),
		})
	}

	return responses
}

func toPersonBalanceResponses(balances []dashboard.PersonBalance) []personBalanceResponse {
	responses := make([]personBalanceResponse, 0, len(balances))
	for _, balance := range balances {
		responses = append(responses, personBalanceResponse{
			Type: balance.Type,
			OtherUser: dashboardUserResponse{
				ID:   balance.OtherUserID,
				Name: balance.OtherUserName,
			},
			RemainingAmount: money.FormatMinor(balance.RemainingAmountMinor),
			RemainingMinor:  balance.RemainingAmountMinor,
			DebtCount:       balance.DebtCount,
		})
	}

	return responses
}
