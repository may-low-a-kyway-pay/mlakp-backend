package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"mlakp-backend/internal/auth"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/users"
)

type AuthHandler struct {
	users        *users.Service
	tokenManager *auth.TokenManager
}

type authUserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type tokenResponse struct {
	AccessToken string           `json:"access_token"`
	TokenType   string           `json:"token_type"`
	ExpiresAt   string           `json:"expires_at"`
	User        authUserResponse `json:"user"`
}

func NewAuthHandler(users *users.Service, tokenManager *auth.TokenManager) *AuthHandler {
	return &AuthHandler{
		users:        users,
		tokenManager: tokenManager,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	user, err := h.users.Register(r.Context(), request.Name, request.Email, request.Password)
	if err != nil {
		writeUserError(w, err)
		return
	}

	accessToken, expiresAt, err := h.tokenManager.IssueAccessToken(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	response.JSON(w, http.StatusCreated, tokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt.Format(timeFormatRFC3339),
		User:        toAuthUserResponse(user),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	user, err := h.users.Authenticate(r.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			response.Error(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect")
			return
		}
		writeUserError(w, err)
		return
	}

	accessToken, expiresAt, err := h.tokenManager.IssueAccessToken(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	response.JSON(w, http.StatusOK, tokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt.Format(timeFormatRFC3339),
		User:        toAuthUserResponse(user),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, users.ErrInvalidName):
		response.Error(w, http.StatusBadRequest, "invalid_name", "Name must be between 1 and 120 characters")
	case errors.Is(err, users.ErrInvalidEmail):
		response.Error(w, http.StatusBadRequest, "invalid_email", "Email is invalid")
	case errors.Is(err, users.ErrInvalidPassword):
		response.Error(w, http.StatusBadRequest, "invalid_password", "Password must be at least 8 characters")
	case errors.Is(err, users.ErrEmailConflict):
		response.Error(w, http.StatusConflict, "email_already_registered", "Email is already registered")
	default:
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func toAuthUserResponse(user users.User) authUserResponse {
	return authUserResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"
