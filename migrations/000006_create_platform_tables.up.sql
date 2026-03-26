-- M09: 平台管理与运维表

-- 订阅表
CREATE TABLE IF NOT EXISTS subscriptions (
    id            VARCHAR(64) NOT NULL PRIMARY KEY,
    org_node_id   VARCHAR(64) NOT NULL,
    plan          VARCHAR(32) NOT NULL DEFAULT 'free',
    status        VARCHAR(16) NOT NULL DEFAULT 'active',
    max_employees INT         NOT NULL DEFAULT 20,
    max_ai_tokens INT         NOT NULL DEFAULT 10000,
    start_date    DATE        NOT NULL,
    end_date      DATE,
    created_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_sub_org (org_node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id            VARCHAR(64)  NOT NULL PRIMARY KEY,
    org_node_id   VARCHAR(64),
    user_id       VARCHAR(64)  NOT NULL,
    username      VARCHAR(64)  NOT NULL DEFAULT '',
    action        VARCHAR(64)  NOT NULL,
    resource_type VARCHAR(64)  NOT NULL,
    resource_id   VARCHAR(64),
    detail        JSON,
    ip            VARCHAR(45),
    user_agent    VARCHAR(256),
    status_code   INT          NOT NULL DEFAULT 0,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_audit_org_time (org_node_id, created_at),
    INDEX idx_audit_user (user_id, created_at),
    INDEX idx_audit_action (action, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 系统配置表
CREATE TABLE IF NOT EXISTS system_configs (
    id    VARCHAR(64)  NOT NULL PRIMARY KEY,
    `key` VARCHAR(128) NOT NULL,
    value TEXT,

    UNIQUE KEY uk_config_key (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
