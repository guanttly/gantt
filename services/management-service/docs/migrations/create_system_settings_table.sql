-- 创建系统设置表
-- 用于存储组织级别的系统配置

CREATE TABLE IF NOT EXISTS `system_settings` (
  `id` VARCHAR(64) NOT NULL PRIMARY KEY COMMENT '设置ID',
  `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
  `key` VARCHAR(128) NOT NULL COMMENT '设置键',
  `value` TEXT NOT NULL COMMENT '设置值',
  `description` VARCHAR(512) DEFAULT NULL COMMENT '设置描述',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  INDEX `idx_org_key` (`org_id`, `key`),
  UNIQUE KEY `uk_org_key` (`org_id`, `key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统设置表';

-- 插入默认配置（可选）
-- INSERT INTO `system_settings` (`id`, `org_id`, `key`, `value`, `description`) 
-- VALUES (UUID(), 'default-org', 'continuous_scheduling', 'true', '连续排班配置，默认开启');

