ALTER TABLE users
    DROP COLUMN IF EXISTS verification_deadline,
    DROP COLUMN IF EXISTS email_verified_at;

DROP INDEX IF EXISTS idx_email_verifications_active;
DROP INDEX IF EXISTS idx_email_verifications_user_id;
DROP INDEX IF EXISTS idx_email_verifications_email_purpose;

DROP TABLE IF EXISTS email_verifications;