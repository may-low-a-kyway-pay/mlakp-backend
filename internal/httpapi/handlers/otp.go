package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"mlakp-backend/internal/auth"
	"mlakp-backend/internal/email"
	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/otp"
	"mlakp-backend/internal/sessions"
	"mlakp-backend/internal/users"
)

const otpTimeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

type OTPHandler struct {
	usersService    *users.Service
	otpService      *otp.Service
	emailProvider   *email.Provider
	sessionsService *sessions.Service
	tokenManager    *auth.TokenManager
}

type sendOTPRequest struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
}

type verifyOTPRequest struct {
	Email   string `json:"email"`
	OTP     string `json:"otp"`
	Purpose string `json:"purpose"`
}

type resetPasswordRequest struct {
	Email       string `json:"email"`
	OTP         string `json:"otp"`
	NewPassword string `json:"new_password"`
}

func NewOTPHandler(usersService *users.Service, otpService *otp.Service, emailProvider *email.Provider, sessionsService *sessions.Service, tokenManager *auth.TokenManager) *OTPHandler {
	return &OTPHandler{
		usersService:    usersService,
		otpService:      otpService,
		emailProvider:   emailProvider,
		sessionsService: sessionsService,
		tokenManager:    tokenManager,
	}
}

func (h *OTPHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var request sendOTPRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.Purpose = strings.TrimSpace(request.Purpose)

	if request.Email == "" {
		response.Error(w, http.StatusBadRequest, "invalid_email", "Email is required")
		return
	}
	if request.Purpose != "signup" && request.Purpose != "password_reset" {
		response.Error(w, http.StatusBadRequest, "invalid_purpose", "Purpose must be 'signup' or 'password_reset'")
		return
	}

	if err := h.otpService.CheckCooldown(r.Context(), request.Email, request.Purpose); err != nil {
		if err == otp.ErrOTPCooldown {
			response.Error(w, http.StatusTooManyRequests, "otp_cooldown", "Please wait before requesting another OTP")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	if request.Purpose == "password_reset" {
		_, err := h.usersService.GetByEmail(r.Context(), request.Email)
		if err != nil {
			if err == users.ErrNotFound || err == users.ErrEmailConflict {
				response.Error(w, http.StatusBadRequest, "email_not_found", "No account found with this email")
				return
			}
			response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
			return
		}
	}

	var userID *string
	userName := "User"
	if request.Purpose == "signup" {
		user, err := h.usersService.GetByEmail(r.Context(), request.Email)
		if err == nil {
			if user.EmailVerifiedAt != nil {
				response.Error(w, http.StatusConflict, "email_already_verified", "Email is already verified")
				return
			}
			userID = &user.ID
			userName = user.Name
		} else if err == users.ErrNotFound {
			response.Error(w, http.StatusBadRequest, "email_not_registered", "Please register first before verifying email")
			return
		} else {
			response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
			return
		}
	}

	otpCode, verification, err := h.otpService.CreateVerification(r.Context(), request.Email, request.Purpose, userID)
	if err != nil {
		if err == otp.ErrOTPRateLimited {
			response.Error(w, http.StatusTooManyRequests, "otp_rate_limited", "Too many OTP requests. Please try again later.")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	if err := h.emailProvider.SendOTPEmail(r.Context(), request.Email, userName, otpCode, request.Purpose); err != nil {
		slog.Default().Error("failed to send otp email", "error", err, "purpose", request.Purpose)
		response.Error(w, http.StatusInternalServerError, "email_send_failed", "Failed to send email. Please try again.")
		return
	}

	response.Success(w, http.StatusOK, map[string]interface{}{
		"message":    "OTP sent successfully",
		"expires_at": verification.ExpiresAt.Format(otpTimeFormatRFC3339),
		"purpose":    request.Purpose,
	})
}

func (h *OTPHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var request verifyOTPRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.OTP = strings.TrimSpace(request.OTP)
	request.Purpose = strings.TrimSpace(request.Purpose)

	if request.Email == "" || request.OTP == "" || request.Purpose == "" {
		response.Error(w, http.StatusBadRequest, "missing_fields", "Email, OTP, and purpose are required")
		return
	}
	if request.Purpose != "signup" {
		response.Error(w, http.StatusBadRequest, "invalid_purpose", "Use the password reset endpoint for password reset OTPs")
		return
	}

	privateUser, err := h.usersService.GetByEmail(r.Context(), request.Email)
	if err != nil {
		if err == users.ErrNotFound {
			response.Error(w, http.StatusBadRequest, "email_not_registered", "Please register first before verifying email")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	if privateUser.EmailVerifiedAt != nil {
		response.Error(w, http.StatusConflict, "email_already_verified", "Email is already verified")
		return
	}

	_, err = h.otpService.VerifyOTPForUser(r.Context(), request.Email, "signup", request.OTP, privateUser.ID)
	if err != nil {
		switch err {
		case otp.ErrOTPNotFound:
			response.Error(w, http.StatusBadRequest, "otp_not_found", "OTP not found or expired. Please request a new one.")
		case otp.ErrOTPExpired:
			response.Error(w, http.StatusBadRequest, "otp_expired", "OTP has expired. Please request a new one.")
		case otp.ErrOTPMaxAttempts:
			response.Error(w, http.StatusBadRequest, "otp_max_attempts", "Maximum verification attempts exceeded. Please request a new OTP.")
		case otp.ErrOTPInvalid:
			response.Error(w, http.StatusBadRequest, "otp_invalid", "Invalid OTP code")
		default:
			response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	user, err := h.usersService.MarkEmailVerified(r.Context(), privateUser.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	session, refreshToken, err := h.sessionsService.Create(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	accessToken, expiresAt, err := h.tokenManager.IssueAccessToken(r.Context(), user.ID, session.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	response.Success(w, http.StatusOK, map[string]interface{}{
		"verified":      true,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_at":    expiresAt.Format(otpTimeFormatRFC3339),
	})
}

func (h *OTPHandler) SendOTPForAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
		return
	}

	user, err := h.usersService.GetByID(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	if err := h.otpService.CheckCooldown(r.Context(), user.Email, "password_reset"); err != nil {
		if err == otp.ErrOTPCooldown {
			response.Error(w, http.StatusTooManyRequests, "otp_cooldown", "Please wait before requesting another OTP")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	otpCode, verification, err := h.otpService.CreateVerification(r.Context(), user.Email, "password_reset", &userID)
	if err != nil {
		if err == otp.ErrOTPRateLimited {
			response.Error(w, http.StatusTooManyRequests, "otp_rate_limited", "Too many OTP requests. Please try again later.")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	if err := h.emailProvider.SendOTPEmail(r.Context(), user.Email, user.Name, otpCode, "password_reset"); err != nil {
		slog.Default().Error("failed to send otp email", "error", err, "purpose", "password_reset")
		response.Error(w, http.StatusInternalServerError, "internal_error", "Failed to send email")
		return
	}

	response.Success(w, http.StatusOK, map[string]interface{}{
		"message":    "OTP sent successfully",
		"expires_at": verification.ExpiresAt.Format(otpTimeFormatRFC3339),
	})
}

func (h *OTPHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var request resetPasswordRequest
	if err := decodeJSON(r, &request); err != nil {
		writeDecodeError(w, err)
		return
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.OTP = strings.TrimSpace(request.OTP)
	request.NewPassword = strings.TrimSpace(request.NewPassword)

	if request.Email == "" || request.OTP == "" || request.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, "missing_fields", "Email, OTP, and new_password are required")
		return
	}

	if len(request.NewPassword) < 8 {
		response.Error(w, http.StatusBadRequest, "invalid_password", "Password must be at least 8 characters")
		return
	}

	_, err := h.otpService.VerifyOTP(r.Context(), request.Email, "password_reset", request.OTP)
	if err != nil {
		switch err {
		case otp.ErrOTPNotFound:
			response.Error(w, http.StatusBadRequest, "otp_not_found", "OTP not found or expired. Please request a new one.")
		case otp.ErrOTPExpired:
			response.Error(w, http.StatusBadRequest, "otp_expired", "OTP has expired. Please request a new one.")
		case otp.ErrOTPMaxAttempts:
			response.Error(w, http.StatusBadRequest, "otp_max_attempts", "Maximum verification attempts exceeded. Please request a new OTP.")
		case otp.ErrOTPInvalid:
			response.Error(w, http.StatusBadRequest, "otp_invalid", "Invalid OTP code")
		default:
			response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	user, err := h.usersService.GetByEmail(r.Context(), request.Email)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	if err := h.usersService.UpdatePassword(r.Context(), user.ID, request.NewPassword); err != nil {
		writeUserError(w, err)
		return
	}

	if err := h.usersService.RevokeAllUserSessions(r.Context(), user.ID); err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_error", "Failed to revoke sessions")
		return
	}

	response.Success(w, http.StatusOK, map[string]interface{}{
		"message": "Password reset successfully",
	})
}
