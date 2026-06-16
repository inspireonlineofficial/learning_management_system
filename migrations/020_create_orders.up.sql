-- Create orders table for book purchases
-- Supports soft-delete via deleted_at
-- idempotency_key is UNIQUE to prevent duplicate order processing
-- Requirements: 20.1, 20.3, 20.4

CREATE TYPE order_format AS ENUM ('physical', 'digital');
CREATE TYPE order_status AS ENUM ('placed', 'shipped', 'delivered', 'refunded', 'cancelled');

CREATE TABLE orders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id       UUID NOT NULL REFERENCES users(id),
    book_id          UUID NOT NULL REFERENCES books(id),
    format           order_format NOT NULL,
    amount           NUMERIC(10, 2) NOT NULL,
    currency         CHAR(3) NOT NULL,
    status           order_status NOT NULL DEFAULT 'placed',
    tracking_number  VARCHAR(100),
    idempotency_key  VARCHAR(255) UNIQUE,   -- prevents duplicate order creation
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ            -- soft-delete support
);

-- Indexes for common query patterns
CREATE INDEX idx_orders_student_id ON orders (student_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_book_id ON orders (book_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_status ON orders (status) WHERE deleted_at IS NULL;
