-- Only a verified MoMo notice marks an order paid, so the first note's second clause was
-- untrue. Changed only while the record still has its first text: a note the owner
-- reworded in the Studio is left alone, and a second run matches nothing.
UPDATE payment_methods
SET note = 'Confirm the payment in the MoMo app.'
WHERE id = 'momo' AND note = 'Confirm the payment in the MoMo app; the order is marked paid at once.';
