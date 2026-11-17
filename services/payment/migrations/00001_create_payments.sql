-- +goose Up
-- One payment per order. The UNIQUE order_id is the idempotency guard: a
-- request handled twice cannot create (or announce) a second payment.
CREATE TABLE payments (
    id         TEXT        PRIMARY KEY DEFAULT gen_random_uuid()::text,
    order_id   TEXT        NOT NULL UNIQUE,
    amount_idr BIGINT      NOT NULL CHECK (amount_idr > 0),
    status     TEXT        NOT NULL CHECK (status IN ('succeeded', 'failed')),
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
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
DROP TABLE payments;
