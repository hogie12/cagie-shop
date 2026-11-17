-- +goose Up
CREATE TABLE products (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    category    TEXT        NOT NULL CHECK (category IN ('pastry', 'flower')),
    description TEXT        NOT NULL DEFAULT '',
    price_idr   BIGINT      NOT NULL CHECK (price_idr >= 0),
    image_url   TEXT        NOT NULL DEFAULT '',
    active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX products_category_idx ON products (category) WHERE active;

-- +goose Down
DROP TABLE products;
