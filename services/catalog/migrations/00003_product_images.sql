-- +goose Up
-- Photo paths are served by the web app (apps/web/public). Only rows still
-- without a photo are touched, so an admin's later choice is never overwritten.
UPDATE products SET image_url = '/images/products/butter-croissant.webp' WHERE name = 'Butter Croissant' AND image_url = '';
UPDATE products SET image_url = '/images/products/chocolate-eclair.webp' WHERE name = 'Chocolate Eclair' AND image_url = '';
UPDATE products SET image_url = '/images/products/strawberry-tart.webp'  WHERE name = 'Strawberry Tart'  AND image_url = '';
UPDATE products SET image_url = '/images/products/red-rose-bouquet.webp' WHERE name = 'Red Rose Bouquet' AND image_url = '';
UPDATE products SET image_url = '/images/products/sunflower-bunch.webp'  WHERE name = 'Sunflower Bunch'  AND image_url = '';
UPDATE products SET image_url = '/images/products/white-lily-vase.webp'  WHERE name = 'White Lily Vase'  AND image_url = '';

-- +goose Down
UPDATE products SET image_url = '' WHERE image_url LIKE '/images/products/%';
