-- Puts the four columns back and fills them from the counts, so sold pairs stay
-- sold and a sold-out size stays listed. For local resets: a running shop
-- never runs this.
ALTER TABLE products
    ADD COLUMN sizes     text[]  NOT NULL DEFAULT '{}',
    ADD COLUMN sizes_low text[]  NOT NULL DEFAULT '{}',
    ADD COLUMN sizes_out text[]  NOT NULL DEFAULT '{}',
    ADD COLUMN stock     integer NOT NULL DEFAULT 0;

ALTER TABLE products ADD CONSTRAINT ck_products_stock_not_negative CHECK (stock >= 0);

UPDATE products p
SET sizes = v.sizes, sizes_low = v.sizes_low, sizes_out = v.sizes_out, stock = v.stock
FROM product_stock v
WHERE v.product_id = p.id;

DROP VIEW product_stock;
DROP TABLE product_sizes;
