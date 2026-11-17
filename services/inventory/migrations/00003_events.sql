-- +goose Up
-- One row per order: the stock taken for it. product_id has no foreign key
-- to anything because products live in the catalog service.
CREATE TABLE reservations (
    order_id   TEXT        NOT NULL,
    product_id BIGINT      NOT NULL,
    quantity   INT         NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id, product_id)
);

-- Event ids already handled. Checked in the same transaction as the stock
-- change, so a message delivered twice only changes stock once.
CREATE TABLE inbox (
    event_id     TEXT        PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE outbox (
    id           BIGSERIAL   PRIMARY KEY,
    event_id     TEXT        NOT NULL UNIQUE,
    routing_key  TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);
CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE outbox;
DROP TABLE inbox;
DROP TABLE reservations;
