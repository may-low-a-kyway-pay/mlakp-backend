package users

import "time"

type User struct {
	ID        string
	Name      string
	Username  string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PrivateUser struct {
	User
	PasswordHash string
}
