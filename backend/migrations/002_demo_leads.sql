-- 002_demo_leads.sql
-- 官网「预约演示」意向客户线索表
CREATE TABLE IF NOT EXISTS `demo_leads` (
  `id`         BIGINT UNSIGNED NOT NULL COMMENT '主键（Snowflake ShortID）',
  `name`       VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '联系人姓名',
  `phone`      VARCHAR(20)  NOT NULL DEFAULT '' COMMENT '手机号',
  `company`    VARCHAR(128) NOT NULL DEFAULT '' COMMENT '公司/门店名',
  `city`       VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '城市',
  `store_type` VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '门店类型（棋牌室/茶室/球馆…）',
  `source`     VARCHAR(32)  NOT NULL DEFAULT 'website' COMMENT '来源渠道',
  `remark`     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `ip`         VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '提交IP',
  `user_agent` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '浏览器UA',
  `status`     TINYINT      NOT NULL DEFAULT 0 COMMENT '0新线索 1已联系 2已成交 3无效',
  `created_at` DATETIME(3)  NULL DEFAULT NULL,
  `updated_at` DATETIME(3)  NULL DEFAULT NULL,
  `deleted_at` DATETIME(3)  NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_demo_leads_phone` (`phone`),
  KEY `idx_demo_leads_status` (`status`),
  KEY `idx_demo_leads_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='官网留资线索';
