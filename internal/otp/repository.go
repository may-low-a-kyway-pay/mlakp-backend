package otp

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Verification struct {
	ID            string
	UserID        *string
	Email         string
	Purpose       string
	OTPHash       string
	ExpiresAt     time.Time
	VerifiedAt    *time.Time
	AttemptCount  int
	RequestCount  int
	LastRequestAt *time.Time
	CreatedAt     time.Time
}

type Store interface {
	Create(ctx context.Context, params CreateParams) (Verification, error)
	GetActiveByEmailAndPurpose(ctx context.Context, email, purpose string) (Verification, error)
	IncrementAttempt(ctx context.Context, id string) (Verification, error)
	MarkVerified(ctx context.Context, id string) error
	ExpireOldVerifications(ctx context.Context, email, purpose string) error
	CountRecentRequests(ctx context.Context, email, purpose string, windowStart time.Time) (int, error)
}

type CreateParams struct {
	UserID    *string
	Email     string
	Purpose   string
	OTPHash   string
	ExpiresAt time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, params CreateParams) (Verification, error) {
	sql := `
		INSERT INTO email_verifications (
			user_id, email, purpose, otp_hash, expires_at, request_count, last_request_at
		)
		VALUES ($1, $2, $3, $4, $5, 1, now())
		RETURNING id, user_id, email, purpose, otp_hash, expires_at, verified_at,
		          attempt_count, request_count, last_request_at, created_at
	`

	var v Verification
	var userID *string
	var verifiedAt *time.Time
	var lastRequestAt *time.Time

	err := r.pool.QueryRow(ctx, sql,
		params.UserID,
		params.Email,
		params.Purpose,
		params.OTPHash,
		params.ExpiresAt,
	).Scan(
		&v.ID,
		&userID,
		&v.Email,
		&v.Purpose,
		&v.OTPHash,
		&v.ExpiresAt,
		&verifiedAt,
		&v.AttemptCount,
		&v.RequestCount,
		&lastRequestAt,
		&v.CreatedAt,
	)

	if err != nil {
		return Verification{}, err
	}

	v.UserID = userID
	v.VerifiedAt = verifiedAt
	v.LastRequestAt = lastRequestAt

	return v, nil
}

func (r *Repository) GetActiveByEmailAndPurpose(ctx context.Context, email, purpose string) (Verification, error) {
	sql := `
		SELECT id, user_id, email, purpose, otp_hash, expires_at, verified_at,
		       attempt_count, request_count, last_request_at, created_at
		FROM email_verifications
		WHERE email = $1 AND purpose = $2
		  AND verified_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`

	var v Verification
	var userID *string
	var verifiedAt *time.Time
	var lastRequestAt *time.Time

	err := r.pool.QueryRow(ctx, sql, email, purpose).Scan(
		&v.ID,
		&userID,
		&v.Email,
		&v.Purpose,
		&v.OTPHash,
		&v.ExpiresAt,
		&verifiedAt,
		&v.AttemptCount,
		&v.RequestCount,
		&lastRequestAt,
		&v.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Verification{}, ErrOTPNotFound
		}
		return Verification{}, err
	}

	v.UserID = userID
	v.VerifiedAt = verifiedAt
	v.LastRequestAt = lastRequestAt

	return v, nil
}

func (r *Repository) IncrementAttempt(ctx context.Context, id string) (Verification, error) {
	sql := `
		UPDATE email_verifications
		SET attempt_count = attempt_count + 1
		WHERE id = $1
		RETURNING id, user_id, email, purpose, otp_hash, expires_at, verified_at,
		          attempt_count, request_count, last_request_at, created_at
	`

	var v Verification
	var userID *string
	var verifiedAt *time.Time
	var lastRequestAt *time.Time

	err := r.pool.QueryRow(ctx, sql, id).Scan(
		&v.ID,
		&userID,
		&v.Email,
		&v.Purpose,
		&v.OTPHash,
		&v.ExpiresAt,
		&verifiedAt,
		&v.AttemptCount,
		&v.RequestCount,
		&lastRequestAt,
		&v.CreatedAt,
	)

	if err != nil {
		return Verification{}, err
	}

	v.UserID = userID
	v.VerifiedAt = verifiedAt
	v.LastRequestAt = lastRequestAt

	return v, nil
}

func (r *Repository) MarkVerified(ctx context.Context, id string) error {
	sql := `
		UPDATE email_verifications
		SET verified_at = now()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, sql, id)
	return err
}

func (r *Repository) ExpireOldVerifications(ctx context.Context, email, purpose string) error {
	sql := `
		UPDATE email_verifications
		SET expires_at = now()
		WHERE email = $1 AND purpose = $2 AND verified_at IS NULL
		  AND expires_at > now()
	`

	_, err := r.pool.Exec(ctx, sql, email, purpose)
	return err
}

func (r *Repository) CountRecentRequests(ctx context.Context, email, purpose string, windowStart time.Time) (int, error) {
	sql := `
		SELECT COUNT(*)
		FROM email_verifications
		WHERE email = $1 AND purpose = $2 AND created_at >= $3
	`

	var count int
	err := r.pool.QueryRow(ctx, sql, email, purpose, windowStart).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
