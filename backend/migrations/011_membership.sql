CREATE TABLE IF NOT EXISTS `member_levels` (
  `id` BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  `store_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(64) NOT NULL,
  `discount_bp` INT NOT NULL DEFAULT 10000,
  `sort` INT NOT NULL DEFAULT 0,
  `status` TINYINT NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_member_level_store_name` (`store_id`, `name`),
  KEY `idx_member_level_store_sort` (`store_id`, `sort`),
  KEY `idx_member_level_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `store_members` (
  `id` BIGINT UNSIGNED NOT NULL,
  `created_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  `store_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `level_id` BIGINT UNSIGNED NOT NULL,
  `expires_at` BIGINT NULL,
  `status` TINYINT NOT NULL DEFAULT 1,
  `remark` VARCHAR(255) NULL,
  `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_store_member` (`store_id`, `user_id`),
  KEY `idx_store_member_level` (`store_id`, `level_id`),
  KEY `idx_store_member_user` (`user_id`),
  KEY `idx_store_member_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE `orders` ADD COLUMN `original_amount` BIGINT NOT NULL DEFAULT 0;
ALTER TABLE `orders` ADD COLUMN `member_discount` BIGINT NOT NULL DEFAULT 0;
ALTER TABLE `orders` ADD COLUMN `member_level_id` BIGINT UNSIGNED NULL;
ALTER TABLE `orders` ADD COLUMN `member_level_name` VARCHAR(64) NULL;

UPDATE `orders` SET `original_amount` = `total_amount` WHERE `original_amount` = 0;
