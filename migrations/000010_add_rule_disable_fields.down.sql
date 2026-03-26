DROP INDEX idx_rule_disabled ON rules;

ALTER TABLE rules
    DROP COLUMN disabled_reason,
    DROP COLUMN disabled_at,
    DROP COLUMN disabled_by,
    DROP COLUMN disabled;