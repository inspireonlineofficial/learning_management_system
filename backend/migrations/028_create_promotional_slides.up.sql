CREATE TABLE IF NOT EXISTS promotional_slides (
    id             UUID PRIMARY KEY,
    title          VARCHAR(160) NOT NULL,
    subtitle       TEXT NOT NULL DEFAULT '',
    link_url       TEXT NOT NULL DEFAULT '',
    media_key      TEXT NOT NULL,
    media_type     VARCHAR(80) NOT NULL,
    duration_ms    INT NOT NULL DEFAULT 5000 CHECK (duration_ms > 0),
    position       INT NOT NULL DEFAULT 0,
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_promotional_slides_active_position
    ON promotional_slides (is_active, position, created_at);
