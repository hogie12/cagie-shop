-- name: GetPaymentByOrder :one
SELECT * FROM payments WHERE order_id = $1;

-- Zero rows affected means this order already has a payment.
-- name: CreatePayment :execrows
INSERT INTO payments (order_id, amount_idr, status, reason) VALUES ($1, $2, $3, $4)
ON CONFLICT (order_id) DO NOTHING;
