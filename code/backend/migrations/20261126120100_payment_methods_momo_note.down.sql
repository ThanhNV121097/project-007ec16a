-- Reversible with no loss; a note changed since is left alone.
UPDATE payment_methods
SET note = 'Confirm the payment in the MoMo app; the order is marked paid at once.'
WHERE id = 'momo' AND note = 'Confirm the payment in the MoMo app.';
