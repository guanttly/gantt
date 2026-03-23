package client

import (
	"jusha/agent/sdk/rostering/domain"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

type rosteringClient struct {
	toolBus mcp.IToolBus
	logger  logging.ILogger
}

// NewClient 创建Rostering客户端
func NewClient(toolBus mcp.IToolBus, logger logging.ILogger) domain.IClient {
	return &rosteringClient{
		toolBus: toolBus,
		logger:  logger.With("component", "RosteringClient"),
	}
}
