package ws

import (
	"jusha/mcp/pkg/logging"
)

// NewDefaultHub 创建默认的 Hub 实现
// 提供基于内存的连接管理
func NewDefaultHub() IHub {
	return newHub()
}

// NewDefaultServer 创建默认的 WebSocket 服务器
// 内部自动创建默认的 Hub
// logger: 日志记录器
// opts: 可选配置（如 WithMessageHandler, WithUpgrader）
func NewDefaultServer(logger logging.ILogger, opts ...ServerOption) IWSServer {
	hub := newHub()
	return newServer(hub, logger, opts...)
}
