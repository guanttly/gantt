// Package wsbridge 提供 workflow 和 ws 的集成桥接
package wsbridge

import (
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/session"
	"jusha/mcp/pkg/ws"
)

// NewDefaultBridge 创建默认的 Bridge 实现
// 内部自动创建默认的 Hub 和 SessionService
// logger: 日志记录器
func NewDefaultBridge(hub ws.IHub, sessionService session.ISessionService, logger logging.ILogger) IBridge {
	return newBridge(hub, sessionService, logger)
}
