-- Forum posts (aggregate root, soft-delete)
CREATE TABLE IF NOT EXISTS forum_posts (
    id           UUID PRIMARY KEY,
    author_id    UUID NOT NULL REFERENCES users(id),
    course_id    UUID REFERENCES courses(id),
    title        VARCHAR(255) NOT NULL,
    body_markdown TEXT NOT NULL,
    body_html    TEXT NOT NULL,
    upvotes      INT NOT NULL DEFAULT 0,
    flag_count   INT NOT NULL DEFAULT 0,
    status       VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'removed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_forum_posts_status_created ON forum_posts(status, created_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_forum_posts_author ON forum_posts(author_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_forum_posts_course ON forum_posts(course_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_forum_posts_flag_count ON forum_posts(flag_count) WHERE deleted_at IS NULL;

-- Forum comments (entity, soft-delete)
CREATE TABLE IF NOT EXISTS forum_comments (
    id           UUID PRIMARY KEY,
    post_id      UUID NOT NULL REFERENCES forum_posts(id),
    author_id    UUID NOT NULL REFERENCES users(id),
    body_markdown TEXT NOT NULL,
    body_html    TEXT NOT NULL,
    flag_count   INT NOT NULL DEFAULT 0,
    status       VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'removed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_forum_comments_post ON forum_comments(post_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_forum_comments_author ON forum_comments(author_id) WHERE deleted_at IS NULL;

-- Post upvotes (composite PK, value object)
CREATE TABLE IF NOT EXISTS post_upvotes (
    post_id  UUID NOT NULL REFERENCES forum_posts(id),
    user_id  UUID NOT NULL REFERENCES users(id),
    PRIMARY KEY (post_id, user_id)
);

-- Content flags (entity)
CREATE TABLE IF NOT EXISTS content_flags (
    id          UUID PRIMARY KEY,
    reporter_id UUID NOT NULL REFERENCES users(id),
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('post', 'comment')),
    target_id   UUID NOT NULL,
    reason      VARCHAR(30) NOT NULL CHECK (reason IN ('spam', 'offensive', 'misinformation', 'other')),
    note        TEXT,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'actioned', 'dismissed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_content_flags_target ON content_flags(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_content_flags_status ON content_flags(status, created_at);
