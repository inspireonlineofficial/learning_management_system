ALTER TABLE books DROP CONSTRAINT IF EXISTS books_stock_non_negative;
ALTER TABLE books DROP CONSTRAINT IF EXISTS books_price_non_negative;

ALTER TABLE purchase_requests DROP CONSTRAINT IF EXISTS purchase_requests_book_format_check;
ALTER TABLE purchase_requests
    DROP COLUMN IF EXISTS shipping_address,
    DROP COLUMN IF EXISTS book_format;

ALTER TABLE orders
    DROP COLUMN IF EXISTS delivered_at,
    DROP COLUMN IF EXISTS shipped_at,
    DROP COLUMN IF EXISTS shipping_address;
