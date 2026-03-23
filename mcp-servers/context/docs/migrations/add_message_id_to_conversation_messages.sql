-- 添加 message_id 字段到 conversation_messages 表
-- 用于存储业务层消息唯一标识（来自 session.Message.ID），避免重复保存消息

-- 添加 message_id 列（允许 NULL，向后兼容旧数据）
-- 注意：如果列已存在，会报错，可以使用 IF NOT EXISTS（MySQL 8.0.19+）
ALTER TABLE `conversation_messages` 
ADD COLUMN IF NOT EXISTS `message_id` VARCHAR(64) NULL COMMENT '业务层消息唯一标识' AFTER `conversation_id`;

-- 创建唯一索引，确保同一会话中消息ID唯一
-- MySQL 8.0.13+ 支持函数索引（functional index），可以使用 WHERE 子句
-- MySQL 5.7 及以下版本不支持 WHERE 子句，需要创建普通的唯一索引
-- 注意：MySQL 5.7 的唯一索引允许多个 NULL 值，所以不会影响 NULL 消息
CREATE UNIQUE INDEX `idx_conversation_message_id` 
ON `conversation_messages` (`conversation_id`, `message_id`);
