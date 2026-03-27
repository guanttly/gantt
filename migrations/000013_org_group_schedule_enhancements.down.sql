-- 000013 rollback
ALTER TABLE employee_app_roles DROP COLUMN scope_group_id;

ALTER TABLE schedules DROP INDEX idx_schedules_group,
    DROP COLUMN group_id;
