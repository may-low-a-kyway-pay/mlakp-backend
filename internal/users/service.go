package users

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidName     = errors.New("name must be between 1 and 120 characters")
	ErrInvalidUsername = errors.New("username must be 3 to 30 lowercase letters, numbers, or underscores")
	ErrInvalidEmail    = errors.New("email is invalid")
	ErrInvalidPassword = errors.New("password must be at least 8 characters")
)

const maxSearchResults = 10

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)
var usernameSearchPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(hash, password string) bool
}

type Store interface {
	Create(ctx context.Context, name, username, email, passwordHash string) (PrivateUser, error)
	GetByEmail(ctx context.Context, email string) (PrivateUser, error)
	GetByID(ctx context.Context, id string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	SearchByUsername(ctx context.Context, query string, limit int32) ([]User, error)
	UpdateUsername(ctx context.Context, id, username string) (User, error)
	MarkEmailVerified(ctx context.Context, id string) (User, error)
	RevokeAllUserSessions(ctx context.Context, userID string) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
}

func (s *Service) GetByEmail(ctx context.Context, email string) (PrivateUser, error) {
	return s.repository.GetByEmail(ctx, email)
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

func (s *Service) Register(ctx context.Context, name, username, email, password string) (User, error) {
	name = strings.TrimSpace(name)
	username = normalizeUsername(username)
	email = normalizeEmail(email)

	if err := validateName(name); err != nil {
		return User{}, err
	}
	if err := validateUsername(username); err != nil {
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

	user, err := s.repository.Create(ctx, name, username, email, passwordHash)
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

func (s *Service) GetByUsername(ctx context.Context, username string) (User, error) {
	username = normalizeUsername(username)
	if err := validateUsername(username); err != nil {
		return User{}, err
	}

	return s.repository.GetByUsername(ctx, username)
}

func (s *Service) SearchByUsername(ctx context.Context, query string) ([]User, error) {
	query = normalizeUsername(query)
	if len(query) < 2 {
		return []User{}, nil
	}
	// Search is typeahead-friendly: incomplete or invalid prefixes simply return no matches.
	if !usernameSearchPattern.MatchString(query) {
		return []User{}, nil
	}

	return s.repository.SearchByUsername(ctx, query, maxSearchResults)
}

func (s *Service) UpdateUsername(ctx context.Context, id, username string) (User, error) {
	username = normalizeUsername(username)
	if strings.TrimSpace(id) == "" {
		return User{}, ErrNotFound
	}
	if err := validateUsername(username); err != nil {
		return User{}, err
	}

	return s.repository.UpdateUsername(ctx, id, username)
}

func (s *Service) MarkEmailVerified(ctx context.Context, userID string) (User, error) {
	return s.repository.MarkEmailVerified(ctx, userID)
}

func (s *Service) UpdatePassword(ctx context.Context, userID, password string) error {
	if len(password) < 8 {
		return ErrInvalidPassword
	}

	passwordHash, err := s.passwordHasher.HashPassword(password)
	if err != nil {
		return err
	}

	return s.repository.UpdatePassword(ctx, userID, passwordHash)
}

func (s *Service) RevokeAllUserSessions(ctx context.Context, userID string) error {
	return s.repository.RevokeAllUserSessions(ctx, userID)
}

func (s *Service) GetVerificationStatus(user *User) VerificationStatus {
	if user.EmailVerifiedAt != nil {
		return VerificationStatus{
			IsVerified: true,
			Status:     "verified",
		}
	}

	if user.VerificationDeadline == nil {
		return VerificationStatus{
			IsVerified: false,
			Status:     "pending",
		}
	}

	daysRemaining := int(time.Until(*user.VerificationDeadline).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	return VerificationStatus{
		IsVerified:    false,
		DaysRemaining: daysRemaining,
		Deadline:      user.VerificationDeadline,
		Status:        "pending_grace_period",
	}
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
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

func validateUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}

	return nil
}
