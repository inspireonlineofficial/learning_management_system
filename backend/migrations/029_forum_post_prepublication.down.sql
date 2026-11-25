DROP TABLE IF EXISTS forum_post_reviews;
ALTER TABLE forum_posts DROP CONSTRAINT IF EXISTS forum_posts_status_check;
ALTER TABLE forum_posts ALTER COLUMN status SET DEFAULT 'active';
UPDATE forum_posts SET status = 'active' WHERE status IN ('pending', 'rejected');
ALTER TABLE forum_posts ADD CONSTRAINT forum_posts_status_check
    CHECK (status IN ('active', 'removed'));
