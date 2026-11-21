-- 001_initial_schema.sql
-- Smart Space Management — 初始数据库 Schema
-- 由 AutoMigrate 结构生成，用于版本化迁移

CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `open_id` VARCHAR(128) NOT NULL,
    `union_id` VARCHAR(64) DEFAULT '',
    `nickname` VARCHAR(64) DEFAULT '',
    `avatar` VARCHAR(512) DEFAULT '',
    `phone` VARCHAR(20) DEFAULT '',
    `password_hash` VARCHAR(256) DEFAULT '',
    `role` VARCHAR(20) NOT NULL DEFAULT 'user',
    `status` TINYINT NOT NULL DEFAULT 1,
    `store_id` BIGINT UNSIGNED DEFAULT 0,
    `last_login` BIGINT DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_users_open_id` (`open_id`),
    INDEX `idx_users_deleted_at` (`deleted_at`),
    INDEX `idx_users_union_id` (`union_id`),
    INDEX `idx_users_phone` (`phone`),
    INDEX `idx_users_store_id` (`store_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `stores` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `name` VARCHAR(128) NOT NULL,
    `logo` VARCHAR(512) DEFAULT '',
    `cover` VARCHAR(512) DEFAULT '',
    `phone` VARCHAR(20) DEFAULT '',
    `address` VARCHAR(512) NOT NULL,
    `province` VARCHAR(32) DEFAULT '',
    `city` VARCHAR(32) DEFAULT '',
    `district` VARCHAR(32) DEFAULT '',
    `latitude` DECIMAL(10,7) DEFAULT 0,
    `longitude` DECIMAL(10,7) DEFAULT 0,
    `status` TINYINT NOT NULL DEFAULT 1,
    `description` TEXT,
    `open_time` VARCHAR(8) DEFAULT '00:00',
    `close_time` VARCHAR(8) DEFAULT '24:00',
    `tags` VARCHAR(512) DEFAULT '',
    `invite_code` VARCHAR(32) DEFAULT '',
    `config_json` TEXT,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_stores_invite_code` (`invite_code`),
    INDEX `idx_stores_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `resource_types` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `name` VARCHAR(64) NOT NULL,
    `icon` VARCHAR(512) DEFAULT '',
    `description` TEXT,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `config_json` TEXT,
    `sort` INT DEFAULT 0,
    PRIMARY KEY (`id`),
    INDEX `idx_resource_types_store_id` (`store_id`),
    INDEX `idx_resource_types_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `resources` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `resource_type_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(128) NOT NULL,
    `image` VARCHAR(512) DEFAULT '',
    `status` TINYINT NOT NULL DEFAULT 0,
    `device_ids` VARCHAR(1024) DEFAULT '',
    `sort` INT DEFAULT 0,
    `remark` VARCHAR(512) DEFAULT '',
    PRIMARY KEY (`id`),
    INDEX `idx_resources_store_id` (`store_id`),
    INDEX `idx_resources_resource_type_id` (`resource_type_id`),
    INDEX `idx_resources_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `devices` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `device_sn` VARCHAR(64) NOT NULL,
    `name` VARCHAR(128) NOT NULL,
    `type` VARCHAR(32) NOT NULL,
    `status` TINYINT NOT NULL DEFAULT 0,
    `mqtt_topic` VARCHAR(256) DEFAULT '',
    `last_online_at` BIGINT DEFAULT NULL,
    `config_json` TEXT,
    PRIMARY KEY (`id`),
    INDEX `idx_devices_store_id` (`store_id`),
    INDEX `idx_devices_device_sn` (`device_sn`),
    INDEX `idx_devices_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `orders` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `order_no` VARCHAR(32) NOT NULL,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `resource_id` BIGINT UNSIGNED NOT NULL,
    `resource_name` VARCHAR(128) DEFAULT '',
    `start_time` BIGINT NOT NULL,
    `end_time` BIGINT NOT NULL,
    `duration` INT NOT NULL,
    `unit_price` INT NOT NULL,
    `total_amount` INT NOT NULL,
    `status` TINYINT NOT NULL DEFAULT 0,
    `paid_at` BIGINT DEFAULT NULL,
    `cancel_reason` VARCHAR(512) DEFAULT '',
    `refund_amount` INT DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_orders_order_no` (`order_no`),
    INDEX `idx_orders_user_id` (`user_id`),
    INDEX `idx_orders_store_id` (`store_id`),
    INDEX `idx_orders_resource_id` (`resource_id`),
    INDEX `idx_orders_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `price_rules` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `resource_type_id` BIGINT UNSIGNED NOT NULL,
    `name` VARCHAR(64) NOT NULL,
    `time_unit` INT NOT NULL DEFAULT 60,
    `base_price` INT NOT NULL,
    `peak_surcharge` TEXT,
    `package_discount` TEXT,
    `is_default` TINYINT(1) DEFAULT 0,
    `status` TINYINT DEFAULT 1,
    PRIMARY KEY (`id`),
    INDEX `idx_price_rules_store_id` (`store_id`),
    INDEX `idx_price_rules_resource_type_id` (`resource_type_id`),
    INDEX `idx_price_rules_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `booking_rules` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `resource_type_id` BIGINT UNSIGNED NOT NULL,
    `advance_minutes` INT DEFAULT 30,
    `max_duration` INT DEFAULT 240,
    `min_duration` INT DEFAULT 30,
    `cancel_before` INT DEFAULT 30,
    `max_advance_days` INT DEFAULT 7,
    PRIMARY KEY (`id`),
    INDEX `idx_booking_rules_store_id` (`store_id`),
    INDEX `idx_booking_rules_resource_type_id` (`resource_type_id`),
    INDEX `idx_booking_rules_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ui_configs` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `key` VARCHAR(64) NOT NULL,
    `value` TEXT,
    `remark` VARCHAR(255) DEFAULT '',
    PRIMARY KEY (`id`),
    INDEX `idx_ui_configs_store_id` (`store_id`),
    INDEX `idx_ui_configs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `config_histories` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `config_type` VARCHAR(32) NOT NULL,
    `config_id` BIGINT UNSIGNED NOT NULL,
    `store_id` BIGINT UNSIGNED DEFAULT 0,
    `old_value` TEXT,
    `new_value` TEXT,
    `operator_id` BIGINT UNSIGNED DEFAULT 0,
    `version` INT NOT NULL DEFAULT 1,
    PRIMARY KEY (`id`),
    INDEX `idx_config_histories_config_type` (`config_type`),
    INDEX `idx_config_histories_config_id` (`config_id`),
    INDEX `idx_config_histories_store_id` (`store_id`),
    INDEX `idx_config_histories_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `refund_requests` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `order_id` BIGINT UNSIGNED NOT NULL,
    `user_id` BIGINT UNSIGNED NOT NULL,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `amount` INT NOT NULL DEFAULT 0,
    `reason` VARCHAR(512) DEFAULT '',
    `status` TINYINT NOT NULL DEFAULT 0,
    `reviewed_by` BIGINT UNSIGNED DEFAULT 0,
    `review_comment` VARCHAR(512) DEFAULT '',
    `reviewed_at` BIGINT DEFAULT NULL,
    PRIMARY KEY (`id`),
    INDEX `idx_refund_requests_order_id` (`order_id`),
    INDEX `idx_refund_requests_user_id` (`user_id`),
    INDEX `idx_refund_requests_store_id` (`store_id`),
    INDEX `idx_refund_requests_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `coupons` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `store_id` BIGINT UNSIGNED NOT NULL,
    `code` VARCHAR(32) NOT NULL,
    `type` VARCHAR(16) NOT NULL DEFAULT 'discount',
    `value` INT NOT NULL DEFAULT 0,
    `min_amount` INT DEFAULT 0,
    `total_count` INT DEFAULT 0,
    `used_count` INT DEFAULT 0,
    `start_at` BIGINT DEFAULT NULL,
    `end_at` BIGINT DEFAULT NULL,
    `status` TINYINT DEFAULT 1,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_coupons_code` (`code`),
    INDEX `idx_coupons_store_id` (`store_id`),
    INDEX `idx_coupons_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `audit_logs` (
    `id` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(3) NOT NULL,
    `updated_at` DATETIME(3) NOT NULL,
    `deleted_at` DATETIME(3) DEFAULT NULL,
    `user_id` BIGINT UNSIGNED DEFAULT 0,
    `store_id` BIGINT UNSIGNED DEFAULT 0,
    `action` VARCHAR(255) NOT NULL DEFAULT '',
    `ip` VARCHAR(45) DEFAULT '',
    `user_agent` VARCHAR(512) DEFAULT '',
    PRIMARY KEY (`id`),
    INDEX `idx_audit_logs_user_id` (`user_id`),
    INDEX `idx_audit_logs_store_id` (`store_id`),
    INDEX `idx_audit_logs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
