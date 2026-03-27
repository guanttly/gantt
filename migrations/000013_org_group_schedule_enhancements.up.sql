-- 000013: 组织架构-排班分组增强
-- 1. schedules 表增加 group_id，排班以分组为单位
ALTER TABLE schedules
    ADD COLUMN group_id VARCHAR(64) DEFAULT NULL COMMENT '排班分组ID，关联 employee_groups.id',
    ADD INDEX idx_schedules_group (group_id);

-- 2. employee_app_roles 表增加 scope_group_id，支持分组级权限约束
ALTER TABLE employee_app_roles
    ADD COLUMN scope_group_id VARCHAR(64) DEFAULT NULL
        COMMENT '限制权限生效范围到指定分组，NULL 表示科室全范围';
