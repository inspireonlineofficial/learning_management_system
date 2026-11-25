-- Student refund requests and a record of every refund paid out.
-- Reversible: see 045_refund_requests.down.sql.

CREATE TABLE refund_requests (
    id           UUID PRIMARY KEY,
    order_id     UUID         NOT NULL REFERENCES orders(id),
    student_id   UUID         NOT NULL REFERENCES users(id),
    reason       TEXT         NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 2000),
    status       VARCHAR(10)  NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending', 'approved', 'rejected')),
    admin_note   TEXT,
    reviewed_by  UUID REFERENCES users(id),
    reviewed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- At most one open request per order.
CREATE UNIQUE INDEX refund_requests_one_pending ON refund_requests (order_id) WHERE status = 'pending';
CREATE INDEX refund_requests_status_created ON refund_requests (status, created_at DESC);
CREATE INDEX refund_requests_student ON refund_requests (student_id, created_at DESC);

-- One row per refunded order: how much was returned, how, and the
-- transaction reference, so refunds can be reconciled with bKash/bank
-- statements. The LMS records the payout; it does not move money.
CREATE TABLE order_refunds (
    id                 UUID PRIMARY KEY,
    order_id           UUID          NOT NULL UNIQUE REFERENCES orders(id),
    refund_request_id  UUID REFERENCES refund_requests(id),
    amount             NUMERIC(10, 2) NOT NULL CHECK (amount >= 0),
    currency           VARCHAR(3)    NOT NULL,
    payout_method      VARCHAR(20)   NOT NULL
                       CHECK (payout_method IN ('bkash', 'nagad', 'rocket', 'bank', 'cash', 'other', 'none')),
    payout_reference   VARCHAR(200),
    note               TEXT,
    processed_by       UUID REFERENCES users(id),
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
