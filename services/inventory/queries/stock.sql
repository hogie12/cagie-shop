-- name: GetStock :one
SELECT * FROM stock WHERE product_id = $1;

-- name: SetStock :one
INSERT INTO stock (product_id, quantity) VALUES ($1, $2)
ON CONFLICT (product_id) DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = now()
RETURNING *;

-- The WHERE clause makes check-and-decrement one atomic statement,
-- so concurrent reservations can never push stock below zero.
-- name: ReserveStock :execrows
UPDATE stock SET quantity = quantity - $2, updated_at = now()
WHERE product_id = $1 AND quantity >= $2;

-- name: ReleaseStock :execrows
UPDATE stock SET quantity = quantity + $2, updated_at = now() WHERE product_id = $1;

-- name: InsertInbox :execrows
INSERT INTO inbox (event_id) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: CreateReservation :exec
INSERT INTO reservations (order_id, product_id, quantity) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: ReleaseReservations :many
UPDATE reservations SET released_at = now()
WHERE order_id = $1 AND released_at IS NULL
RETURNING product_id, quantity;
