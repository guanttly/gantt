DROP TABLE IF EXISTS shift_weekly_staff;
DROP TABLE IF EXISTS fixed_assignments;
DROP TABLE IF EXISTS shift_groups;

ALTER TABLE shifts
    DROP COLUMN metadata,
    DROP COLUMN description,
    DROP COLUMN type;