ALTER TABLE `product_templates`
  ADD COLUMN `type` varchar(32) NOT NULL DEFAULT 'other';

UPDATE `product_templates`
SET `type` = 'speaker'
WHERE `name` LIKE '%云喇叭%'
   OR `name` LIKE '%音响%'
   OR `name` LIKE '%喇叭%';
