ALTER TABLE forum_posts DROP CONSTRAINT IF EXISTS forum_posts_status_check;
ALTER TABLE forum_posts ALTER COLUMN status SET DEFAULT 'pending';
ALTER TABLE forum_posts ADD CONSTRAINT forum_posts_status_check
    CHECK (status IN ('pending', 'active', 'rejected', 'removed'));

CREATE TABLE IF NOT EXISTS forum_post_reviews (
    id          UUID PRIMARY KEY,
    post_id     UUID NOT NULL REFERENCES forum_posts(id),
    reviewer_id UUID NOT NULL REFERENCES users(id),
    action      VARCHAR(20) NOT NULL CHECK (action IN ('approve', 'reject')),
    reason      TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_forum_post_reviews_post ON forum_post_reviews(post_id, created_at DESC);
