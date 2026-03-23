-- ============================================================
-- 班次固定人员配置表设计
-- 目的：支持在班次中配置固定人员（按周重复或指定日期）
-- ============================================================

-- 1. 创建班次固定人员配置表
CREATE TABLE IF NOT EXISTS shift_fixed_assignments (
    id VARCHAR(36) PRIMARY KEY COMMENT '配置ID',
    shift_id VARCHAR(36) NOT NULL COMMENT '关联的班次ID',
    staff_id VARCHAR(36) NOT NULL COMMENT '固定人员ID',
    
    -- 模式配置
    pattern_type ENUM('weekly', 'monthly', 'specific') NOT NULL COMMENT '模式类型: weekly=按周重复, monthly=按月重复, specific=指定日期',
    
    -- 按周重复配置（pattern_type='weekly'时使用）
    weekdays JSON COMMENT '周几上班，例如 [1,3,5] 表示周一、三、五',
    week_pattern ENUM('every', 'odd', 'even') DEFAULT 'every' COMMENT '周重复模式: every=每周, odd=奇数周, even=偶数周',
    
    -- 按月重复配置（pattern_type='monthly'时使用）
    monthdays JSON COMMENT '每月哪几天上班，例如 [1,15,30] 表示每月1号、15号、30号',
    
    -- 指定日期配置（pattern_type='specific'时使用）
    specific_dates JSON COMMENT '指定日期列表，例如 ["2025-01-01", "2025-01-05"]',
    
    -- 生效时间
    start_date DATE COMMENT '生效开始日期',
    end_date DATE COMMENT '生效结束日期（NULL表示永久生效）',
    
    -- 状态与审计
    is_active BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL COMMENT '软删除时间',
    
    -- 索引
    UNIQUE KEY uk_shift_staff (shift_id, staff_id, deleted_at) COMMENT '班次-人员唯一索引（支持软删除）',
    INDEX idx_shift (shift_id) COMMENT '班次ID索引',
    INDEX idx_staff (staff_id) COMMENT '人员ID索引',
    INDEX idx_active (is_active) COMMENT '状态索引',
    INDEX idx_pattern_type (pattern_type) COMMENT '模式类型索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='班次固定人员配置表';

-- 2. 示例数据
-- 示例1：张三每周一、三、五上白班（每周）
INSERT INTO shift_fixed_assignments (id, shift_id, staff_id, pattern_type, weekdays, week_pattern, start_date, is_active) VALUES
('example-1', 'shift-001', 'staff-zhang-san', 'weekly', JSON_ARRAY(1, 3, 5), 'every', '2025-01-01', TRUE);

-- 示例1b：王五奇数周的周二、四上白班
INSERT INTO shift_fixed_assignments (id, shift_id, staff_id, pattern_type, weekdays, week_pattern, start_date, is_active) VALUES
('example-1b', 'shift-001', 'staff-wang-wu', 'weekly', JSON_ARRAY(2, 4), 'odd', '2025-01-01', TRUE);

-- 示例1c：赵六偶数周的周二、四上白班
INSERT INTO shift_fixed_assignments (id, shift_id, staff_id, pattern_type, weekdays, week_pattern, start_date, is_active) VALUES
('example-1c', 'shift-001', 'staff-zhao-liu', 'weekly', JSON_ARRAY(2, 4), 'even', '2025-01-01', TRUE);

-- 示例2：李四每月1号、15号、30号上夜班
INSERT INTO shift_fixed_assignments (id, shift_id, staff_id, pattern_type, monthdays, start_date, is_active) VALUES
('example-2', 'shift-002', 'staff-li-si', 'monthly', JSON_ARRAY(1, 15, 30), '2025-01-01', TRUE);

-- 示例3：王五在指定日期上夜班
INSERT INTO shift_fixed_assignments (id, shift_id, staff_id, pattern_type, specific_dates, start_date, end_date, is_active) VALUES
('example-3', 'shift-003', 'staff-wang-wu', 'specific', JSON_ARRAY('2025-01-01', '2025-01-05', '2025-01-10'), '2025-01-01', '2025-01-31', TRUE);

-- 3. 查询示例

-- 查询某个班次的所有固定人员配置
-- SELECT * FROM shift_fixed_assignments 
-- WHERE shift_id = 'shift-001' AND is_active = TRUE AND deleted_at IS NULL;

-- 查询某个人员的所有固定班次
-- SELECT * FROM shift_fixed_assignments 
-- WHERE staff_id = 'staff-zhang-san' AND is_active = TRUE AND deleted_at IS NULL;

-- 查询指定日期范围内生效的固定配置
-- SELECT * FROM shift_fixed_assignments 
-- WHERE shift_id = 'shift-001' 
--   AND is_active = TRUE 
--   AND deleted_at IS NULL
--   AND (start_date IS NULL OR start_date <= '2025-01-31')
--   AND (end_date IS NULL OR end_date >= '2025-01-01');

-- 4. 清理示例数据（可选）
-- DELETE FROM shift_fixed_assignments WHERE id LIKE 'example-%';

