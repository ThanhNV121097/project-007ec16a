-- Removes the link between orders and accounts. The orders themselves stay.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_account_id;
ALTER TABLE orders DROP COLUMN IF EXISTS account_id;
