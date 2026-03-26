-- M02: 组织树节点表
CREATE TABLE IF NOT EXISTS org_nodes (
    id              VARCHAR(64) PRIMARY KEY,
    parent_id       VARCHAR(64) DEFAULT NULL,
    node_type       VARCHAR(32) NOT NULL COMMENT 'organization / campus / department / custom',
    name            VARCHAR(128) NOT NULL,
    code            VARCHAR(64) NOT NULL,
    path            VARCHAR(512) NOT NULL COMMENT '物化路径: /org1/campusA/deptX',
    depth           INT NOT NULL DEFAULT 0,
    is_login_point  BOOLEAN NOT NULL DEFAULT FALSE,
    status          VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active / suspended',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (parent_id) REFERENCES org_nodes(id) ON DELETE RESTRICT,
    UNIQUE KEY uk_parent_code (parent_id, code),
    INDEX idx_path (path),
    INDEX idx_parent (parent_id),
    INDEX idx_type_status (node_type, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
