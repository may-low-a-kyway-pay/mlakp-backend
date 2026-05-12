package middleware

import (
	"context"
	"net/http"
	"strings"

	"mlakp-backend/internal/auth"
	"mlakp-backend/internal/httpapi/response"
)

type userIDContextKey struct{}
type sessionIDContextKey struct{}

type SessionValidator interface {
	ValidateAccessSession(ctx context.Context, sessionID string) error
}

func Authenticate(tokenManager *auth.TokenManager, sessionValidator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if header == "" {
				response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
				return
			}

			scheme, token, ok := strings.Cut(header, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
				response.Error(w, http.StatusUnauthorized, "invalid_authorization_header", "Authorization header must use Bearer token")
				return
			}

			claims, err := tokenManager.ValidateAccessToken(strings.TrimSpace(token))
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid_access_token", "Access token is invalid or expired")
				return
			}
			if sessionValidator != nil {
				// Access tokens are valid only while their backing server session is active.
				if err := sessionValidator.ValidateAccessSession(r.Context(), claims.SessionID); err != nil {
					response.Error(w, http.StatusUnauthorized, "invalid_access_token", "Access token is invalid or expired")
					return
				}
			}

			ctx := context.WithValue(r.Context(), userIDContextKey{}, claims.Subject)
			ctx = context.WithValue(ctx, sessionIDContextKey{}, claims.SessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthenticateRealtime(tokenManager *auth.TokenManager, sessionValidator SessionValidator) func(http.Handler) http.Handler {
	return authenticateWithTokenExtractor(tokenManager, sessionValidator, func(r *http.Request) string {
		if header := strings.TrimSpace(r.Header.Get("Authorization")); header != "" {
			scheme, token, ok := strings.Cut(header, " ")
			if ok && strings.EqualFold(scheme, "Bearer") {
				return strings.TrimSpace(token)
			}
		}

		// Browser WebSocket APIs cannot set Authorization headers, so the app
		// falls back to a short-lived access token query parameter for realtime.
		return strings.TrimSpace(r.URL.Query().Get("access_token"))
	})
}

func authenticateWithTokenExtractor(tokenManager *auth.TokenManager, sessionValidator SessionValidator, extractToken func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
				return
			}

			claims, err := tokenManager.ValidateAccessToken(token)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid_access_token", "Access token is invalid or expired")
				return
			}
			if sessionValidator != nil {
				if err := sessionValidator.ValidateAccessSession(r.Context(), claims.SessionID); err != nil {
					response.Error(w, http.StatusUnauthorized, "invalid_access_token", "Access token is invalid or expired")
					return
				}
			}

			ctx := context.WithValue(r.Context(), userIDContextKey{}, claims.Subject)
			ctx = context.WithValue(ctx, sessionIDContextKey{}, claims.SessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(string)
	return userID, ok && userID != ""
}

func SessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionIDContextKey{}).(string)
	return sessionID, ok && sessionID != ""
}
