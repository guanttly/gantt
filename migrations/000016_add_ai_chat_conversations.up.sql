CREATE TABLE IF NOT EXISTS ai_conversations (
    id                   VARCHAR(64)  NOT NULL PRIMARY KEY,
    org_node_id          VARCHAR(64)  NOT NULL,
    user_id              VARCHAR(64)  NOT NULL,
    title                VARCHAR(128) NOT NULL,
    message_count        INT          NOT NULL DEFAULT 0,
    last_message_at      DATETIME     NULL,
    last_message_preview VARCHAR(255) NOT NULL DEFAULT '',
    created_at           DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_ai_conversation_user_time (org_node_id, user_id, updated_at),
    KEY idx_ai_conversation_last_message_at (last_message_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 助手会话';

CREATE TABLE IF NOT EXISTS ai_conversation_messages (
    id              VARCHAR(64) NOT NULL PRIMARY KEY,
    conversation_id VARCHAR(64) NOT NULL,
    org_node_id     VARCHAR(64) NOT NULL,
    user_id         VARCHAR(64) NOT NULL,
    role            VARCHAR(16) NOT NULL,
    content         LONGTEXT    NOT NULL,
    created_at      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_ai_conversation_message_time (conversation_id, created_at),
    KEY idx_ai_conversation_message_owner (org_node_id, user_id),
    CONSTRAINT fk_ai_conversation_messages_conversation
        FOREIGN KEY (conversation_id) REFERENCES ai_conversations(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 助手会话消息';