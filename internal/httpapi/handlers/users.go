package handlers

import (
	"errors"
	"net/http"
	"time"

	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/users"
)

type UserHandler struct {
	users *users.Service
}

func NewUserHandler(users *users.Service) *UserHandler {
	return &UserHandler{users: users}
}

type userMeResponse struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Username             string     `json:"username"`
	Email                string     `json:"email"`
	EmailVerifiedAt      *time.Time `json:"email_verified_at"`
	VerificationDeadline *time.Time `json:"verification_deadline"`
	VerificationStatus   string     `json:"verification_status"`
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			response.Error(w, http.StatusUnauthorized, "invalid_access_token", "Access token user no longer exists")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	verificationStatus := h.users.GetVerificationStatus(&user)

	resp := userMeResponse{
		ID:                   user.ID,
		Name:                 user.Name,
		Username:             user.Username,
		Email:                user.Email,
		EmailVerifiedAt:      user.EmailVerifiedAt,
		VerificationDeadline: user.VerificationDeadline,
		VerificationStatus:   verificationStatus.Status,
	}

	response.Success(w, http.StatusOK, map[string]any{
		"user":           resp,
		"days_remaining": verificationStatus.DaysRemaining,
	})
}

func (h *UserHandler) Search(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	results, err := h.users.SearchByUsername(r.Context(), r.URL.Query().Get("username"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	usersResponse := make([]authUserResponse, 0, len(results))
	for _, user := range results {
		usersResponse = append(usersResponse, toAuthUserResponse(user))
	}

	response.Success(w, http.StatusOK, map[string][]authUserResponse{
		"users": usersResponse,
	})
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	var request struct {
		Username string `json:"username"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	user, err := h.users.UpdateUsername(r.Context(), userID, request.Username)
	if err != nil {
		writeUserError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]authUserResponse{
		"user": toAuthUserResponse(user),
	})
}
