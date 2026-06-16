-- Create system_settings (singleton) and system_setting_history (append-only) tables.
-- Requirements: 25.1, 25.3

-- system_settings: singleton row (id=1) holding all platform-wide configuration.
-- maintenance_mode and feature_flags are enforced at the middleware/service layer.
CREATE TABLE system_settings (
    id                      INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    platform_name           VARCHAR(100)    NOT NULL DEFAULT 'LMS',
    default_timezone        VARCHAR(50)     NOT NULL DEFAULT 'UTC',
    oauth_providers_enabled TEXT[]          NOT NULL DEFAULT '{google}',
    maintenance_mode        BOOLEAN         NOT NULL DEFAULT false,
    feature_flags           JSONB           NOT NULL DEFAULT '{}',
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_by              UUID REFERENCES users(id)
);

-- Seed the singleton row so it always exists.
INSERT INTO system_settings (id) VALUES (1) ON CONFLICT DO NOTHING;

-- system_setting_history: append-only audit trail of every settings change.
-- diff contains only the changed fields; snapshot contains the full state after the change.
-- No UPDATE or DELETE is granted to the application DB user on this table.
CREATE TABLE system_setting_history (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    changed_by  UUID        NOT NULL REFERENCES users(id),
    diff        JSONB       NOT NULL,
    snapshot    JSONB       NOT NULL,
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_setting_history_changed_at ON system_setting_history (changed_at DESC);
CREATE INDEX idx_system_setting_history_changed_by ON system_setting_history (changed_by);
