-- The orders still waiting to be paid, oldest first: the 30 second expiry check reads this and nothing else.
-- This file runs outside a transaction, one statement at a time, so it holds plain statements only.
-- The drop comes first because a failed concurrent build leaves an invalid index of this name.
DROP INDEX CONCURRENTLY IF EXISTS idx_orders_unpaid_open;
CREATE INDEX CONCURRENTLY idx_orders_unpaid_open ON orders (placed_at) WHERE NOT paid AND NOT is_sample AND status IN ('placed', 'confirmed', 'packed');
