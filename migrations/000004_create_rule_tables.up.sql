-- M05: 规则引擎表

CREATE TABLE IF NOT EXISTS rules (
    id               VARCHAR(64) PRIMARY KEY,
    org_node_id      VARCHAR(64) NOT NULL,
    name             VARCHAR(128) NOT NULL,
    category         VARCHAR(32) NOT NULL,              -- constraint / preference / dependency
    sub_type         VARCHAR(32) NOT NULL,              -- forbid / limit / must / prefer / combinable / source / order
    config           JSON NOT NULL,                     -- 规则参数（结构化 JSON）
    priority         INT NOT NULL DEFAULT 0,            -- 同类规则优先级
    is_enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    override_rule_id VARCHAR(64) DEFAULT NULL,          -- 覆盖的上级规则 ID（NULL=新增）
    description      VARCHAR(512) DEFAULT NULL,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_org_node (org_node_id),
    INDEX idx_category (org_node_id, category, sub_type),
    INDEX idx_override (override_rule_id)
);

CREATE TABLE IF NOT EXISTS rule_associations (
    id            VARCHAR(64) PRIMARY KEY,
    rule_id       VARCHAR(64) NOT NULL,
    org_node_id   VARCHAR(64) NOT NULL,
    target_type   VARCHAR(32) NOT NULL,                 -- shift / group / employee
    target_id     VARCHAR(64) NOT NULL,

    FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE,
    INDEX idx_org_node (org_node_id),
    INDEX idx_rule (rule_id),
    INDEX idx_target (target_type, target_id)
);
