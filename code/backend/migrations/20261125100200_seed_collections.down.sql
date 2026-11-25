-- Local reset only: the server never runs a down file. It deletes the seeded
-- ids in reverse foreign-key order and stops with a foreign-key error once
-- later orders refer to a seeded shoe, which is the right answer.
DELETE FROM order_items WHERE order_id IN ('doi-24804', 'doi-24805', 'doi-24806', 'doi-24807', 'doi-24808', 'doi-24809', 'doi-24810', 'doi-24811', 'doi-24812', 'doi-24813', 'doi-24814', 'doi-24815', 'doi-24816', 'doi-24817');
DELETE FROM orders WHERE id IN ('doi-24804', 'doi-24805', 'doi-24806', 'doi-24807', 'doi-24808', 'doi-24809', 'doi-24810', 'doi-24811', 'doi-24812', 'doi-24813', 'doi-24814', 'doi-24815', 'doi-24816', 'doi-24817');
DELETE FROM size_chart WHERE id IN ('eu-35-1-2', 'eu-36', 'eu-36-2-3', 'eu-37-1-3', 'eu-38', 'eu-38-2-3', 'eu-39-1-3', 'eu-40', 'eu-40-2-3', 'eu-41-1-3', 'eu-42', 'eu-42-2-3', 'eu-43-1-3', 'eu-44', 'eu-44-2-3', 'eu-45-1-3', 'eu-46');
DELETE FROM faqs WHERE id IN ('f-delivery-time', 'f-delivery-where', 'f-delivery-change', 'f-delivery-gift', 'f-returns-size', 'f-returns-wrong', 'f-sizing-small', 'f-sizing-between', 'f-sizing-kids', 'f-payment-how', 'f-payment-genuine', 'f-orders-number');
DELETE FROM payment_methods WHERE id IN ('cod', 'bank', 'vnpay', 'momo');
DELETE FROM delivery_options WHERE id = 'standard';
DELETE FROM shop_terms WHERE id = 'shop';
DELETE FROM reviews WHERE id IN (SELECT 'r-' || lpad(n::text, 2, '0') FROM generate_series(1, 36) AS g(n));
DELETE FROM products WHERE id IN ('samba-og-w-core-black', 'samba-og-w-crystal-white', 'samba-og-w-wonder-quartz', 'handball-spezial-w-earth-strata', 'gazelle-indoor-solar-gold', 'adizero-adios-wonder-blue', 'adizero-evo-sl-charcoal', 'megaride-s2-cloud-white', 'megaride-bones-halo-green', 'megaride-ag-xl-core-black', 'samba-og-kids-wonder-mauve', 'samba-og-kids-cloud-white');
DELETE FROM lines WHERE id IN ('samba', 'handball-spezial', 'gazelle', 'adizero', 'megaride');
DELETE FROM categories WHERE id IN ('women', 'men', 'kids');
