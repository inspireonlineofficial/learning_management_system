-- Create books table for the bookshop catalog
-- Requirements: 19.1, 19.6

CREATE TYPE book_format AS ENUM ('physical', 'digital', 'both');

CREATE TABLE books (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title                   VARCHAR(255) NOT NULL,
    author                  VARCHAR(255) NOT NULL,
    subject                 VARCHAR(100),
    class_grade             VARCHAR(50),
    description             TEXT,
    format                  book_format NOT NULL,
    price                   NUMERIC(10, 2) NOT NULL,
    currency                CHAR(3) NOT NULL DEFAULT 'BDT',
    physical_stock          INT NOT NULL DEFAULT 0,
    digital_file_rustfs_key TEXT,           -- NEVER exposed in API responses
    preview_rustfs_key      TEXT,           -- NEVER exposed in API responses
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for public catalog queries (only active books)
CREATE INDEX idx_books_is_active ON books (is_active);
CREATE INDEX idx_books_subject ON books (subject) WHERE is_active = true;
CREATE INDEX idx_books_class_grade ON books (class_grade) WHERE is_active = true;
CREATE INDEX idx_books_format ON books (format) WHERE is_active = true;
