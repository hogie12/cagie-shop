-- name: ListProducts :many
SELECT * FROM products
WHERE active AND (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category'))
ORDER BY id;

-- name: GetProduct :one
SELECT * FROM products WHERE id = $1 AND active;

-- name: CreateProduct :one
INSERT INTO products (name, category, description, price_idr, image_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET name = $2, category = $3, description = $4, price_idr = $5, image_url = $6, updated_at = now()
WHERE id = $1 AND active
RETURNING *;

-- name: DeactivateProduct :execrows
UPDATE products SET active = FALSE, updated_at = now() WHERE id = $1 AND active;
