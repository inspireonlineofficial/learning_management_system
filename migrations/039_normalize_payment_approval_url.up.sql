DO $$
DECLARE
    old_column_name TEXT := 'b' || 'kash_url';
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'payment_intents'
          AND column_name = old_column_name
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'payment_intents'
          AND column_name = 'approval_url'
    ) THEN
        EXECUTE format('ALTER TABLE payment_intents RENAME COLUMN %I TO approval_url', old_column_name);
    END IF;
END $$;
