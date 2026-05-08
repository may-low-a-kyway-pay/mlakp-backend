package handlers

import (
	"errors"
	"net/http"

	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/money"
	"mlakp-backend/internal/payments"
)

type PaymentHandler struct {
	payments *payments.Service
}

type markPaymentRequest struct {
	Amount string  `json:"amount"`
	Note   *string `json:"note"`
}

type reviewPaymentRequest struct {
	Type string `json:"type"`
}

type paymentResponse struct {
	ID          string  `json:"id"`
	DebtID      string  `json:"debt_id"`
	PaidBy      string  `json:"paid_by"`
	ReceivedBy  string  `json:"received_by"`
	Amount      string  `json:"amount"`
	AmountMinor int64   `json:"amount_minor"`
	Status      string  `json:"status"`
	Note        *string `json:"note"`
	ConfirmedAt *string `json:"confirmed_at"`
	RejectedAt  *string `json:"rejected_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type paymentListItemResponse struct {
	paymentResponse
	ExpenseID                string `json:"expense_id"`
	ExpenseTitle             string `json:"expense_title"`
	PaidByName               string `json:"paid_by_name"`
	ReceivedByName           string `json:"received_by_name"`
	DebtRemainingAmount      string `json:"debt_remaining_amount"`
	DebtRemainingAmountMinor int64  `json:"debt_remaining_amount_minor"`
	DebtStatus               string `json:"debt_status"`
}

func NewPaymentHandler(payments *payments.Service) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

func (h *PaymentHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	paymentList, err := h.payments.List(r.Context(), payments.ListInput{
		UserID: userID,
		Status: r.URL.Query().Get("status"),
		Type:   r.URL.Query().Get("type"),
	})
	if err != nil {
		writePaymentError(w, err)
		return
	}

	paymentsResponse := make([]paymentListItemResponse, 0, len(paymentList))
	for _, payment := range paymentList {
		paymentsResponse = append(paymentsResponse, toPaymentListItemResponse(payment))
	}

	response.Success(w, http.StatusOK, map[string][]paymentListItemResponse{
		"payments": paymentsResponse,
	})
}

func (h *PaymentHandler) Mark(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	var request markPaymentRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	payment, err := h.payments.Mark(r.Context(), payments.MarkInput{
		DebtID: r.PathValue("debtID"),
		UserID: userID,
		Amount: request.Amount,
		Note:   request.Note,
	})
	if err != nil {
		writePaymentError(w, err)
		return
	}

	response.Success(w, http.StatusCreated, map[string]paymentResponse{
		"payment": toPaymentResponse(payment),
	})
}

func (h *PaymentHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	var request reviewPaymentRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	payment, err := h.payments.Review(r.Context(), payments.ReviewInput{
		PaymentID: r.PathValue("paymentID"),
		UserID:    userID,
		Type:      request.Type,
	})
	if err != nil {
		writePaymentError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]paymentResponse{
		"payment": toPaymentResponse(payment),
	})
}

func writePaymentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payments.ErrInvalidDebtID):
		response.Error(w, http.StatusBadRequest, "invalid_debt_id", "Debt ID is invalid")
	case errors.Is(err, payments.ErrInvalidPaymentID):
		response.Error(w, http.StatusBadRequest, "invalid_payment_id", "Payment ID is invalid")
	case errors.Is(err, payments.ErrInvalidUserID):
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
	case errors.Is(err, payments.ErrInvalidAmount):
		response.Error(w, http.StatusBadRequest, "invalid_amount", "Payment amount is invalid")
	case errors.Is(err, payments.ErrNotFound):
		response.Error(w, http.StatusNotFound, "payment_not_found", "Payment or debt was not found")
	case errors.Is(err, payments.ErrForbidden):
		response.Error(w, http.StatusForbidden, "payment_forbidden", "You are not allowed to update this payment")
	case errors.Is(err, payments.ErrInvalidDebtState):
		response.Error(w, http.StatusConflict, "invalid_debt_state", "Debt cannot accept payments in its current state")
	case errors.Is(err, payments.ErrInvalidPaymentState):
		response.Error(w, http.StatusConflict, "invalid_payment_state", "Payment cannot be updated from its current state")
	case errors.Is(err, payments.ErrInvalidReviewType):
		response.Error(w, http.StatusBadRequest, "invalid_payment_review_type", "Payment review type must be confirm or reject")
	case errors.Is(err, payments.ErrInvalidStatus):
		response.Error(w, http.StatusBadRequest, "invalid_payment_status", "Payment status filter is invalid")
	case errors.Is(err, payments.ErrInvalidType):
		response.Error(w, http.StatusBadRequest, "invalid_payment_type", "Payment type filter must be received, sent, or all")
	case errors.Is(err, payments.ErrPendingPaymentExists):
		response.Error(w, http.StatusConflict, "payment_pending_confirmation_exists", "A payment for this debt is already waiting for creditor review")
	case errors.Is(err, payments.ErrAmountExceedsRemaining):
		response.Error(w, http.StatusConflict, "payment_amount_exceeds_remaining", "Payment amount exceeds remaining debt amount")
	default:
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func toPaymentListItemResponse(payment payments.ListItem) paymentListItemResponse {
	return paymentListItemResponse{
		paymentResponse:          toPaymentResponse(payment.Payment),
		ExpenseID:                payment.ExpenseID,
		ExpenseTitle:             payment.ExpenseTitle,
		PaidByName:               payment.PaidByName,
		ReceivedByName:           payment.ReceivedByName,
		DebtRemainingAmount:      money.FormatMinor(payment.DebtRemainingAmountMinor),
		DebtRemainingAmountMinor: payment.DebtRemainingAmountMinor,
		DebtStatus:               payment.DebtStatus,
	}
}

func toPaymentResponse(payment payments.Payment) paymentResponse {
	return paymentResponse{
		ID:          payment.ID,
		DebtID:      payment.DebtID,
		PaidBy:      payment.PaidBy,
		ReceivedBy:  payment.ReceivedBy,
		Amount:      money.FormatMinor(payment.AmountMinor),
		AmountMinor: payment.AmountMinor,
		Status:      payment.Status,
		Note:        payment.Note,
		ConfirmedAt: timeStringPtr(payment.ConfirmedAt),
		RejectedAt:  timeStringPtr(payment.RejectedAt),
		CreatedAt:   payment.CreatedAt.Format(timeFormatRFC3339),
		UpdatedAt:   payment.UpdatedAt.Format(timeFormatRFC3339),
	}
}
