CREATE TABLE email_verifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NULL REFERENCES users(id) ON DELETE CASCADE,
    email text NOT NULL,
    purpose text NOT NULL,
    otp_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    verified_at timestamptz NULL,
    attempt_count int NOT NULL DEFAULT 0,
    request_count int NOT NULL DEFAULT 1,
    last_request_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT email_verifications_purpose_check CHECK (
        purpose IN ('signup', 'password_reset')
    ),
    CONSTRAINT email_verifications_expires_future CHECK (
        expires_at > created_at
    )
);

CREATE INDEX idx_email_verifications_email_purpose
    ON email_verifications(email, purpose);
CREATE INDEX idx_email_verifications_user_id
    ON email_verifications(user_id);
CREATE INDEX idx_email_verifications_active
    ON email_verifications(email, expires_at)
    WHERE verified_at IS NULL;

ALTER TABLE users
    ADD COLUMN email_verified_at timestamptz NULL,
    ADD COLUMN verification_deadline timestamptz NULL;

UPDATE users
SET email_verified_at = now()
WHERE email_verified_at IS NULL;
