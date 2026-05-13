package otp

import "errors"

var (
	ErrOTPExpired      = errors.New("otp has expired")
	ErrOTPInvalid      = errors.New("invalid otp")
	ErrOTPNotFound     = errors.New("otp not found")
	ErrOTPMaxAttempts  = errors.New("maximum verification attempts exceeded")
	ErrOTPRateLimited  = errors.New("too many otp requests")
	ErrOTPCooldown     = errors.New("please wait before requesting another otp")
	ErrEmailNotFound   = errors.New("user not found")
	ErrInvalidPurpose  = errors.New("invalid purpose")
	ErrAlreadyVerified = errors.New("email already verified")
)
