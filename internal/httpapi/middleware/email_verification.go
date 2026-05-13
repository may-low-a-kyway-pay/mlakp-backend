package middleware

import (
	"net/http"

	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/users"
)

type EmailVerificationMiddleware struct {
	usersService *users.Service
}

func NewEmailVerificationMiddleware(usersService *users.Service) *EmailVerificationMiddleware {
	return &EmailVerificationMiddleware{usersService: usersService}
}

func (m *EmailVerificationMiddleware) RequireVerifiedEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
			return
		}

		user, err := m.usersService.GetByID(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
			return
		}

		if user.EmailVerifiedAt == nil {
			response.Error(w, http.StatusForbidden, "email_verification_required", "Please verify your email to continue. Check your inbox or request a new verification code.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
