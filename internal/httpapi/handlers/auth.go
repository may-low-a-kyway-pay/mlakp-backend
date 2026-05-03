package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"mlakp-backend/internal/auth"
	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/sessions"
	"mlakp-backend/internal/users"
)

type AuthHandler struct {
	users        *users.Service
	tokenManager *auth.TokenManager
	sessions     *sessions.Service
}

type authUserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type tokenResponse struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	TokenType    string           `json:"token_type"`
	ExpiresAt    string           `json:"expires_at"`
	User         authUserResponse `json:"user"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresAt    string `json:"expires_at"`
}

func NewAuthHandler(users *users.Service, tokenManager *auth.TokenManager, sessions *sessions.Service) *AuthHandler {
	return &AuthHandler{
		users:        users,
		tokenManager: tokenManager,
		sessions:     sessions,
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

	session, refreshToken, err := h.sessions.Create(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	accessToken, expiresAt, err := h.tokenManager.IssueAccessToken(r.Context(), user.ID, session.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	response.Success(w, http.StatusCreated, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt.Format(timeFormatRFC3339),
		User:         toAuthUserResponse(user),
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

	session, refreshToken, err := h.sessions.Create(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	accessToken, expiresAt, err := h.tokenManager.IssueAccessToken(r.Context(), user.ID, session.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	response.Success(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt.Format(timeFormatRFC3339),
		User:         toAuthUserResponse(user),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}

	session, refreshToken, err := h.sessions.Refresh(r.Context(), request.RefreshToken)
	if err != nil {
		if errors.Is(err, sessions.ErrInvalidRefreshToken) {
			response.Error(w, http.StatusUnauthorized, "invalid_refresh_token", "Refresh token is invalid or expired")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	accessToken, expiresAt, err := h.tokenManager.IssueAccessToken(r.Context(), session.UserID, session.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	response.Success(w, http.StatusOK, refreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt.Format(timeFormatRFC3339),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := middleware.SessionIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}
	if err := h.sessions.Revoke(r.Context(), sessionID); err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	response.Success(w, http.StatusOK, nil)
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
