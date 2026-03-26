-- M03: 用户认证表
CREATE TABLE IF NOT EXISTS users (
    id              VARCHAR(64) PRIMARY KEY,
    username        VARCHAR(64) NOT NULL,
    email           VARCHAR(128) NOT NULL,
    phone           VARCHAR(20) DEFAULT NULL,
    password_hash   VARCHAR(256) NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active / disabled',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_username (username),
    UNIQUE KEY uk_email (email),
    INDEX idx_phone (phone)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- M03: 角色表
CREATE TABLE IF NOT EXISTS roles (
    id              VARCHAR(64) PRIMARY KEY,
    name            VARCHAR(64) NOT NULL,
    display_name    VARCHAR(64) NOT NULL,
    permissions     JSON NOT NULL,
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_role_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- M03: 用户-组织节点-角色关联表
CREATE TABLE IF NOT EXISTS user_node_roles (
    id              VARCHAR(64) PRIMARY KEY,
    user_id         VARCHAR(64) NOT NULL,
    org_node_id     VARCHAR(64) NOT NULL,
    role_id         VARCHAR(64) NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (org_node_id) REFERENCES org_nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT,
    UNIQUE KEY uk_user_node_role (user_id, org_node_id, role_id),
    INDEX idx_unr_user (user_id),
    INDEX idx_unr_node (org_node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 系统预置角色
INSERT IGNORE INTO roles (id, name, display_name, permissions, is_system) VALUES
    ('role-platform-admin', 'platform_admin', '平台管理员', '["*"]', TRUE),
    ('role-org-admin', 'org_admin', '机构管理员', '["org:*","employee:*","shift:*","rule:*","schedule:*","leave:*","ai:*"]', TRUE),
    ('role-dept-admin', 'dept_admin', '科室管理员', '["employee:*","shift:*","rule:*","schedule:*","leave:*"]', TRUE),
    ('role-scheduler', 'scheduler', '排班负责人', '["employee:read","shift:read","rule:read","schedule:*","leave:read"]', TRUE),
    ('role-employee', 'employee', '普通员工', '["schedule:read:self","leave:create:self","preference:*:self"]', TRUE);
