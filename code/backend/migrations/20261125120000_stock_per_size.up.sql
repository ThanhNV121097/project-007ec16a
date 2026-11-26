-- One count of pairs per shoe and EU size (docs/architecture/erd/checkout-and-placing-an-order.md, migration 4).
-- A shoe's sizes, sizes_low, sizes_out and stock stop being columns of products
-- and are derived from these counts through the view product_stock.

CREATE TABLE product_sizes (
    product_id text         NOT NULL,
    size       text         NOT NULL,
    size_value numeric(5,2) NOT NULL,
    pairs      integer      NOT NULL DEFAULT 0,
    created_at timestamptz  NOT NULL DEFAULT now(),
    updated_at timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT product_sizes_pkey PRIMARY KEY (product_id, size),
    CONSTRAINT fk_product_sizes_product_id FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE CASCADE,
    CONSTRAINT ck_product_sizes_size_not_blank CHECK (btrim(size) <> '' AND length(size) <= 20),
    CONSTRAINT ck_product_sizes_size_value_positive CHECK (size_value > 0),
    CONSTRAINT ck_product_sizes_pairs_not_negative CHECK (pairs >= 0)
);

CREATE INDEX idx_product_sizes_size_in_stock ON product_sizes (size, product_id) WHERE pairs > 0;

CREATE TRIGGER trg_product_sizes_updated_at BEFORE UPDATE ON product_sizes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 0 pairs is sold out, 1 or 2 is last pairs, 3 or more is in neither list.
CREATE VIEW product_stock AS
SELECT p.id AS product_id,
       coalesce(array_agg(s.size ORDER BY s.size_value, s.size)
                FILTER (WHERE s.size IS NOT NULL), '{}'::text[])        AS sizes,
       coalesce(array_agg(s.size ORDER BY s.size_value, s.size)
                FILTER (WHERE s.pairs BETWEEN 1 AND 2), '{}'::text[])   AS sizes_low,
       coalesce(array_agg(s.size ORDER BY s.size_value, s.size)
                FILTER (WHERE s.pairs = 0), '{}'::text[])               AS sizes_out,
       coalesce(sum(s.pairs), 0)::integer                               AS stock
FROM products p
LEFT JOIN product_sizes s ON s.product_id = p.id
GROUP BY p.id;

-- Backfill: every size a shoe lists gets 0 pairs when it is in sizes_out, 1 when
-- it is in sizes_low and 3 otherwise, so the derived lists equal the seeded ones.
-- A size that does not start with a number gives a NULL size_value and fails the
-- whole file, naming the shoe and the size.
INSERT INTO product_sizes (product_id, size, size_value, pairs)
SELECT p.id, s.size,
       round(
           substring(s.size from '^[0-9]+(?:\.[0-9]+)?')::numeric
           + coalesce(substring(s.size from ' ([0-9]+)/[0-9]+$')::numeric
                      / nullif(substring(s.size from ' [0-9]+/([0-9]+)$')::numeric, 0), 0),
           2),
       CASE WHEN s.size = ANY (p.sizes_out) THEN 0
            WHEN s.size = ANY (p.sizes_low) THEN 1
            ELSE 3 END
FROM products p
CROSS JOIN LATERAL unnest(p.sizes) AS s(size);

-- The assertion that comes before the drop: for every shoe the view's three
-- lists equal the seeded columns as sets. A failure names every shoe that
-- differs and rolls the whole file back.
DO $$
DECLARE
    bad text;
BEGIN
    SELECT string_agg(p.id, ', ' ORDER BY p.id) INTO bad
    FROM products p
    JOIN product_stock v ON v.product_id = p.id
    WHERE ARRAY(SELECT unnest(p.sizes) ORDER BY 1) IS DISTINCT FROM ARRAY(SELECT unnest(v.sizes) ORDER BY 1)
       OR ARRAY(SELECT unnest(p.sizes_low) ORDER BY 1) IS DISTINCT FROM ARRAY(SELECT unnest(v.sizes_low) ORDER BY 1)
       OR ARRAY(SELECT unnest(p.sizes_out) ORDER BY 1) IS DISTINCT FROM ARRAY(SELECT unnest(v.sizes_out) ORDER BY 1);
    IF bad IS NOT NULL THEN
        RAISE EXCEPTION 'product_stock does not reproduce the seeded size lists of: %', bad;
    END IF;
END $$;

-- Plain DROP COLUMN, no CASCADE: an object that still depends on a column fails
-- the file instead of vanishing.
ALTER TABLE products
    DROP COLUMN sizes,
    DROP COLUMN sizes_low,
    DROP COLUMN sizes_out,
    DROP COLUMN stock;
