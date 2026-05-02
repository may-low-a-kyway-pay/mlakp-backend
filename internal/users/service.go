package users

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidName     = errors.New("name must be between 1 and 120 characters")
	ErrInvalidEmail    = errors.New("email is invalid")
	ErrInvalidPassword = errors.New("password must be at least 8 characters")
)

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(hash, password string) bool
}

type Store interface {
	Create(ctx context.Context, name, email, passwordHash string) (PrivateUser, error)
	GetByEmail(ctx context.Context, email string) (PrivateUser, error)
	GetByID(ctx context.Context, id string) (User, error)
}

type Service struct {
	repository     Store
	passwordHasher PasswordHasher
}

func NewService(repository Store, passwordHasher PasswordHasher) *Service {
	return &Service{
		repository:     repository,
		passwordHasher: passwordHasher,
	}
}

func (s *Service) Register(ctx context.Context, name, email, password string) (User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)

	if err := validateName(name); err != nil {
		return User{}, err
	}
	if err := validateEmail(email); err != nil {
		return User{}, err
	}
	if len(password) < 8 {
		return User{}, ErrInvalidPassword
	}

	passwordHash, err := s.passwordHasher.HashPassword(password)
	if err != nil {
		return User{}, err
	}

	user, err := s.repository.Create(ctx, name, email, passwordHash)
	if err != nil {
		return User{}, err
	}

	return user.User, nil
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	if err := validateEmail(email); err != nil {
		return User{}, ErrNotFound
	}

	user, err := s.repository.GetByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	if !s.passwordHasher.ComparePassword(user.PasswordHash, password) {
		return User{}, ErrNotFound
	}

	return user.User, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (User, error) {
	return s.repository.GetByID(ctx, id)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateName(name string) error {
	if len(name) < 1 || len(name) > 120 {
		return ErrInvalidName
	}

	return nil
}

func validateEmail(email string) error {
	if email == "" || len(email) > 254 {
		return ErrInvalidEmail
	}

	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return ErrInvalidEmail
	}
	if strings.ContainsAny(email, " \t\r\n") {
		return ErrInvalidEmail
	}

	return nil
}
