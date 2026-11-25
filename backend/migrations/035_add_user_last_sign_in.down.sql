DROP INDEX IF EXISTS idx_users_last_sign_in_at;

ALTER TABLE users
    DROP COLUMN IF EXISTS last_sign_in_at;
