-- Rollback payments migration
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS payment_intents;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_item_type;
DROP TYPE IF EXISTS payment_intent_status;
