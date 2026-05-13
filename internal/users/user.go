package users

import "time"

type User struct {
	ID                   string
	Name                 string
	Username             string
	Email                string
	EmailVerifiedAt      *time.Time
	VerificationDeadline *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type PrivateUser struct {
	User
	PasswordHash string
}

type VerificationStatus struct {
	IsVerified    bool
	DaysRemaining int
	Deadline      *time.Time
	Status        string
}
