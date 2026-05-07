ALTER TABLE users
ADD COLUMN username text;

UPDATE users
SET username = 'user_' || substr(replace(id::text, '-', ''), 1, 12)
WHERE username IS NULL;

ALTER TABLE users
ALTER COLUMN username SET NOT NULL,
ADD CONSTRAINT users_username_unique UNIQUE (username),
ADD CONSTRAINT users_username_lowercase CHECK (username = lower(username)),
ADD CONSTRAINT users_username_format CHECK (username ~ '^[a-z0-9_]{3,30}$');

CREATE INDEX idx_users_username_search ON users (username text_pattern_ops);
