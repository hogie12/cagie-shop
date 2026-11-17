-- +goose Up
-- pending -> awaiting_payment (stock held) -> confirmed | cancelled
ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'awaiting_payment', 'confirmed', 'cancelled'));

-- +goose Down
UPDATE orders SET status = 'pending' WHERE status = 'awaiting_payment';
ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'confirmed', 'cancelled'));
