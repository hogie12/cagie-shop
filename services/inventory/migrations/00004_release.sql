-- +goose Up
-- Set when the stock of a reservation was put back (payment failed). The
-- column makes releasing idempotent: a reservation is released at most once.
ALTER TABLE reservations ADD COLUMN released_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE reservations DROP COLUMN released_at;
