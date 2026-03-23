// Package session 提供通用的会话管理
package session

import (
	"jusha/mcp/pkg/logging"
)

// NewDefaultStore 创建默认的 Store 实现
// 提供基于内存的会话存储，适用于单机部署或测试环境
// 生产环境建议实现基于 Redis/数据库的 Store
func NewDefaultStore() IStore {
	return newInMemoryStore()
}

// NewDefaultSessionService 创建默认的 SessionService 实现
// 内部自动创建默认的 InMemoryStore
// logger: 日志记录器
//
// 使用示例：
//
//	service := session.NewDefaultSessionService(logger)
//
// 如需注入意图识别器，使用 WithIntentRecognizer 方法：
//
//	service := session.NewDefaultSessionService(logger).
//	    WithIntentRecognizer(myRecognizer)
func NewDefaultSessionService(logger logging.ILogger) ISessionService {
	store := NewDefaultStore()
	return newSessionService(store, logger)
}
