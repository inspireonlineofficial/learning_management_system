-- Drop courses table and indexes
DROP INDEX IF EXISTS idx_courses_published_at;
DROP INDEX IF EXISTS idx_courses_slug;
DROP INDEX IF EXISTS idx_courses_teacher_id;
DROP INDEX IF EXISTS idx_courses_status_deleted;
DROP TABLE IF EXISTS courses;
