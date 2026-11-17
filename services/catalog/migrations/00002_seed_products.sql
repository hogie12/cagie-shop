-- +goose Up
INSERT INTO products (name, category, description, price_idr) VALUES
    ('Butter Croissant',  'pastry', 'Flaky, baked fresh every morning',      28000),
    ('Chocolate Eclair',  'pastry', 'Choux pastry with dark chocolate cream', 32000),
    ('Strawberry Tart',   'pastry', 'Vanilla custard and fresh strawberries', 45000),
    ('Red Rose Bouquet',  'flower', 'Twelve long-stem red roses',            350000),
    ('Sunflower Bunch',   'flower', 'Five sunflowers, wrapped in kraft paper', 120000),
    ('White Lily Vase',   'flower', 'Lilies in a small glass vase',          220000);

-- +goose Down
DELETE FROM products WHERE name IN (
    'Butter Croissant', 'Chocolate Eclair', 'Strawberry Tart',
    'Red Rose Bouquet', 'Sunflower Bunch', 'White Lily Vase'
);
