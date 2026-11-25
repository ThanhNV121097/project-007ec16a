-- Local reset only: the server never runs a down file. Rolling this back after
-- the admin story has written records drops them. Triggers and indexes go
-- with their tables.
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS size_chart;
DROP TABLE IF EXISTS faqs;
DROP TABLE IF EXISTS payment_methods;
DROP TABLE IF EXISTS delivery_options;
DROP TABLE IF EXISTS shop_terms;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS lines;
DROP TABLE IF EXISTS categories;
