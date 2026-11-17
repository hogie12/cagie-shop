-- +goose Up
-- Product ids 1-6 match the catalog service's seed data (pastries 1-3, flowers 4-6).
INSERT INTO stock (product_id, quantity) VALUES
    (1, 20), (2, 20), (3, 20),
    (4, 10), (5, 10), (6, 10);

-- +goose Down
DELETE FROM stock WHERE product_id IN (1, 2, 3, 4, 5, 6);
