-- name: CreateOrder :one
INSERT INTO orders (user_id, total_idr) VALUES ($1, $2) RETURNING *;

-- name: CreateOrderItem :exec
INSERT INTO order_items (order_id, product_id, name, unit_price_idr, quantity)
VALUES ($1, $2, $3, $4, $5);

-- Each transition has a guard on the current status, so a message handled
-- twice (or arriving late) cannot move an order that is already decided.
-- name: MarkAwaitingPayment :one
UPDATE orders SET status = 'awaiting_payment', updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING total_idr;

-- name: ConfirmOrder :execrows
UPDATE orders SET status = 'confirmed', updated_at = now() WHERE id = $1 AND status = 'awaiting_payment';

-- name: CancelOrder :execrows
UPDATE orders SET status = 'cancelled', cancel_reason = $2, updated_at = now()
WHERE id = $1 AND status IN ('pending', 'awaiting_payment');

-- name: GetOrder :one
SELECT * FROM orders WHERE id = $1;

-- name: ListOrdersByUser :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50;

-- name: ListItemsForOrders :many
SELECT * FROM order_items WHERE order_id = ANY($1::text[]) ORDER BY id;
