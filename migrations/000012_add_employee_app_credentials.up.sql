ALTER TABLE employees
    ADD COLUMN scheduling_role VARCHAR(16) NOT NULL DEFAULT 'employee' COMMENT 'DEPRECATED: 向后兼容字段，实际权限由 employee_app_roles 表管理',
    ADD COLUMN app_password_hash VARCHAR(256) DEFAULT NULL,
    ADD COLUMN app_must_reset_pwd BOOLEAN NOT NULL DEFAULT TRUE;