ALTER TABLE users RENAME TO platform_users;

ALTER TABLE org_nodes
    ADD COLUMN contact_name VARCHAR(64) DEFAULT NULL AFTER code,
    ADD COLUMN contact_phone VARCHAR(20) DEFAULT NULL AFTER contact_name;

ALTER TABLE platform_users
    ADD COLUMN bound_employee_id VARCHAR(64) DEFAULT NULL AFTER must_reset_pwd,
    ADD INDEX idx_bound_employee (bound_employee_id);