package otp

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	verification Verification
	getErr       error
	recentCount  int

	created          bool
	expiredOld       bool
	incremented      bool
	markedVerified   bool
	lastCreateParams CreateParams
}

func (s *fakeStore) Create(ctx context.Context, params CreateParams) (Verification, error) {
	s.created = true
	s.lastCreateParams = params
	return s.verification, nil
}

func (s *fakeStore) GetActiveByEmailAndPurpose(ctx context.Context, email, purpose string) (Verification, error) {
	if s.getErr != nil {
		return Verification{}, s.getErr
	}
	return s.verification, nil
}

func (s *fakeStore) IncrementAttempt(ctx context.Context, id string) (Verification, error) {
	s.incremented = true
	s.verification.AttemptCount++
	return s.verification, nil
}

func (s *fakeStore) MarkVerified(ctx context.Context, id string) error {
	s.markedVerified = true
	return nil
}

func (s *fakeStore) ExpireOldVerifications(ctx context.Context, email, purpose string) error {
	s.expiredOld = true
	return nil
}

func (s *fakeStore) CountRecentRequests(ctx context.Context, email, purpose string, windowStart time.Time) (int, error) {
	return s.recentCount, nil
}

func TestVerifyOTPForUserRejectsUnlinkedVerificationBeforeConsuming(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, Config{
		MaxAttempts: 5,
	})

	hash, err := service.HashOTP("123456")
	if err != nil {
		t.Fatalf("HashOTP returned error: %v", err)
	}

	store.verification = Verification{
		ID:        "verification-1",
		Email:     "thomas@example.com",
		Purpose:   "signup",
		OTPHash:   hash,
		ExpiresAt: time.Now().Add(time.Minute),
	}

	_, err = service.VerifyOTPForUser(context.Background(), "thomas@example.com", "signup", "123456", "user-1")
	if !errors.Is(err, ErrOTPNotFound) {
		t.Fatalf("VerifyOTPForUser error = %v, want ErrOTPNotFound", err)
	}
	if store.markedVerified {
		t.Fatal("VerifyOTPForUser marked an unlinked verification as verified")
	}
	if store.incremented {
		t.Fatal("VerifyOTPForUser incremented attempts for an unlinked verification")
	}
}

func TestVerifyOTPReturnsExpiredForExpiredLatestVerification(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, Config{
		MaxAttempts: 5,
	})

	hash, err := service.HashOTP("123456")
	if err != nil {
		t.Fatalf("HashOTP returned error: %v", err)
	}

	store.verification = Verification{
		ID:        "verification-1",
		Email:     "thomas@example.com",
		Purpose:   "signup",
		OTPHash:   hash,
		ExpiresAt: time.Now().Add(-time.Minute),
	}

	_, err = service.VerifyOTP(context.Background(), "thomas@example.com", "signup", "123456")
	if !errors.Is(err, ErrOTPExpired) {
		t.Fatalf("VerifyOTP error = %v, want ErrOTPExpired", err)
	}
	if store.markedVerified {
		t.Fatal("VerifyOTP marked an expired verification as verified")
	}
}

func TestCreateVerificationRateLimitsBeforeCreatingOTP(t *testing.T) {
	store := &fakeStore{recentCount: 3}
	service := NewService(store, Config{
		ExpiryMinutes:     10,
		RequestsPerWindow: 3,
		WindowMins:        10,
	})

	_, _, err := service.CreateVerification(context.Background(), "thomas@example.com", "signup", nil)
	if !errors.Is(err, ErrOTPRateLimited) {
		t.Fatalf("CreateVerification error = %v, want ErrOTPRateLimited", err)
	}
	if store.created {
		t.Fatal("CreateVerification created an OTP after rate limit was reached")
	}
	if store.expiredOld {
		t.Fatal("CreateVerification expired old OTPs after rate limit was reached")
	}
}
