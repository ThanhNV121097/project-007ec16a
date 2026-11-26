-- One attempt to pay an order on a provider's page: every "Pay now", "Try again" and placing with
-- VNPay writes a new row with a new reference. Append-only, written only by internal/payments.
CREATE TABLE payment_attempts (
    id         uuid        NOT NULL DEFAULT gen_random_uuid(),
    order_id   text        NOT NULL,
    provider   text        NOT NULL,
    reference  text        NOT NULL,
    amount     bigint      NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_payment_attempts PRIMARY KEY (id),
    CONSTRAINT fk_payment_attempts_order_id FOREIGN KEY (order_id) REFERENCES orders (id) ON DELETE CASCADE ON UPDATE NO ACTION,
    CONSTRAINT uq_payment_attempts_reference UNIQUE (reference),
    CONSTRAINT ck_payment_attempts_provider CHECK (provider IN ('vnpay', 'momo')),
    CONSTRAINT ck_payment_attempts_reference_length CHECK (length(reference) BETWEEN 1 AND 100),
    CONSTRAINT ck_payment_attempts_amount_not_negative CHECK (amount >= 0)
);

CREATE INDEX idx_payment_attempts_order_id ON payment_attempts (order_id);
