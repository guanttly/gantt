package client

import (
	"jusha/agent/sdk/context/domain"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

type contextClient struct {
	toolBus mcp.IToolBus
	logger  logging.ILogger
}

// NewClient 创建 Context 客户端
func NewClient(toolBus mcp.IToolBus, logger logging.ILogger) domain.IContextClient {
	return &contextClient{
		toolBus: toolBus,
		logger:  logger.With("component", "ContextClient"),
	}
}
