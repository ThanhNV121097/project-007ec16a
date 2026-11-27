-- The index behind "My orders": a customer's orders, newest first. Partial, so a guest order
-- (nearly all of the history) is never in it. This file runs outside a transaction, one
-- statement at a time, so it holds plain statements only. The drop comes first because an
-- interrupted build leaves an invalid index of this name that IF NOT EXISTS would then skip.
DROP INDEX CONCURRENTLY IF EXISTS idx_orders_account_id_placed_at;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_account_id_placed_at
    ON orders (account_id, placed_at DESC) WHERE account_id IS NOT NULL;
