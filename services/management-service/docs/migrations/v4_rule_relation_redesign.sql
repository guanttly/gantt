-- ============================================================================
-- V4排班规则关联关系重新设计 - 数据库迁移脚本
-- 创建日期: 2026-02-12
-- 功能说明: 
--   1. 重新设计规则与班次的关联关系，区分主体(subject)和客体(object)
--   2. 重新设计规则的适用范围，支持全局、员工、分组
--   3. 保留旧表数据兼容性，新表提供更精确的语义
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 设计说明
-- ----------------------------------------------------------------------------
-- 规则语义结构: 主体(谁) + 动作(什么关系) + 客体(与谁) + 范围(适用于哪些人)
--
-- 示例1: "禁止审核自己写的报告"
--   - 规则类型: exclusive (排他)
--   - 主体班次: 写报告班(下午)
--   - 客体班次: 审核班(次日上午)
--   - 适用范围: 全局
--   - 语义: 同一人排了"写报告班"后，不能排"审核班"
--
-- 示例2: "CT审核上下午班可合并"
--   - 规则类型: combinable (可组合)
--   - 主体班次: CT上午审核
--   - 客体班次: CT下午审核
--   - 适用范围: 全局
--   - 语义: 同一人可以同时排"CT上午审核"和"CT下午审核"
--
-- 示例3: "王晨每周最多3次夜班"
--   - 规则类型: maxCount (最大次数)
--   - 目标班次: 夜班
--   - 适用范围: 员工-王晨
--   - 语义: 王晨每周最多排3次夜班
-- ----------------------------------------------------------------------------

-- ----------------------------------------------------------------------------
-- 1. 规则班次关系表（区分主体和客体）
-- 用于排他、可组合、必须同时等涉及多班次关系的规则
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `rule_shift_relations` (
    `id` VARCHAR(64) PRIMARY KEY COMMENT '主键ID',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `rule_id` VARCHAR(64) NOT NULL COMMENT '规则ID',
    `shift_id` VARCHAR(64) NOT NULL COMMENT '班次ID',
    `relation_role` VARCHAR(32) NOT NULL COMMENT '关系角色: subject(主体-触发规则的班次) / object(客体-被约束的班次) / target(目标-单一班次规则)',
    `seq_order` INT DEFAULT 0 COMMENT '同角色内的排序（用于有序关系）',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX `idx_rule_id` (`rule_id`),
    INDEX `idx_shift_id` (`shift_id`),
    INDEX `idx_org_rule` (`org_id`, `rule_id`),
    INDEX `idx_role` (`rule_id`, `relation_role`),
    UNIQUE INDEX `idx_unique_relation` (`rule_id`, `shift_id`, `relation_role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='规则班次关系表';

-- ----------------------------------------------------------------------------
-- 2. 规则适用范围表（员工/分组范围约束）
-- 用于限定规则对哪些员工/分组生效
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `rule_apply_scopes` (
    `id` VARCHAR(64) PRIMARY KEY COMMENT '主键ID',
    `org_id` VARCHAR(64) NOT NULL COMMENT '组织ID',
    `rule_id` VARCHAR(64) NOT NULL COMMENT '规则ID',
    `scope_type` VARCHAR(32) NOT NULL COMMENT '范围类型: all(全局) / employee(员工) / group(分组) / exclude_employee(排除员工) / exclude_group(排除分组)',
    `scope_id` VARCHAR(64) COMMENT '范围对象ID（当scope_type为employee/group时必填）',
    `scope_name` VARCHAR(128) COMMENT '范围对象名称（冗余存储，便于展示）',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX `idx_rule_id` (`rule_id`),
    INDEX `idx_org_rule` (`org_id`, `rule_id`),
    INDEX `idx_scope_type` (`rule_id`, `scope_type`),
    INDEX `idx_scope_id` (`scope_type`, `scope_id`),
    UNIQUE INDEX `idx_unique_scope` (`rule_id`, `scope_type`, `scope_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='规则适用范围表';

-- ----------------------------------------------------------------------------
-- 3. 更新主规则表，添加关系类型说明字段
-- ----------------------------------------------------------------------------
ALTER TABLE `scheduling_rules`
ADD COLUMN IF NOT EXISTS `relation_type` VARCHAR(32) NULL COMMENT '班次关系类型: single(单班次)/binary(二元关系)/multi(多元关系)' AFTER `time_scope`,
ADD COLUMN IF NOT EXISTS `relation_desc` VARCHAR(256) NULL COMMENT '关系描述（如：A排他B、A可与B组合）' AFTER `relation_type`;

-- ----------------------------------------------------------------------------
-- 4. 数据迁移：从旧的 scheduling_rule_associations 迁移到新表
-- ----------------------------------------------------------------------------
-- 注意：这个迁移需要根据规则类型来判断角色
-- 对于 exclusive/combinable/required_together 类型，需要人工确认主体和客体
-- 对于其他类型，统一设置为 target

-- 迁移班次关联到新表（默认为 target 角色）
INSERT IGNORE INTO `rule_shift_relations` (`id`, `org_id`, `rule_id`, `shift_id`, `relation_role`, `created_at`)
SELECT 
    UUID(),
    sra.org_id,
    sra.rule_id,
    sra.association_id,
    CASE 
        WHEN sra.role = 'source' THEN 'subject'
        WHEN sra.role = 'target' THEN 'object'
        ELSE 'target'
    END,
    sra.created_at
FROM `scheduling_rule_associations` sra
WHERE sra.association_type = 'shift';

-- 迁移员工关联到新范围表
INSERT IGNORE INTO `rule_apply_scopes` (`id`, `org_id`, `rule_id`, `scope_type`, `scope_id`, `created_at`)
SELECT 
    UUID(),
    sra.org_id,
    sra.rule_id,
    'employee',
    sra.association_id,
    sra.created_at
FROM `scheduling_rule_associations` sra
WHERE sra.association_type = 'employee';

-- 迁移分组关联到新范围表
INSERT IGNORE INTO `rule_apply_scopes` (`id`, `org_id`, `rule_id`, `scope_type`, `scope_id`, `created_at`)
SELECT 
    UUID(),
    sra.org_id,
    sra.rule_id,
    'group',
    sra.association_id,
    sra.created_at
FROM `scheduling_rule_associations` sra
WHERE sra.association_type = 'group';

-- 为没有范围限定的规则添加默认全局范围
INSERT IGNORE INTO `rule_apply_scopes` (`id`, `org_id`, `rule_id`, `scope_type`, `created_at`)
SELECT 
    UUID(),
    sr.org_id,
    sr.id,
    'all',
    sr.created_at
FROM `scheduling_rules` sr
WHERE sr.apply_scope = 'global'
AND NOT EXISTS (
    SELECT 1 FROM `rule_apply_scopes` ras 
    WHERE ras.rule_id = sr.id
);

-- ============================================================================
-- 回滚脚本（如需回滚执行以下语句）
-- ============================================================================
-- DROP TABLE IF EXISTS `rule_apply_scopes`;
-- DROP TABLE IF EXISTS `rule_shift_relations`;
-- ALTER TABLE `scheduling_rules` DROP COLUMN IF EXISTS `relation_type`;
-- ALTER TABLE `scheduling_rules` DROP COLUMN IF EXISTS `relation_desc`;

-- ============================================================================
-- 规则类型与班次关系要求对照表
-- ============================================================================
-- | 规则类型         | relation_type | 主体(subject) | 客体(object) | 目标(target) |
-- |-----------------|---------------|--------------|-------------|-------------|
-- | exclusive       | binary        | 必填(1+)      | 必填(1+)     | -           |
-- | combinable      | binary        | 必填(1+)      | 必填(1+)     | -           |
-- | required_together| binary       | 必填(1+)      | 必填(1+)     | -           |
-- | periodic        | single        | -            | -           | 必填(1)      |
-- | maxCount        | single        | -            | -           | 必填(1+)     |
-- | forbidden_day   | single        | -            | -           | 可选(0+)     |
-- | preferred       | single        | -            | -           | 可选(0+)     |
-- ============================================================================
