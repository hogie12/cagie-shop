-- +goose Up
ALTER TABLE orders ADD COLUMN cancel_reason TEXT;

-- Events waiting to be published. Written in the same transaction as the order.
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
ALTER TABLE orders DROP COLUMN cancel_reason;
