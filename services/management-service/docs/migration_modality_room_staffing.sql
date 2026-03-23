-- ============================================================================
-- 机房报告量与排班人数计算功能 - 数据库迁移脚本
-- 创建日期: 2025-11-28
-- 更新日期: 2025-11-28
-- 功能说明: 
--   1. 时间段管理 (time_periods)
--   2. 机房管理 (modality_rooms) - 放射科CT/MRI/DR等设备检查室
--   3. 检查类型管理 (scan_types) - 平扫/增强等检查类型
--   4. 机房周检查量预估 (modality_room_weekly_volumes) - 替代旧的 modality_room_volumes
--   5. 班次排班人数计算规则 (shift_staffing_rules)
--   6. 班次周默认人数配置 (shift_weekly_staff)
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. 时间段表 (time_periods)
-- 用于定义检查量统计的时间段，支持用户自定义配置
-- 例如：上午段(08:00-12:00)、下午段(14:00-18:00)、夜班段(20:00-次日08:00)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `time_periods` (
    `id` VARCHAR(64) NOT NULL COMMENT '时间段ID (UUID)',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `code` VARCHAR(64) NOT NULL COMMENT '时间段编码，如 morning/afternoon/night',
    `name` VARCHAR(128) NOT NULL COMMENT '时间段名称，如 上午段、下午段',
    `start_time` VARCHAR(8) NOT NULL COMMENT '开始时间 HH:MM，如 08:00',
    `end_time` VARCHAR(8) NOT NULL COMMENT '结束时间 HH:MM，如 12:00',
    `is_cross_day` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否跨日（如前日20:00到当日08:00）',
    `description` TEXT COMMENT '时间段说明',
    `is_active` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序序号',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_time_period_org_code` (`org_id`, `code`),
    INDEX `idx_time_period_org_id` (`org_id`),
    INDEX `idx_time_period_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='时间段配置表';

-- ----------------------------------------------------------------------------
-- 2. 机房表 (modality_rooms)
-- 机房指放射科CT/MRI/DR等大型设备的检查室
-- 命名遵循DICOM/RIS医学影像标准中的Modality概念
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `modality_rooms` (
    `id` VARCHAR(64) NOT NULL COMMENT '机房ID (UUID)',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `code` VARCHAR(64) NOT NULL COMMENT '机房编码，如 CT1/MRI2/DR3',
    `name` VARCHAR(128) NOT NULL COMMENT '机房名称，如 CT1号机房',
    `description` TEXT COMMENT '机房说明',
    `location` VARCHAR(256) DEFAULT NULL COMMENT '位置信息',
    `is_active` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序序号',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_modality_room_org_code` (`org_id`, `code`),
    INDEX `idx_modality_room_org_id` (`org_id`),
    INDEX `idx_modality_room_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='机房配置表（放射科CT/MRI/DR设备检查室）';

-- ----------------------------------------------------------------------------
-- 3. 检查类型表 (scan_types)
-- 放射科检查类型配置，如平扫、增强等
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `scan_types` (
    `id` VARCHAR(64) NOT NULL COMMENT '检查类型ID (UUID)',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `code` VARCHAR(64) NOT NULL COMMENT '类型编码，如 plain/enhanced',
    `name` VARCHAR(128) NOT NULL COMMENT '类型名称，如 平扫、增强',
    `description` TEXT COMMENT '类型说明',
    `is_active` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序序号',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_scan_type_org_code` (`org_id`, `code`),
    INDEX `idx_scan_type_org_id` (`org_id`),
    INDEX `idx_scan_type_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='检查类型配置表（放射科平扫/增强等）';

-- ----------------------------------------------------------------------------
-- 4. 机房周检查量预估表 (modality_room_weekly_volumes)
-- 记录每个机房按星期几、时间段、检查类型的预估检查量
-- 替代旧的按日期记录的 modality_room_volumes 表
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `modality_room_weekly_volumes` (
    `id` VARCHAR(64) NOT NULL COMMENT '记录ID (UUID)',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `modality_room_id` VARCHAR(64) NOT NULL COMMENT '机房ID',
    `weekday` INT NOT NULL COMMENT '周几：0=周日,1=周一,...,6=周六',
    `time_period_id` VARCHAR(64) NOT NULL COMMENT '时间段ID',
    `scan_type_id` VARCHAR(64) NOT NULL COMMENT '检查类型ID',
    `volume` INT NOT NULL DEFAULT 0 COMMENT '预估检查量',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_weekly_volume_unique` (`modality_room_id`, `weekday`, `time_period_id`, `scan_type_id`),
    INDEX `idx_weekly_volume_org_id` (`org_id`),
    INDEX `idx_weekly_volume_room_id` (`modality_room_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='机房周检查量预估表';

-- ----------------------------------------------------------------------------
-- 5. 班次排班人数计算规则表 (shift_staffing_rules)
-- 每个班次可配置一个计算规则，用于根据机房报告量自动计算默认人数
-- 与班次(shifts)一对一关系
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `shift_staffing_rules` (
    `id` VARCHAR(64) NOT NULL COMMENT '规则ID (UUID)',
    `shift_id` VARCHAR(64) NOT NULL COMMENT '班次ID（关联shifts表）',
    `modality_room_ids` TEXT NOT NULL COMMENT '关联的机房ID列表（JSON数组格式）',
    `time_period_id` VARCHAR(64) NOT NULL COMMENT '时间段ID',
    `avg_report_limit` INT NOT NULL DEFAULT 0 COMMENT '人均报告处理上限（0表示使用全局默认值）',
    `rounding_mode` VARCHAR(16) NOT NULL DEFAULT 'ceil' COMMENT '取整方式：ceil=向上取整，floor=向下取整',
    `is_active` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    `description` TEXT COMMENT '规则说明',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_staffing_rule_shift_id` (`shift_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='班次排班人数计算规则表';

-- ----------------------------------------------------------------------------
-- 6. 班次周默认人数表 (shift_weekly_staff)
-- 支持班次按周一到周日单独配置默认人数
-- 当某天有自定义配置时使用自定义值，否则使用班次的通用默认人数
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `shift_weekly_staff` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `shift_id` VARCHAR(64) NOT NULL COMMENT '班次ID（关联shifts表）',
    `weekday` INT NOT NULL COMMENT '周几：0=周日,1=周一,...,6=周六',
    `staff_count` INT NOT NULL DEFAULT 1 COMMENT '默认人数',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_shift_weekday` (`shift_id`, `weekday`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='班次周默认人数配置表';

-- ============================================================================
-- 初始化数据（可选）
-- ============================================================================

-- 示例：插入默认时间段配置
-- INSERT INTO `time_periods` (`id`, `org_id`, `code`, `name`, `start_time`, `end_time`, `is_cross_day`, `sort_order`) VALUES
-- (UUID(), 'your_org_id', 'morning', '上午段', '08:00', '12:00', 0, 1),
-- (UUID(), 'your_org_id', 'afternoon', '下午段', '14:00', '18:00', 0, 2),
-- (UUID(), 'your_org_id', 'night', '夜班段', '20:00', '08:00', 1, 3);

-- ============================================================================
-- 回滚脚本（如需回滚执行以下语句）
-- ============================================================================
-- DROP TABLE IF EXISTS `shift_weekly_staff`;
-- DROP TABLE IF EXISTS `shift_staffing_rules`;
-- DROP TABLE IF EXISTS `modality_room_weekly_volumes`;
-- DROP TABLE IF EXISTS `scan_types`;
-- DROP TABLE IF EXISTS `modality_rooms`;
-- DROP TABLE IF EXISTS `time_periods`;
