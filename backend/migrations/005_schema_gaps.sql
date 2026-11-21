-- 005_schema_gaps.sql
-- 目的：把「纯 migrations」路径补齐到模型（GORM）schema，收敛「迁移」与「AutoMigrate」两条路径。
-- 约定：以模型为准；不改动 001（老库不会重跑 001）。
-- 幂等：老库对应对象已存在时，迁移执行器会跳过「Duplicate column name / already exists」等错误。
--
-- 机器 diff 依据：临时库分别跑 migrations(001-004) 与 AutoMigrate 后比对 information_schema.columns。

-- ── ① 迁移历史缺失的表（此前仅由 AutoMigrate 创建）────────────────────────────
CREATE TABLE IF NOT EXISTS `payments` (
  `id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `out_trade_no` varchar(64) NOT NULL,
  `order_id` bigint unsigned NOT NULL,
  `order_no` varchar(32) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `store_id` bigint unsigned NOT NULL,
  `amount` bigint NOT NULL,
  `transaction_id` varchar(64) DEFAULT NULL,
  `status` tinyint NOT NULL DEFAULT 0,
  `pay_time` bigint DEFAULT NULL,
  `refund_amount` bigint DEFAULT 0,
  `refund_id` varchar(64) DEFAULT NULL,
  `callback_raw` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_payments_out_trade_no` (`out_trade_no`),
  KEY `idx_payments_deleted_at` (`deleted_at`),
  KEY `idx_payments_order_id` (`order_id`),
  KEY `idx_payments_order_no` (`order_no`),
  KEY `idx_payments_user_id` (`user_id`),
  KEY `idx_payments_store_id` (`store_id`),
  KEY `idx_payments_transaction_id` (`transaction_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `recharges` (
  `id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned NOT NULL,
  `store_id` bigint unsigned NOT NULL,
  `amount` bigint NOT NULL,
  `status` bigint NOT NULL DEFAULT 0,
  `note` varchar(255) DEFAULT NULL,
  `confirm_at` datetime(3) DEFAULT NULL,
  `confirm_by` bigint unsigned DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_recharges_deleted_at` (`deleted_at`),
  KEY `idx_recharges_user_id` (`user_id`),
  KEY `idx_recharges_store_id` (`store_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `balance_logs` (
  `id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned NOT NULL,
  `amount` bigint NOT NULL,
  `balance` bigint NOT NULL,
  `type` varchar(32) NOT NULL,
  `ref_id` varchar(64) DEFAULT NULL,
  `desc` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_balance_logs_deleted_at` (`deleted_at`),
  KEY `idx_balance_logs_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `product_templates` (
  `id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(128) NOT NULL,
  `store_id` bigint unsigned NOT NULL DEFAULT 0,
  `devices_json` text,
  `status` bigint NOT NULL DEFAULT 1,
  `protocol` varchar(16) DEFAULT 'json',
  PRIMARY KEY (`id`),
  KEY `idx_product_templates_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `merchant_stores` (
  `id` bigint unsigned NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `merchant_id` bigint unsigned NOT NULL,
  `store_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_merchant_stores_merchant_id` (`merchant_id`),
  KEY `idx_merchant_stores_store_id` (`store_id`),
  KEY `idx_merchant_stores_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── ② 共同表缺失的列（模型有、迁移无）────────────────────────────────────────
ALTER TABLE `users`  ADD COLUMN `balance` bigint NOT NULL DEFAULT 0;
ALTER TABLE `stores` ADD COLUMN `notify_webhook` varchar(512) DEFAULT NULL;
ALTER TABLE `devices` ADD COLUMN `protocol` varchar(16) DEFAULT 'json';

ALTER TABLE `audit_logs`
  ADD COLUMN `target_type` varchar(32) DEFAULT NULL,
  ADD COLUMN `target_id` bigint unsigned DEFAULT NULL,
  ADD COLUMN `old_value` text,
  ADD COLUMN `new_value` text,
  ADD COLUMN `remark` varchar(512) DEFAULT NULL;

ALTER TABLE `coupons`
  ADD COLUMN `resource_type_id` bigint unsigned NOT NULL DEFAULT 0,
  ADD COLUMN `name` varchar(128) DEFAULT NULL,
  ADD COLUMN `hours` bigint DEFAULT 0,
  ADD COLUMN `price` bigint DEFAULT 0,
  ADD COLUMN `discount` bigint DEFAULT 0,
  ADD COLUMN `used_at` datetime(3) DEFAULT NULL,
  ADD COLUMN `user_id` bigint unsigned DEFAULT 0;

-- ── ③ 类型对齐（模型为准；老库已是目标类型，MODIFY 为幂等空操作）────────────
-- 001 的 coupons 是旧设计（type varchar(16)/status tinyint），模型为 int → bigint
ALTER TABLE `coupons` MODIFY `type` bigint DEFAULT 0, MODIFY `status` bigint DEFAULT 0;
-- 001 的 audit_logs.action 为 varchar(255)，模型为 varchar(64)
ALTER TABLE `audit_logs` MODIFY `action` varchar(64) NOT NULL;
