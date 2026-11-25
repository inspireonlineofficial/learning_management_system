-- Create payment_intents and payments tables for the Payments bounded context.
-- Raw card data is NEVER stored — all sensitive payment data is handled by the provider SDK.
-- Requirements: 24.1, 24.2, 24.3, 24.7

CREATE TYPE payment_intent_status AS ENUM ('pending', 'confirmed', 'failed');
CREATE TYPE payment_item_type AS ENUM ('course', 'book');
CREATE TYPE payment_status AS ENUM ('success', 'failed', 'refunded');

-- payment_intents: represents a checkout session initiated by a student.
-- client_secret is returned to the frontend once and never logged.
CREATE TABLE payment_intents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id          UUID NOT NULL REFERENCES users(id),
    item_type           payment_item_type NOT NULL,
    item_id             UUID NOT NULL,
    amount              NUMERIC(10, 2) NOT NULL,
    currency            CHAR(3) NOT NULL,
    status              payment_intent_status NOT NULL DEFAULT 'pending',
    provider_intent_id  VARCHAR(255),   -- opaque provider reference, never exposed in API
    client_secret       TEXT,           -- returned to frontend once; never logged
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_intents_student_id ON payment_intents (student_id);
CREATE INDEX idx_payment_intents_item ON payment_intents (item_id, item_type);

-- payments: records a completed or failed payment transaction.
-- idempotency_key UNIQUE constraint prevents duplicate processing on retried confirm requests.
-- Requirements: 24.3
CREATE TABLE payments (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_intent_id        UUID NOT NULL REFERENCES payment_intents(id),
    student_id               UUID NOT NULL REFERENCES users(id),
    idempotency_key          VARCHAR(255) UNIQUE NOT NULL,  -- prevents duplicate confirm processing
    provider_transaction_id  VARCHAR(255) NOT NULL,         -- opaque provider reference, never exposed
    amount                   NUMERIC(10, 2) NOT NULL,
    currency                 CHAR(3) NOT NULL,
    status                   payment_status NOT NULL,
    receipt_url              TEXT,
    paid_at                  TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_student_id ON payments (student_id);
CREATE INDEX idx_payments_intent_id ON payments (payment_intent_id);
CREATE INDEX idx_payments_status ON payments (status);
CREATE INDEX idx_payments_created_at ON payments (created_at);
