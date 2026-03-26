ALTER TABLE platform_users
    DROP INDEX idx_bound_employee,
    DROP COLUMN bound_employee_id;

ALTER TABLE org_nodes
    DROP COLUMN contact_phone,
    DROP COLUMN contact_name;

ALTER TABLE platform_users RENAME TO users;