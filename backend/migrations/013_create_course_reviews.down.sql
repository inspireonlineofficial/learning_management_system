-- Drop course_reviews table and indexes
DROP INDEX IF EXISTS idx_course_reviews_student_id;
DROP INDEX IF EXISTS idx_course_reviews_course_id;
DROP TABLE IF EXISTS course_reviews;
