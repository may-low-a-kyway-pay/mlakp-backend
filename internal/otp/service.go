package otp

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	digits     = "0123456789"
	otpLength  = 6
	bcryptCost = 10
)

type Config struct {
	ExpiryMinutes     int
	RequestCooldown   int
	MaxAttempts       int
	RequestsPerWindow int
	WindowMins        int
}

type Service struct {
	store  Store
	config Config
}

func NewService(store Store, config Config) *Service {
	return &Service{
		store:  store,
		config: config,
	}
}

func (s *Service) GenerateOTP() (string, error) {
	otp := make([]byte, otpLength)
	for i := 0; i < otpLength; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[num.Int64()]
	}
	return string(otp), nil
}

func (s *Service) HashOTP(otp string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(otp), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *Service) CompareOTP(hashedOTP, otp string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedOTP), []byte(otp)) == nil
}

func (s *Service) CreateVerification(ctx context.Context, email, purpose string, userID *string) (string, *Verification, error) {
	if purpose != "signup" && purpose != "password_reset" {
		return "", nil, ErrInvalidPurpose
	}

	windowStart := time.Now().Add(-time.Duration(s.config.WindowMins) * time.Minute)
	requestCount, err := s.store.CountRecentRequests(ctx, email, purpose, windowStart)
	if err != nil {
		return "", nil, err
	}

	if requestCount >= s.config.RequestsPerWindow {
		return "", nil, ErrOTPRateLimited
	}

	if err := s.store.ExpireOldVerifications(ctx, email, purpose); err != nil {
		return "", nil, err
	}

	otp, err := s.GenerateOTP()
	if err != nil {
		return "", nil, err
	}

	otpHash, err := s.HashOTP(otp)
	if err != nil {
		return "", nil, err
	}

	expiresAt := time.Now().Add(time.Duration(s.config.ExpiryMinutes) * time.Minute)

	verification, err := s.store.Create(ctx, CreateParams{
		UserID:    userID,
		Email:     email,
		Purpose:   purpose,
		OTPHash:   otpHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", nil, err
	}

	return otp, &verification, nil
}

func (s *Service) VerifyOTP(ctx context.Context, email, purpose, otp string) (*Verification, error) {
	if purpose != "signup" && purpose != "password_reset" {
		return nil, ErrInvalidPurpose
	}

	verification, err := s.store.GetActiveByEmailAndPurpose(ctx, email, purpose)
	if err != nil {
		if err == ErrOTPNotFound {
			return nil, ErrOTPNotFound
		}
		return nil, err
	}

	return s.verify(ctx, verification, otp)
}

func (s *Service) VerifyOTPForUser(ctx context.Context, email, purpose, otp, userID string) (*Verification, error) {
	if purpose != "signup" && purpose != "password_reset" {
		return nil, ErrInvalidPurpose
	}

	verification, err := s.store.GetActiveByEmailAndPurpose(ctx, email, purpose)
	if err != nil {
		if err == ErrOTPNotFound {
			return nil, ErrOTPNotFound
		}
		return nil, err
	}

	if verification.UserID == nil || *verification.UserID != userID {
		return nil, ErrOTPNotFound
	}

	return s.verify(ctx, verification, otp)
}

func (s *Service) verify(ctx context.Context, verification Verification, otp string) (*Verification, error) {
	if time.Now().After(verification.ExpiresAt) {
		return nil, ErrOTPExpired
	}

	if verification.AttemptCount >= s.config.MaxAttempts {
		return nil, ErrOTPMaxAttempts
	}

	if !s.CompareOTP(verification.OTPHash, otp) {
		updated, err := s.store.IncrementAttempt(ctx, verification.ID)
		if err != nil {
			return nil, err
		}
		if updated.AttemptCount >= s.config.MaxAttempts {
			return nil, ErrOTPMaxAttempts
		}
		return nil, ErrOTPInvalid
	}

	if err := s.store.MarkVerified(ctx, verification.ID); err != nil {
		return nil, err
	}

	verification.VerifiedAt = new(time.Time)
	*verification.VerifiedAt = time.Now()

	return &verification, nil
}

func (s *Service) CheckCooldown(ctx context.Context, email, purpose string) error {
	verification, err := s.store.GetActiveByEmailAndPurpose(ctx, email, purpose)
	if err != nil {
		if err == ErrOTPNotFound {
			return nil
		}
		return err
	}

	if verification.LastRequestAt == nil {
		return nil
	}

	cooldownUntil := verification.LastRequestAt.Add(time.Duration(s.config.RequestCooldown) * time.Second)
	if time.Now().Before(cooldownUntil) {
		return ErrOTPCooldown
	}

	return nil
}
