-- For local resets only: rolling this back on a shop with real orders deletes
-- every customer's access token and every idempotency key.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS ck_orders_idempotency_key_length;
ALTER TABLE orders DROP COLUMN IF EXISTS idempotency_key;
DROP TABLE IF EXISTS counters;
DROP TABLE IF EXISTS order_tokens;
