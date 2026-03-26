-- AI 配额表
CREATE TABLE IF NOT EXISTS ai_quotas (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    org_node_id   BIGINT UNSIGNED NOT NULL COMMENT '租户组织节点 ID',
    provider      VARCHAR(32)     NOT NULL DEFAULT 'openai' COMMENT 'AI 供应商',
    monthly_limit INT             NOT NULL DEFAULT 100000 COMMENT '每月 token 限额',
    used_tokens   INT             NOT NULL DEFAULT 0 COMMENT '已使用 token',
    reset_at      DATETIME        NOT NULL COMMENT '下次重置时间',
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_ai_quotas_org_provider (org_node_id, provider),
    INDEX idx_ai_quotas_reset (reset_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 配额';

-- AI 使用记录表
CREATE TABLE IF NOT EXISTS ai_usage_logs (
    id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    org_node_id       BIGINT UNSIGNED NOT NULL COMMENT '租户组织节点 ID',
    user_id           VARCHAR(64)     NOT NULL COMMENT '操作用户 ID',
    provider          VARCHAR(32)     NOT NULL COMMENT 'AI 供应商',
    model             VARCHAR(64)     NOT NULL COMMENT '模型名称',
    prompt_tokens     INT             NOT NULL DEFAULT 0 COMMENT '输入 token',
    completion_tokens INT             NOT NULL DEFAULT 0 COMMENT '输出 token',
    purpose           VARCHAR(64)     NOT NULL COMMENT '用途 (chat/schedule/rule)',
    created_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_ai_usage_org (org_node_id),
    INDEX idx_ai_usage_user (user_id),
    INDEX idx_ai_usage_time (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 使用记录';
