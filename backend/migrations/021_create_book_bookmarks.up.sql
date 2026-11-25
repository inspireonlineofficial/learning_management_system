-- Create book_bookmarks table for tracking reading progress
-- UNIQUE(student_id, book_id) enforces one bookmark per student per book
-- Requirements: 19.5

CREATE TABLE book_bookmarks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id     UUID NOT NULL REFERENCES users(id),
    book_id        UUID NOT NULL REFERENCES books(id),
    last_page_read INT NOT NULL DEFAULT 1,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, book_id)
);

CREATE INDEX idx_book_bookmarks_student_id ON book_bookmarks (student_id);
