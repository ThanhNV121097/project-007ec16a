-- For local resets only: this deletes the only shop-side record of what the provider took.
-- The server never runs a down file.
DROP TABLE IF EXISTS payment_notices;
