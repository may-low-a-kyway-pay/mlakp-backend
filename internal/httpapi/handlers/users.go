package handlers

import (
	"errors"
	"net/http"

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

	response.JSON(w, http.StatusOK, map[string]authUserResponse{
		"user": toAuthUserResponse(user),
	})
}
