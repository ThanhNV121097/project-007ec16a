-- A MoMo success has no '00' anywhere: its result code is the integer 0 and it has no
-- transaction status. The VNPay story's check accepted only VNPay's '00' / '00', so the
-- database would refuse the first MoMo payment. The two arms are disjoint (VNPay's
-- response codes are two digits, MoMo's is a plain integer), so a row can satisfy only
-- its own provider's. The application decides is_success per provider; this is the
-- backstop. Every existing row satisfies the first arm, so no row is rewritten.
ALTER TABLE payment_notices DROP CONSTRAINT IF EXISTS ck_payment_notices_success_is_00;
ALTER TABLE payment_notices DROP CONSTRAINT IF EXISTS ck_payment_notices_success_codes;
ALTER TABLE payment_notices ADD CONSTRAINT ck_payment_notices_success_codes CHECK (
    NOT is_success
    OR (response_code = '00' AND transaction_status = '00')
    OR (response_code = '0' AND transaction_status = '')
);
