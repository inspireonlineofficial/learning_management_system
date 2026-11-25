CREATE TABLE IF NOT EXISTS purchase_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id),
    item_type VARCHAR(20) NOT NULL CHECK (item_type IN ('course', 'book')),
    item_id UUID NOT NULL,
    file_name TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    rejection_reason TEXT,
    result_enrollment_id UUID REFERENCES enrollments(id),
    result_order_id UUID REFERENCES orders(id),
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_requests_student_status_created
    ON purchase_requests(student_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_requests_item
    ON purchase_requests(item_type, item_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_purchase_requests_status_created
    ON purchase_requests(status, created_at DESC);
