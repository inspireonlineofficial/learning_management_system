-- Bookshop fulfilment: shipping details for physical orders, fulfilment
-- timestamps, format/shipping on book purchase requests, and catalogue
-- guards. Reversible: see 044_bookshop_fulfilment.down.sql.

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS shipping_address JSONB,
    ADD COLUMN IF NOT EXISTS shipped_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS delivered_at     TIMESTAMPTZ;

ALTER TABLE purchase_requests
    ADD COLUMN IF NOT EXISTS book_format      VARCHAR(10),
    ADD COLUMN IF NOT EXISTS shipping_address JSONB;

ALTER TABLE purchase_requests
    ADD CONSTRAINT purchase_requests_book_format_check
    CHECK (book_format IS NULL OR book_format IN ('digital', 'physical'));

-- NOT VALID: existing rows are not re-checked (no rewrite, no failure on
-- legacy data); every new or updated row must satisfy the rule.
ALTER TABLE books ADD CONSTRAINT books_price_non_negative CHECK (price >= 0) NOT VALID;
ALTER TABLE books ADD CONSTRAINT books_stock_non_negative CHECK (physical_stock >= 0) NOT VALID;
