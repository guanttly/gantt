CREATE TABLE IF NOT EXISTS employee_app_roles (
    id              VARCHAR(64) PRIMARY KEY,
    employee_id     VARCHAR(64) NOT NULL,
    org_node_id     VARCHAR(64) NOT NULL,
    app_role        VARCHAR(64) NOT NULL,
    source          VARCHAR(16) NOT NULL DEFAULT 'manual',
    source_group_id VARCHAR(64) DEFAULT NULL,
    granted_by      VARCHAR(64) NOT NULL,
    granted_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME DEFAULT NULL,

    UNIQUE KEY uk_emp_node_role (employee_id, org_node_id, app_role),
    INDEX idx_employee (employee_id),
    INDEX idx_node (org_node_id),
    INDEX idx_expires (expires_at),
    INDEX idx_source_group (source_group_id)
);

CREATE TABLE IF NOT EXISTS group_default_app_roles (
    id          VARCHAR(64) PRIMARY KEY,
    group_id    VARCHAR(64) NOT NULL,
    org_node_id VARCHAR(64) NOT NULL,
    app_role    VARCHAR(64) NOT NULL,
    created_by  VARCHAR(64) NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_group_node_role (group_id, org_node_id, app_role),
    INDEX idx_group (group_id)
);