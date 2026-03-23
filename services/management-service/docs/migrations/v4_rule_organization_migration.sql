-- ============================================================================
-- V4排班规则组织系统 - 数据库迁移脚本
-- 创建日期: 2026-02-11
-- 功能说明: 
--   1. 扩展规则表，添加分类字段
--   2. 扩展规则关联表，添加角色字段
--   3. 创建规则依赖关系表
--   4. 创建规则冲突关系表
--   5. 创建班次依赖关系表
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. 扩展规则表，添加分类字段
-- ----------------------------------------------------------------------------
ALTER TABLE `scheduling_rules` 
ADD COLUMN IF NOT EXISTS `category` VARCHAR(32) NULL COMMENT '规则分类: constraint/preference/dependency' AFTER `rule_data`,
ADD COLUMN IF NOT EXISTS `sub_category` VARCHAR(32) NULL COMMENT '规则子分类: forbid/must/limit/prefer/suggest/source/resource/order' AFTER `category`,
ADD COLUMN IF NOT EXISTS `original_rule_id` VARCHAR(64) NULL COMMENT '原始规则ID（如果是从语义化规则解析出来的）' AFTER `sub_category`,
ADD COLUMN IF NOT EXISTS `source_type` VARCHAR(32) NULL COMMENT '规则来源类型: manual/llm_parsed/migrated' AFTER `original_rule_id`,
ADD COLUMN IF NOT EXISTS `parse_confidence` DECIMAL(3,2) NULL COMMENT 'LLM 解析置信度 (0.0-1.0)' AFTER `source_type`,
ADD COLUMN IF NOT EXISTS `version` VARCHAR(8) NULL DEFAULT 'v4' COMMENT '规则版本号（V3=空或"v3", V4="v4"）' AFTER `parse_confidence`;

-- 添加索引
CREATE INDEX IF NOT EXISTS `idx_category` ON `scheduling_rules` (`category`);
CREATE INDEX IF NOT EXISTS `idx_sub_category` ON `scheduling_rules` (`sub_category`);
CREATE INDEX IF NOT EXISTS `idx_original_rule_id` ON `scheduling_rules` (`original_rule_id`);
CREATE INDEX IF NOT EXISTS `idx_source_type` ON `scheduling_rules` (`source_type`);
CREATE INDEX IF NOT EXISTS `idx_version` ON `scheduling_rules` (`version`);

-- ----------------------------------------------------------------------------
-- 2. 扩展规则关联表，添加角色字段
-- ----------------------------------------------------------------------------
ALTER TABLE `scheduling_rule_associations`
ADD COLUMN IF NOT EXISTS `role` VARCHAR(32) NOT NULL DEFAULT 'target' COMMENT '关联角色: target(约束目标)/source(数据来源)/reference(引用对象)' AFTER `association_id`;

-- ----------------------------------------------------------------------------
-- 3. 创建规则依赖关系表
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `rule_dependencies` (
    `id` VARCHAR(64) PRIMARY KEY COMMENT '主键ID',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `dependent_rule_id` VARCHAR(64) NOT NULL COMMENT '被依赖的规则ID（需要先执行）',
    `dependent_on_rule_id` VARCHAR(64) NOT NULL COMMENT '依赖的规则ID（后执行）',
    `dependency_type` VARCHAR(32) NOT NULL COMMENT '依赖类型: time/source/resource/order',
    `description` TEXT COMMENT '依赖关系描述',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_dependent_rule` (`dependent_rule_id`),
    INDEX `idx_dependent_on_rule` (`dependent_on_rule_id`),
    INDEX `idx_org` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='规则依赖关系表';

-- ----------------------------------------------------------------------------
-- 4. 创建规则冲突关系表
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `rule_conflicts` (
    `id` VARCHAR(64) PRIMARY KEY COMMENT '主键ID',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `rule_id_1` VARCHAR(64) NOT NULL COMMENT '冲突的规则1',
    `rule_id_2` VARCHAR(64) NOT NULL COMMENT '冲突的规则2',
    `conflict_type` VARCHAR(32) NOT NULL COMMENT '冲突类型: exclusive/resource/time/frequency',
    `description` TEXT COMMENT '冲突描述',
    `resolution_priority` INT COMMENT '解决优先级（数字越小越优先）',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_rule_1` (`rule_id_1`),
    INDEX `idx_rule_2` (`rule_id_2`),
    INDEX `idx_org` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='规则冲突关系表';

-- ----------------------------------------------------------------------------
-- 5. 创建班次依赖关系表
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `shift_dependencies` (
    `id` VARCHAR(64) PRIMARY KEY COMMENT '主键ID',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `dependent_shift_id` VARCHAR(64) NOT NULL COMMENT '被依赖的班次ID（需要先排）',
    `dependent_on_shift_id` VARCHAR(64) NOT NULL COMMENT '依赖的班次ID（后排）',
    `dependency_type` VARCHAR(32) NOT NULL COMMENT '依赖类型: time/source/resource',
    `rule_id` VARCHAR(64) COMMENT '产生此依赖关系的规则ID',
    `description` TEXT COMMENT '依赖关系描述',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_dependent_shift` (`dependent_shift_id`),
    INDEX `idx_dependent_on_shift` (`dependent_on_shift_id`),
    INDEX `idx_rule` (`rule_id`),
    INDEX `idx_org` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='班次依赖关系表';

-- ============================================================================
-- 回滚脚本（如需回滚执行以下语句）
-- ============================================================================
-- DROP TABLE IF EXISTS `shift_dependencies`;
-- DROP TABLE IF EXISTS `rule_conflicts`;
-- DROP TABLE IF EXISTS `rule_dependencies`;
-- ALTER TABLE `scheduling_rule_associations` DROP COLUMN IF EXISTS `role`;
-- ALTER TABLE `scheduling_rules` 
--     DROP INDEX IF EXISTS `idx_version`,
--     DROP INDEX IF EXISTS `idx_source_type`,
--     DROP INDEX IF EXISTS `idx_original_rule_id`,
--     DROP INDEX IF EXISTS `idx_sub_category`,
--     DROP INDEX IF EXISTS `idx_category`,
--     DROP COLUMN IF EXISTS `version`,
--     DROP COLUMN IF EXISTS `parse_confidence`,
--     DROP COLUMN IF EXISTS `source_type`,
--     DROP COLUMN IF EXISTS `original_rule_id`,
--     DROP COLUMN IF EXISTS `sub_category`,
--     DROP COLUMN IF EXISTS `category`;
