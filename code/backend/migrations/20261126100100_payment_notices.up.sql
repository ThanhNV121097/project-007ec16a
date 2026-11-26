-- What the provider told the shop about an attempt, for each notice the shop accepted.
-- No card number is received and none is stored. Append-only, written only by internal/payments.
CREATE TABLE payment_notices (
    id                 uuid        NOT NULL DEFAULT gen_random_uuid(),
    attempt_id         uuid        NOT NULL,
    transaction_no     text        NOT NULL,
    bank_code          text        NOT NULL DEFAULT '',
    amount             bigint      NOT NULL,
    response_code      text        NOT NULL,
    transaction_status text        NOT NULL DEFAULT '',
    is_success         boolean     NOT NULL,
    paid_time          timestamptz,
    received_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_payment_notices PRIMARY KEY (id),
    CONSTRAINT fk_payment_notices_attempt_id FOREIGN KEY (attempt_id) REFERENCES payment_attempts (id) ON DELETE CASCADE ON UPDATE NO ACTION,
    CONSTRAINT ck_payment_notices_amount_not_negative CHECK (amount >= 0),
    CONSTRAINT ck_payment_notices_success_is_00 CHECK (NOT is_success OR (response_code = '00' AND transaction_status = '00'))
);

-- One success per attempt: the second line of defence behind the conditional UPDATE of orders.paid.
CREATE UNIQUE INDEX uq_payment_notices_attempt_id_success ON payment_notices (attempt_id) WHERE is_success;

-- Identical failure notices are one row: a customer's return address carries the same signed fields.
CREATE UNIQUE INDEX uq_payment_notices_attempt_id_failure
    ON payment_notices (attempt_id, transaction_no, response_code, transaction_status) WHERE NOT is_success;

CREATE INDEX idx_payment_notices_attempt_id ON payment_notices (attempt_id);
