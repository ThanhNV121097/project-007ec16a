-- For local resets only: this deletes every attempt and, through the cascade, every notice.
-- The server never runs a down file.
DROP TABLE IF EXISTS payment_attempts;
