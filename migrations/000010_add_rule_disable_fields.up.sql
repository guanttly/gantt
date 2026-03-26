ALTER TABLE rules
    ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT FALSE AFTER is_enabled,
    ADD COLUMN disabled_by VARCHAR(64) NULL AFTER disabled,
    ADD COLUMN disabled_at DATETIME NULL AFTER disabled_by,
    ADD COLUMN disabled_reason TEXT NULL AFTER disabled_at;

CREATE INDEX idx_rule_disabled ON rules (org_node_id, disabled);