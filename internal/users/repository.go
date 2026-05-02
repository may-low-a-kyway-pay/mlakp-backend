package users

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"mlakp-backend/internal/postgres/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrEmailConflict = errors.New("email already registered")
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, name, email, passwordHash string) (PrivateUser, error) {
	user, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return PrivateUser{}, ErrEmailConflict
		}
		return PrivateUser{}, err
	}

	return privateUserFromSQLC(user), nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (PrivateUser, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PrivateUser{}, ErrNotFound
		}
		return PrivateUser{}, err
	}

	return privateUserFromSQLC(user), nil
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

	return publicUserFromSQLC(user), nil
}

func privateUserFromSQLC(user sqlc.User) PrivateUser {
	return PrivateUser{
		User:         publicUserFromSQLC(user),
		PasswordHash: user.PasswordHash,
	}
}

func publicUserFromSQLC(user sqlc.User) User {
	return User{
		ID:        formatUUID(user.ID),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
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
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
