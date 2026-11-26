-- The ten collections the site reads, and the pair lines of an order.
-- Table design: docs/architecture/erd/site-data-from-the-database.md.
-- Every collection table starts with the same six columns: id (the record's
-- text slug), seq (the stable order of every list: the seed's, then the order
-- records were added), source and invented (bookkeeping, never served to a
-- caller with no session) and the two timestamps.

CREATE TABLE categories (
    id         text        PRIMARY KEY,
    seq        bigint      GENERATED ALWAYS AS IDENTITY,
    source     text        NOT NULL DEFAULT '',
    invented   text[]      NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    name       text        NOT NULL,
    blurb      text        NOT NULL DEFAULT '',
    image      text        NOT NULL DEFAULT '',
    CONSTRAINT ck_categories_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_categories_seq UNIQUE (seq),
    CONSTRAINT ck_categories_name_not_blank CHECK (btrim(name) <> '')
);

CREATE TABLE lines (
    id         text        PRIMARY KEY,
    seq        bigint      GENERATED ALWAYS AS IDENTITY,
    source     text        NOT NULL DEFAULT '',
    invented   text[]      NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    name       text        NOT NULL,
    blurb      text        NOT NULL DEFAULT '',
    since      integer,
    fit_note   text        NOT NULL DEFAULT '',
    CONSTRAINT ck_lines_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_lines_seq UNIQUE (seq),
    CONSTRAINT ck_lines_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_lines_since_positive CHECK (since > 0)
);

CREATE TABLE products (
    id           text         PRIMARY KEY,
    seq          bigint       GENERATED ALWAYS AS IDENTITY,
    source       text         NOT NULL DEFAULT '',
    invented     text[]       NOT NULL DEFAULT '{}',
    created_at   timestamptz  NOT NULL DEFAULT now(),
    updated_at   timestamptz  NOT NULL DEFAULT now(),
    name         text         NOT NULL,
    model        text         NOT NULL DEFAULT '',
    line_id      text,
    category_id  text,
    price        bigint       NOT NULL,
    price_listed text         NOT NULL DEFAULT '',
    status       text         NOT NULL DEFAULT 'draft',
    featured     boolean      NOT NULL DEFAULT false,
    is_new       boolean      NOT NULL DEFAULT false,
    description  text         NOT NULL DEFAULT '',
    code         text         NOT NULL DEFAULT '',
    colourway    text         NOT NULL DEFAULT '',
    colour       text         NOT NULL DEFAULT '',
    swatch       text         NOT NULL DEFAULT '',
    upper        text         NOT NULL DEFAULT '',
    lining       text         NOT NULL DEFAULT '',
    sole         text         NOT NULL DEFAULT '',
    fit          text         NOT NULL DEFAULT '',
    care         text         NOT NULL DEFAULT '',
    released     date         NOT NULL DEFAULT ((now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date),
    rating       numeric(2,1) NOT NULL DEFAULT 0,
    review_count integer      NOT NULL DEFAULT 0,
    image        text         NOT NULL DEFAULT '',
    image_back   text         NOT NULL DEFAULT '',
    image_detail text         NOT NULL DEFAULT '',
    sizes        text[]       NOT NULL DEFAULT '{}',
    sizes_low    text[]       NOT NULL DEFAULT '{}',
    sizes_out    text[]       NOT NULL DEFAULT '{}',
    stock        integer      NOT NULL DEFAULT 0,
    CONSTRAINT ck_products_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_products_seq UNIQUE (seq),
    CONSTRAINT fk_products_line_id FOREIGN KEY (line_id) REFERENCES lines (id) ON DELETE RESTRICT,
    CONSTRAINT fk_products_category_id FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE RESTRICT,
    CONSTRAINT ck_products_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_products_price_not_negative CHECK (price >= 0),
    CONSTRAINT ck_products_status CHECK (status IN ('live', 'draft')),
    CONSTRAINT ck_products_colour CHECK (colour IN ('', 'black', 'white', 'pink', 'purple', 'brown', 'yellow', 'blue', 'grey', 'green')),
    CONSTRAINT ck_products_rating_range CHECK (rating BETWEEN 0 AND 5),
    CONSTRAINT ck_products_review_count_not_negative CHECK (review_count >= 0),
    CONSTRAINT ck_products_stock_not_negative CHECK (stock >= 0)
);

CREATE UNIQUE INDEX uq_products_featured ON products (featured) WHERE featured;
CREATE INDEX idx_products_status_released_seq ON products (status, released DESC, seq);
CREATE INDEX idx_products_line_id ON products (line_id);
CREATE INDEX idx_products_category_id ON products (category_id);

CREATE TABLE reviews (
    id         text        PRIMARY KEY,
    seq        bigint      GENERATED ALWAYS AS IDENTITY,
    source     text        NOT NULL DEFAULT '',
    invented   text[]      NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    product_id text        NOT NULL,
    name       text        NOT NULL,
    rating     smallint,
    date       date        NOT NULL DEFAULT ((now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date),
    size_bought text       NOT NULL DEFAULT '',
    fit        text        NOT NULL DEFAULT '',
    body       text        NOT NULL DEFAULT '',
    status     text        NOT NULL DEFAULT 'live',
    CONSTRAINT ck_reviews_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_reviews_seq UNIQUE (seq),
    CONSTRAINT fk_reviews_product_id FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE CASCADE,
    CONSTRAINT ck_reviews_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_reviews_rating_range CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT ck_reviews_fit CHECK (fit IN ('', 'small', 'true', 'large')),
    CONSTRAINT ck_reviews_status CHECK (status IN ('live', 'hidden'))
);

CREATE INDEX idx_reviews_product_id_date ON reviews (product_id, date DESC, seq);

CREATE TABLE shop_terms (
    id              text         PRIMARY KEY,
    seq             bigint       GENERATED ALWAYS AS IDENTITY,
    source          text         NOT NULL DEFAULT '',
    invented        text[]       NOT NULL DEFAULT '{}',
    created_at      timestamptz  NOT NULL DEFAULT now(),
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    standard_days   text         NOT NULL DEFAULT '',
    dispatch_cutoff text         NOT NULL DEFAULT '',
    reply_time      text         NOT NULL DEFAULT '',
    rating          numeric(2,1) NOT NULL DEFAULT 0,
    review_count    integer      NOT NULL DEFAULT 0,
    CONSTRAINT ck_shop_terms_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_shop_terms_seq UNIQUE (seq),
    CONSTRAINT ck_shop_terms_rating_range CHECK (rating BETWEEN 0 AND 5),
    CONSTRAINT ck_shop_terms_review_count_not_negative CHECK (review_count >= 0)
);

CREATE TABLE delivery_options (
    id         text        PRIMARY KEY,
    seq        bigint      GENERATED ALWAYS AS IDENTITY,
    source     text        NOT NULL DEFAULT '',
    invented   text[]      NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    name       text        NOT NULL,
    days       text        NOT NULL DEFAULT '',
    fee        bigint      NOT NULL DEFAULT 0,
    note       text        NOT NULL DEFAULT '',
    CONSTRAINT ck_delivery_options_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_delivery_options_seq UNIQUE (seq),
    CONSTRAINT ck_delivery_options_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_delivery_options_fee_not_negative CHECK (fee >= 0)
);

CREATE TABLE payment_methods (
    id         text        PRIMARY KEY,
    seq        bigint      GENERATED ALWAYS AS IDENTITY,
    source     text        NOT NULL DEFAULT '',
    invented   text[]      NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    name       text        NOT NULL,
    note       text        NOT NULL DEFAULT '',
    online     boolean     NOT NULL DEFAULT false,
    icon       text        NOT NULL DEFAULT '',
    CONSTRAINT ck_payment_methods_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_payment_methods_seq UNIQUE (seq),
    CONSTRAINT ck_payment_methods_name_not_blank CHECK (btrim(name) <> '')
);

CREATE TABLE faqs (
    id         text        PRIMARY KEY,
    seq        bigint      GENERATED ALWAYS AS IDENTITY,
    source     text        NOT NULL DEFAULT '',
    invented   text[]      NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    topic      text        NOT NULL,
    question   text        NOT NULL,
    answer     text        NOT NULL DEFAULT '',
    CONSTRAINT ck_faqs_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_faqs_seq UNIQUE (seq),
    CONSTRAINT ck_faqs_topic CHECK (topic IN ('shipping', 'returns', 'sizing', 'payment', 'orders')),
    CONSTRAINT ck_faqs_question_not_blank CHECK (btrim(question) <> '')
);

CREATE TABLE size_chart (
    id         text         PRIMARY KEY,
    seq        bigint       GENERATED ALWAYS AS IDENTITY,
    source     text         NOT NULL DEFAULT '',
    invented   text[]       NOT NULL DEFAULT '{}',
    created_at timestamptz  NOT NULL DEFAULT now(),
    updated_at timestamptz  NOT NULL DEFAULT now(),
    eu         text         NOT NULL,
    uk         text         NOT NULL DEFAULT '',
    us_m       text         NOT NULL DEFAULT '',
    us_w       text         NOT NULL DEFAULT '',
    cm         numeric(4,1),
    CONSTRAINT ck_size_chart_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_size_chart_seq UNIQUE (seq),
    CONSTRAINT uq_size_chart_eu UNIQUE (eu),
    CONSTRAINT ck_size_chart_cm_positive CHECK (cm > 0)
);

CREATE TABLE orders (
    id                 text        PRIMARY KEY,
    seq                bigint      GENERATED ALWAYS AS IDENTITY,
    source             text        NOT NULL DEFAULT '',
    invented           text[]      NOT NULL DEFAULT '{}',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    number             text        NOT NULL,
    placed_at          timestamptz NOT NULL DEFAULT now(),
    status             text        NOT NULL DEFAULT 'placed',
    payment_method_id  text        NOT NULL,
    paid               boolean     NOT NULL DEFAULT false,
    delivery_option_id text        NOT NULL,
    delivery_fee       bigint      NOT NULL DEFAULT 0,
    carrier            text        NOT NULL DEFAULT '',
    tracking           text        NOT NULL DEFAULT '',
    eta                date,
    customer           text        NOT NULL,
    phone              text        NOT NULL,
    email              text        NOT NULL DEFAULT '',
    address            text        NOT NULL,
    city               text        NOT NULL,
    note               text        NOT NULL DEFAULT '',
    total              bigint      NOT NULL,
    is_sample          boolean     NOT NULL DEFAULT false,
    CONSTRAINT ck_orders_id_slug CHECK (id ~ '^[a-z0-9][a-z0-9._-]*$' AND length(id) <= 100),
    CONSTRAINT uq_orders_seq UNIQUE (seq),
    CONSTRAINT uq_orders_number UNIQUE (number),
    CONSTRAINT fk_orders_payment_method_id FOREIGN KEY (payment_method_id) REFERENCES payment_methods (id) ON DELETE RESTRICT,
    CONSTRAINT fk_orders_delivery_option_id FOREIGN KEY (delivery_option_id) REFERENCES delivery_options (id) ON DELETE RESTRICT,
    CONSTRAINT ck_orders_number_format CHECK (number ~ '^DOI-[0-9]+$'),
    CONSTRAINT ck_orders_status CHECK (status IN ('placed', 'confirmed', 'packed', 'shipped', 'delivered', 'cancelled')),
    CONSTRAINT ck_orders_delivery_fee_not_negative CHECK (delivery_fee >= 0),
    CONSTRAINT ck_orders_customer_not_blank CHECK (btrim(customer) <> ''),
    CONSTRAINT ck_orders_phone_not_blank CHECK (btrim(phone) <> ''),
    CONSTRAINT ck_orders_address_not_blank CHECK (btrim(address) <> ''),
    CONSTRAINT ck_orders_city_not_blank CHECK (btrim(city) <> ''),
    CONSTRAINT ck_orders_total_not_negative CHECK (total >= 0)
);

CREATE INDEX idx_orders_placed_at ON orders (placed_at DESC);
CREATE INDEX idx_orders_status_placed_at ON orders (status, placed_at DESC);
CREATE INDEX idx_orders_payment_method_id_placed_at ON orders (payment_method_id, placed_at DESC);

CREATE TABLE order_items (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id   text        NOT NULL,
    line_no    smallint    NOT NULL,
    product_id text        NOT NULL,
    size       text        NOT NULL,
    qty        integer     NOT NULL,
    unit_price bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_order_items_order_id_line_no UNIQUE (order_id, line_no),
    CONSTRAINT fk_order_items_order_id FOREIGN KEY (order_id) REFERENCES orders (id) ON DELETE CASCADE,
    CONSTRAINT fk_order_items_product_id FOREIGN KEY (product_id) REFERENCES products (id) ON DELETE RESTRICT,
    CONSTRAINT ck_order_items_line_no_positive CHECK (line_no > 0),
    CONSTRAINT ck_order_items_qty_positive CHECK (qty > 0),
    CONSTRAINT ck_order_items_unit_price_not_negative CHECK (unit_price >= 0)
);

CREATE INDEX idx_order_items_product_id ON order_items (product_id);

-- updated_at moves by trigger on the ten collection tables.
CREATE TRIGGER trg_categories_updated_at BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_lines_updated_at BEFORE UPDATE ON lines FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_products_updated_at BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_reviews_updated_at BEFORE UPDATE ON reviews FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_shop_terms_updated_at BEFORE UPDATE ON shop_terms FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_delivery_options_updated_at BEFORE UPDATE ON delivery_options FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_payment_methods_updated_at BEFORE UPDATE ON payment_methods FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_faqs_updated_at BEFORE UPDATE ON faqs FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_size_chart_updated_at BEFORE UPDATE ON size_chart FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION set_updated_at();
