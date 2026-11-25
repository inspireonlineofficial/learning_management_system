-- Notifications table
-- Requirements: 22.1, 22.2
CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    type        VARCHAR(100) NOT NULL,
    title       VARCHAR(255) NOT NULL,
    body        TEXT NOT NULL,
    is_read     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast unread count and listing per user
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread_created
    ON notifications (user_id, is_read, created_at DESC);

-- Notification templates table
-- Requirements: 22.5
CREATE TABLE IF NOT EXISTS notification_templates (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type              VARCHAR(100) UNIQUE NOT NULL,
    channel           VARCHAR(20) NOT NULL CHECK (channel IN ('in_app', 'email', 'both')),
    subject_template  TEXT,
    body_template     TEXT NOT NULL,
    allowed_variables TEXT[] NOT NULL DEFAULT '{}',
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default templates for known notification types
INSERT INTO notification_templates (type, channel, subject_template, body_template, allowed_variables) VALUES
    ('grade_published',          'both',   'Your assignment has been graded',         'Hi {{student_name}}, your submission for "{{assignment_title}}" has been graded. Score: {{score}}.', ARRAY['student_name','assignment_title','score']),
    ('assignment_due_reminder',  'both',   'Assignment due soon: {{assignment_title}}','Hi {{student_name}}, your assignment "{{assignment_title}}" is due on {{due_at}}.', ARRAY['student_name','assignment_title','due_at']),
    ('live_session_scheduled',   'both',   'Live session scheduled: {{session_title}}','Hi {{user_name}}, a live session "{{session_title}}" has been scheduled for {{scheduled_at}}.', ARRAY['user_name','session_title','scheduled_at']),
    ('live_session_rescheduled', 'both',   'Live session rescheduled: {{session_title}}','Hi {{user_name}}, the live session "{{session_title}}" has been rescheduled to {{scheduled_at}}.', ARRAY['user_name','session_title','scheduled_at']),
    ('live_session_cancelled',   'both',   'Live session cancelled: {{session_title}}','Hi {{user_name}}, the live session "{{session_title}}" has been cancelled.', ARRAY['user_name','session_title']),
    ('forum_reply',              'in_app', NULL,                                       'Your post "{{post_title}}" received a new reply from {{commenter_name}}.', ARRAY['post_title','commenter_name']),
    ('points_milestone',         'in_app', NULL,                                       'Congratulations {{student_name}}! You have reached {{milestone}} points.', ARRAY['student_name','milestone']),
    ('course_approved',          'both',   'Your course has been approved',            'Hi {{teacher_name}}, your course "{{course_title}}" has been approved and is now published.', ARRAY['teacher_name','course_title']),
    ('course_rejected',          'both',   'Your course submission was rejected',      'Hi {{teacher_name}}, your course "{{course_title}}" was rejected. Reason: {{reason}}.', ARRAY['teacher_name','course_title','reason']),
    ('role_changed',             'both',   'Your account role has been updated',       'Hi {{user_name}}, your role has been changed from {{from_role}} to {{to_role}}.', ARRAY['user_name','from_role','to_role']),
    ('order_status_changed',     'both',   'Your order status has been updated',       'Hi {{student_name}}, your order #{{order_id}} status is now: {{status}}.', ARRAY['student_name','order_id','status']),
    ('content_removed',          'in_app', NULL,                                       'Your content has been removed by a moderator for violating community guidelines.', ARRAY[]::TEXT[])
ON CONFLICT (type) DO NOTHING;
