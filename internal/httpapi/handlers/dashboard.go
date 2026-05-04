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
	YouOwe dashboardAmountResponse `json:"you_owe"`
	YouGet dashboardAmountResponse `json:"you_get"`
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

	totals, err := h.dashboard.Get(r.Context(), userID)
	if err != nil {
		writeDashboardError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]dashboardResponse{
		"dashboard": toDashboardResponse(totals),
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

func toDashboardResponse(totals dashboard.Totals) dashboardResponse {
	return dashboardResponse{
		YouOwe: toDashboardAmountResponse(totals.YouOwe),
		YouGet: toDashboardAmountResponse(totals.YouGet),
	}
}

func toDashboardAmountResponse(amount dashboard.DashboardAmount) dashboardAmountResponse {
	return dashboardAmountResponse{
		Amount:      money.FormatMinor(amount.AmountMinor),
		AmountMinor: amount.AmountMinor,
		DebtCount:   amount.DebtCount,
	}
}
