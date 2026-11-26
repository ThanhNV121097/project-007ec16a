-- What placing an order needs (docs/architecture/erd/checkout-and-placing-an-order.md, migration 5):
-- the access tokens of an order, the order-number counter and the idempotency key.

CREATE TABLE order_tokens (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   text        NOT NULL,
    token_hash bytea       NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_order_tokens_order_id FOREIGN KEY (order_id) REFERENCES orders (id) ON DELETE CASCADE,
    CONSTRAINT uq_order_tokens_token_hash UNIQUE (token_hash),
    CONSTRAINT ck_order_tokens_hash_length CHECK (octet_length(token_hash) = 32)
);

CREATE INDEX idx_order_tokens_order_id ON order_tokens (order_id);

CREATE TABLE counters (
    name       text        PRIMARY KEY,
    value      bigint      NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_counters_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_counters_value_not_negative CHECK (value >= 0)
);

CREATE TRIGGER trg_counters_updated_at BEFORE UPDATE ON counters
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Seeded with the highest number the orders hold, read from the data.
INSERT INTO counters (name, value)
SELECT 'order_number', coalesce(max(substring(number from '^DOI-([0-9]+)$')::bigint), 0)
FROM orders;

ALTER TABLE orders ADD COLUMN idempotency_key text;
ALTER TABLE orders ADD CONSTRAINT ck_orders_idempotency_key_length
    CHECK (idempotency_key IS NULL OR length(idempotency_key) BETWEEN 8 AND 100);

-- What the order's expected date reads must be there, so a gap fails at boot and
-- not on a customer's order.
DO $$
BEGIN
    ASSERT (SELECT value FROM counters WHERE name = 'order_number')
        >= coalesce((SELECT max(substring(number from '^DOI-([0-9]+)$')::bigint) FROM orders), 0),
        'counters.order_number must be at least the highest order number';
    ASSERT EXISTS (SELECT 1 FROM shop_terms WHERE id = 'shop' AND standard_days ~ '[0-9]'),
        'shop_terms shop must hold a number of days in standard_days';
    ASSERT EXISTS (SELECT 1 FROM shop_terms WHERE id = 'shop' AND dispatch_cutoff ~ '^[0-9]{2}:[0-9]{2}$'),
        'shop_terms shop must hold dispatch_cutoff as HH:MM';
    ASSERT EXISTS (SELECT 1 FROM delivery_options WHERE id = 'standard'),
        'delivery_options must hold standard';
END $$;
