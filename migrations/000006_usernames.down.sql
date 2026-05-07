DROP INDEX IF EXISTS idx_users_username_search;

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_username_format,
DROP CONSTRAINT IF EXISTS users_username_lowercase,
DROP CONSTRAINT IF EXISTS users_username_unique,
DROP COLUMN IF EXISTS username;
