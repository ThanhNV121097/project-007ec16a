-- The unique index of orders.idempotency_key, built without a write lock on orders.
-- This file runs outside a transaction, one statement at a time, so it holds plain
-- statements only. The drop comes first because a failed concurrent build leaves an
-- invalid index of this name that IF NOT EXISTS would then skip.
DROP INDEX CONCURRENTLY IF EXISTS uq_orders_idempotency_key;
CREATE UNIQUE INDEX CONCURRENTLY uq_orders_idempotency_key ON orders (idempotency_key);
