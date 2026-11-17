-- +goose Up
CREATE TABLE orders (
    id         TEXT        PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id    TEXT        NOT NULL,
    status     TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'cancelled')),
    total_idr  BIGINT      NOT NULL CHECK (total_idr >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX orders_user_idx ON orders (user_id, created_at DESC);

-- Name and price are copied from the catalog at purchase time, so later
-- catalog changes never rewrite order history. product_id has no foreign key
-- because products live in another service's database.
CREATE TABLE order_items (
    id             BIGSERIAL PRIMARY KEY,
    order_id       TEXT   NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id     BIGINT NOT NULL,
    name           TEXT   NOT NULL,
    unit_price_idr BIGINT NOT NULL CHECK (unit_price_idr >= 0),
    quantity       INT    NOT NULL CHECK (quantity BETWEEN 1 AND 99),
    UNIQUE (order_id, product_id)
);

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;
