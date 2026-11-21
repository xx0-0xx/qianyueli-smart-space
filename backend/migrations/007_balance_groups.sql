CREATE TABLE IF NOT EXISTS `balance_groups` (
  `id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(128) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_balance_groups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `balance_accounts` (
  `id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned NOT NULL,
  `group_id` bigint unsigned NOT NULL,
  `balance` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_group` (`user_id`,`group_id`),
  KEY `idx_balance_accounts_group_id` (`group_id`),
  KEY `idx_balance_accounts_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE `stores`
  ADD COLUMN `balance_group_id` bigint unsigned NOT NULL DEFAULT 0,
  ADD KEY `idx_stores_balance_group_id` (`balance_group_id`);

ALTER TABLE `balance_logs`
  ADD COLUMN `store_id` bigint unsigned DEFAULT NULL,
  ADD COLUMN `group_id` bigint unsigned DEFAULT NULL,
  ADD KEY `idx_balance_logs_store_id` (`store_id`),
  ADD KEY `idx_balance_logs_group_id` (`group_id`);
