-- 应用配置、工作流配置与节点模型配置

CREATE TABLE IF NOT EXISTS app_configs (
    id         VARCHAR(64)  NOT NULL PRIMARY KEY,
    app_code   VARCHAR(64)  NOT NULL,
    `key`      VARCHAR(128) NOT NULL,
    value      TEXT,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_config_key (app_code, `key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='应用级配置';

CREATE TABLE IF NOT EXISTS app_workflow_configs (
    id           VARCHAR(64)  NOT NULL PRIMARY KEY,
    app_code     VARCHAR(64)  NOT NULL,
    workflow_key VARCHAR(96)  NOT NULL,
    name         VARCHAR(128) NOT NULL,
    version      VARCHAR(32)  NOT NULL DEFAULT '',
    description  TEXT,
    enabled      TINYINT(1)   NOT NULL DEFAULT 1,
    sort_order   INT          NOT NULL DEFAULT 0,
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_workflow (app_code, workflow_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='应用工作流配置';

CREATE TABLE IF NOT EXISTS ai_model_configs (
    id              VARCHAR(64)  NOT NULL PRIMARY KEY,
    app_code        VARCHAR(64)  NOT NULL,
    workflow_key    VARCHAR(96)  NOT NULL,
    node_key        VARCHAR(96)  NOT NULL,
    provider        VARCHAR(32)  NOT NULL DEFAULT '',
    model           VARCHAR(128) NOT NULL DEFAULT '',
    timeout_seconds INT          NOT NULL DEFAULT 60,
    temperature     DOUBLE,
    max_tokens      INT          NOT NULL DEFAULT 0,
    enabled         TINYINT(1)   NOT NULL DEFAULT 1,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_ai_model_node (app_code, workflow_key, node_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 工作流节点模型配置';
