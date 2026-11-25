-- Pre-aggregated analytics tables for the Analytics bounded context.
-- These tables are populated by a scheduled hourly worker to avoid
-- full-table scans on hot read paths. (Requirement 23.6)

-- Enrollment stats: daily snapshot of enrollment counts per course
CREATE TABLE IF NOT EXISTS analytics_enrollment_stats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id       UUID REFERENCES courses(id) ON DELETE CASCADE,
    stat_date       DATE NOT NULL,
    total_enrolled  INT NOT NULL DEFAULT 0,
    free_enrolled   INT NOT NULL DEFAULT 0,
    paid_enrolled   INT NOT NULL DEFAULT 0,
    aggregated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(course_id, stat_date)
);

CREATE INDEX IF NOT EXISTS idx_analytics_enrollment_stats_date ON analytics_enrollment_stats(stat_date);
CREATE INDEX IF NOT EXISTS idx_analytics_enrollment_stats_course ON analytics_enrollment_stats(course_id, stat_date);

-- Progress stats: daily snapshot of completion rates per module per course
CREATE TABLE IF NOT EXISTS analytics_progress_stats (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id           UUID REFERENCES courses(id) ON DELETE CASCADE,
    module_id           UUID REFERENCES modules(id) ON DELETE CASCADE,
    stat_date           DATE NOT NULL,
    total_students      INT NOT NULL DEFAULT 0,
    completed_students  INT NOT NULL DEFAULT 0,
    in_progress_students INT NOT NULL DEFAULT 0,
    aggregated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(course_id, module_id, stat_date)
);

CREATE INDEX IF NOT EXISTS idx_analytics_progress_stats_course_date ON analytics_progress_stats(course_id, stat_date);

-- Revenue stats: daily revenue snapshot
CREATE TABLE IF NOT EXISTS analytics_revenue_stats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stat_date       DATE NOT NULL UNIQUE,
    total_revenue   NUMERIC(14,2) NOT NULL DEFAULT 0,
    course_revenue  NUMERIC(14,2) NOT NULL DEFAULT 0,
    book_revenue    NUMERIC(14,2) NOT NULL DEFAULT 0,
    aggregated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analytics_revenue_stats_date ON analytics_revenue_stats(stat_date);

-- Daily active users: count of distinct users who made at least one request per day
CREATE TABLE IF NOT EXISTS analytics_dau_stats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stat_date       DATE NOT NULL UNIQUE,
    active_users    INT NOT NULL DEFAULT 0,
    aggregated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analytics_dau_stats_date ON analytics_dau_stats(stat_date);
