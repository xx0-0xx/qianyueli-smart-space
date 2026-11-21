ALTER TABLE `product_templates`
  ADD COLUMN `sort` int NOT NULL DEFAULT 0;

UPDATE `product_templates`
SET `sort` = `id`
WHERE `sort` = 0;
