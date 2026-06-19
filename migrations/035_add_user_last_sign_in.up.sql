ALTER TABLE users
    ADD COLUMN IF NOT EXISTS last_sign_in_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_last_sign_in_at
    ON users(last_sign_in_at)
    WHERE deleted_at IS NULL;
