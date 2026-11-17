-- +goose Up
-- product_id has no foreign key: products live in the catalog service's database.
CREATE TABLE stock (
    product_id BIGINT      PRIMARY KEY,
    quantity   INT         NOT NULL CHECK (quantity >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE stock;
