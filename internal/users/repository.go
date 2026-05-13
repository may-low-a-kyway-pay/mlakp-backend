package users

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"mlakp-backend/internal/postgres/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrNotFound         = errors.New("user not found")
	ErrEmailConflict    = errors.New("email already registered")
	ErrUsernameConflict = errors.New("username already registered")
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, name, username, email, passwordHash string) (PrivateUser, error) {
	user, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Name:         name,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if constraintName(err) == "users_username_unique" {
			return PrivateUser{}, ErrUsernameConflict
		}
		if constraintName(err) == "users_email_unique" || isUniqueViolation(err) {
			return PrivateUser{}, ErrEmailConflict
		}
		return PrivateUser{}, err
	}

	return privateUserFromFields(user.ID, user.Name, user.Username, user.Email, user.PasswordHash, user.EmailVerifiedAt, user.VerificationDeadline, user.CreatedAt, user.UpdatedAt), nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (PrivateUser, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PrivateUser{}, ErrNotFound
		}
		return PrivateUser{}, err
	}

	return privateUserFromFields(user.ID, user.Name, user.Username, user.Email, user.PasswordHash, user.EmailVerifiedAt, user.VerificationDeadline, user.CreatedAt, user.UpdatedAt), nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (User, error) {
	uuid, err := parseUUID(id)
	if err != nil {
		return User{}, ErrNotFound
	}

	user, err := r.queries.GetUserByID(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return publicUserFromFields(user.ID, user.Name, user.Username, user.Email, user.EmailVerifiedAt, user.VerificationDeadline, user.CreatedAt, user.UpdatedAt), nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (User, error) {
	user, err := r.queries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return publicUserFromFields(user.ID, user.Name, user.Username, user.Email, user.EmailVerifiedAt, user.VerificationDeadline, user.CreatedAt, user.UpdatedAt), nil
}

func (r *Repository) SearchByUsername(ctx context.Context, query string, limit int32) ([]User, error) {
	rows, err := r.queries.SearchUsersByUsername(ctx, sqlc.SearchUsersByUsernameParams{
		Column1: query,
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(rows))
	for _, row := range rows {
		users = append(users, publicUserFromFields(row.ID, row.Name, row.Username, row.Email, row.EmailVerifiedAt, row.VerificationDeadline, row.CreatedAt, row.UpdatedAt))
	}

	return users, nil
}

func (r *Repository) UpdateUsername(ctx context.Context, id, username string) (User, error) {
	uuid, err := parseUUID(id)
	if err != nil {
		return User{}, ErrNotFound
	}

	user, err := r.queries.UpdateUserUsername(ctx, sqlc.UpdateUserUsernameParams{
		ID:       uuid,
		Username: username,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		if constraintName(err) == "users_username_unique" || isUniqueViolation(err) {
			return User{}, ErrUsernameConflict
		}
		return User{}, err
	}

	return publicUserFromFields(user.ID, user.Name, user.Username, user.Email, user.EmailVerifiedAt, user.VerificationDeadline, user.CreatedAt, user.UpdatedAt), nil
}

func (r *Repository) MarkEmailVerified(ctx context.Context, id string) (User, error) {
	uuid, err := parseUUID(id)
	if err != nil {
		return User{}, ErrNotFound
	}

	user, err := r.queries.MarkEmailVerified(ctx, uuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return publicUserFromFields(user.ID, user.Name, user.Username, user.Email, user.EmailVerifiedAt, user.VerificationDeadline, user.CreatedAt, user.UpdatedAt), nil
}

func (r *Repository) RevokeAllUserSessions(ctx context.Context, userID string) error {
	uuid, err := parseUUID(userID)
	if err != nil {
		return ErrNotFound
	}

	return r.queries.RevokeAllUserSessions(ctx, uuid)
}

func (r *Repository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	uuid, err := parseUUID(userID)
	if err != nil {
		return ErrNotFound
	}

	return r.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           uuid,
		PasswordHash: passwordHash,
	})
}

func privateUserFromFields(
	id pgtype.UUID,
	name string,
	username string,
	email string,
	passwordHash string,
	emailVerifiedAt pgtype.Timestamptz,
	verificationDeadline pgtype.Timestamptz,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) PrivateUser {
	return PrivateUser{
		User:         publicUserFromFields(id, name, username, email, emailVerifiedAt, verificationDeadline, createdAt, updatedAt),
		PasswordHash: passwordHash,
	}
}

func publicUserFromFields(
	id pgtype.UUID,
	name string,
	username string,
	email string,
	emailVerifiedAt pgtype.Timestamptz,
	verificationDeadline pgtype.Timestamptz,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) User {
	var verifiedAt *time.Time
	if emailVerifiedAt.Valid {
		t := emailVerifiedAt.Time
		verifiedAt = &t
	}

	var deadline *time.Time
	if verificationDeadline.Valid {
		t := verificationDeadline.Time
		deadline = &t
	}

	return User{
		ID:                   formatUUID(id),
		Name:                 name,
		Username:             username,
		Email:                email,
		EmailVerifiedAt:      verifiedAt,
		VerificationDeadline: deadline,
		CreatedAt:            createdAt.Time,
		UpdatedAt:            updatedAt.Time,
	}
}

func parseUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	if !uuid.Valid {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid")
	}

	return uuid, nil
}

func formatUUID(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}

	encoded := hex.EncodeToString(uuid.Bytes[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}

func isUniqueViolation(err error) bool {
	return constraintName(err) != "" && pgErrorCode(err) == "23505"
}

func constraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
