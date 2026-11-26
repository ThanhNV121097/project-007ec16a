-- For a local reset only. This FAILS (23514) if a MoMo success row exists: those rows are
-- the only record of what MoMo took, so this file never deletes them. Drop the database
-- instead.
ALTER TABLE payment_notices DROP CONSTRAINT IF EXISTS ck_payment_notices_success_codes;
ALTER TABLE payment_notices DROP CONSTRAINT IF EXISTS ck_payment_notices_success_is_00;
ALTER TABLE payment_notices ADD CONSTRAINT ck_payment_notices_success_is_00 CHECK (
    NOT is_success OR (response_code = '00' AND transaction_status = '00')
);
