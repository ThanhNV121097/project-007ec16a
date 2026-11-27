-- orders.account_id: the customer account that owns an order (plan item 15, "Customer accounts").
-- NULL for a guest order, for an order a staff or admin session placed, and for every order
-- that exists today, so no row needs a backfill. Deleting a customer leaves the order and
-- clears the link. The key is added unchecked and then validated: every row is NULL, which
-- a foreign key accepts, so the validation scan only reads.

ALTER TABLE orders ADD COLUMN account_id uuid;

ALTER TABLE orders ADD CONSTRAINT fk_orders_account_id
    FOREIGN KEY (account_id) REFERENCES users (id) ON DELETE SET NULL ON UPDATE NO ACTION NOT VALID;

ALTER TABLE orders VALIDATE CONSTRAINT fk_orders_account_id;
