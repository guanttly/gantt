-- ============================================================
-- 班次类型独立表设计
-- 目的：将班次类型从硬编码字符串改为可配置的数据库表
-- ============================================================

-- 1. 班次类型主表
CREATE TABLE IF NOT EXISTS shift_types (
    id VARCHAR(36) PRIMARY KEY COMMENT '类型ID',
    org_id VARCHAR(36) NOT NULL COMMENT '组织ID',
    code VARCHAR(50) NOT NULL COMMENT '类型编码（唯一标识，如 regular, overtime, special）',
    name VARCHAR(100) NOT NULL COMMENT '类型名称（显示名称，如 常规班次、加班班次）',
    description TEXT COMMENT '类型描述',
    
    -- 排班优先级配置
    scheduling_priority INT NOT NULL DEFAULT 50 COMMENT '排班优先级（数字越小优先级越高，1-100）',
    workflow_phase VARCHAR(50) NOT NULL DEFAULT 'normal' COMMENT '工作流阶段（normal, special, research, fixed, fill）',
    
    -- 显示配置
    color VARCHAR(20) COMMENT '显示颜色（hex格式，如 #409EFF）',
    icon VARCHAR(50) COMMENT '图标名称',
    sort_order INT DEFAULT 0 COMMENT '显示排序',
    
    -- 业务配置
    is_ai_scheduling BOOLEAN DEFAULT TRUE COMMENT '是否需要AI排班',
    is_fixed_schedule BOOLEAN DEFAULT FALSE COMMENT '是否固定排班（每周固定人员）',
    is_overtime BOOLEAN DEFAULT FALSE COMMENT '是否算加班',
    requires_special_skill BOOLEAN DEFAULT FALSE COMMENT '是否需要特殊技能',
    
    -- 状态与审计
    is_active BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    is_system BOOLEAN DEFAULT FALSE COMMENT '是否系统内置类型（不可删除）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL COMMENT '软删除时间',
    
    UNIQUE KEY uk_org_code (org_id, code, deleted_at),
    INDEX idx_org_priority (org_id, scheduling_priority),
    INDEX idx_workflow_phase (workflow_phase),
    INDEX idx_active (is_active, deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='班次类型表';

-- 2. 班次与类型关联表（多对一关系）
ALTER TABLE shifts 
    ADD COLUMN shift_type_id VARCHAR(36) COMMENT '班次类型ID（外键关联 shift_types.id）',
    ADD INDEX idx_shift_type (shift_type_id);

-- 可选：如果需要保持向后兼容，暂时保留 type 字段
-- ALTER TABLE shifts ADD COLUMN type_legacy VARCHAR(50) COMMENT '旧版类型字段（兼容用）';

-- 3. 初始化系统内置班次类型
INSERT INTO shift_types (id, org_id, code, name, description, scheduling_priority, workflow_phase, color, is_system, is_ai_scheduling, is_overtime, requires_special_skill) VALUES
-- 固定班次（最高优先级）
('sys-type-fixed', 'system', 'fixed', '固定班次', '每周固定人员的班次，无需AI排班', 10, 'fixed', '#909399', TRUE, FALSE, FALSE, FALSE),

-- 特殊班次（高优先级）
('sys-type-special', 'system', 'special', '特殊班次', '有特殊技能要求的班次，优先排班', 30, 'special', '#E6A23C', TRUE, TRUE, FALSE, TRUE),
('sys-type-overtime', 'system', 'overtime', '加班班次', '节假日或额外工作时间，优先排班', 31, 'special', '#F56C6C', TRUE, TRUE, TRUE, FALSE),
('sys-type-standby', 'system', 'standby', '备班班次', '待命或应急班次，优先排班', 32, 'special', '#C71585', TRUE, TRUE, FALSE, TRUE),

-- 普通班次（中等优先级）
('sys-type-regular', 'system', 'regular', '常规班次', '日常工作班次', 50, 'normal', '#409EFF', TRUE, TRUE, FALSE, FALSE),
('sys-type-normal', 'system', 'normal', '普通班次', '标准工作班次', 50, 'normal', '#67C23A', TRUE, TRUE, FALSE, FALSE),

-- 科研班次（较低优先级）
('sys-type-research', 'system', 'research', '科研班次', '科研或学习时间', 70, 'research', '#20B2AA', TRUE, TRUE, FALSE, FALSE),

-- 填充班次（最低优先级）
('sys-type-fill', 'system', 'fill', '填充班次', '用于补充排班不足', 90, 'fill', '#C0C4CC', TRUE, FALSE, FALSE, FALSE),
('sys-type-leave', 'system', 'leave', '请假班次', '休假或请假', 91, 'fill', '#DCDFE6', TRUE, FALSE, FALSE, FALSE);

-- 4. 数据迁移脚本（将现有 shifts.type 迁移到 shift_type_id）
-- 步骤1：创建临时映射表
CREATE TEMPORARY TABLE type_mapping AS
SELECT DISTINCT s.org_id, s.type, st.id as type_id
FROM shifts s
LEFT JOIN shift_types st ON (
    st.code = s.type 
    AND (st.org_id = s.org_id OR st.org_id = 'system')
)
WHERE s.type IS NOT NULL AND s.type != '';

-- 步骤2：更新班次的 shift_type_id
UPDATE shifts s
INNER JOIN type_mapping tm ON s.org_id = tm.org_id AND s.type = tm.type
SET s.shift_type_id = tm.type_id
WHERE s.shift_type_id IS NULL;

-- 步骤3：处理未匹配的班次（使用默认的 regular 类型）
UPDATE shifts s
SET s.shift_type_id = (
    SELECT id FROM shift_types 
    WHERE code = 'regular' AND (org_id = s.org_id OR org_id = 'system')
    LIMIT 1
)
WHERE s.shift_type_id IS NULL AND (s.type IS NULL OR s.type = '');

-- 5. 添加外键约束（可选，在数据迁移完成后）
-- ALTER TABLE shifts 
--     ADD CONSTRAINT fk_shift_type 
--     FOREIGN KEY (shift_type_id) 
--     REFERENCES shift_types(id) 
--     ON DELETE RESTRICT;

-- 6. 创建视图：带类型信息的班次列表
CREATE OR REPLACE VIEW v_shifts_with_type AS
SELECT 
    s.*,
    st.code as type_code,
    st.name as type_name,
    st.scheduling_priority,
    st.workflow_phase,
    st.color as type_color,
    st.is_ai_scheduling,
    st.is_fixed_schedule,
    st.is_overtime,
    st.requires_special_skill
FROM shifts s
LEFT JOIN shift_types st ON s.shift_type_id = st.id
WHERE s.deleted_at IS NULL;

-- 7. 便捷查询函数
-- 获取某组织的所有班次类型（按优先级排序）
-- SELECT * FROM shift_types 
-- WHERE org_id IN ('your-org-id', 'system') 
-- AND is_active = TRUE 
-- AND deleted_at IS NULL
-- ORDER BY scheduling_priority ASC, sort_order ASC;

-- 获取需要AI排班的班次（按优先级排序）
-- SELECT s.*, st.scheduling_priority, st.workflow_phase
-- FROM shifts s
-- INNER JOIN shift_types st ON s.shift_type_id = st.id
-- WHERE s.org_id = 'your-org-id'
-- AND st.is_ai_scheduling = TRUE
-- AND st.is_active = TRUE
-- ORDER BY st.scheduling_priority ASC;

