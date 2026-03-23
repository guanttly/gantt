package manager

import (
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp/model"
)

// NewDefaultMCPServerManager 创建默认配置的MCP服务器管理器
func NewDefaultMCPServerManager(
	namingClient naming_client.INamingClient,
	groupName string,
	logger logging.ILogger,
) MCPServerManager {
	return NewNacosMCPServerManager(
		namingClient,
		groupName,
		logger,
		WithHealthCheckTimeout(10*time.Second),
		WithHTTPClientTimeout(2*time.Minute), // 默认2分钟超时
		WithClientInfo(model.ClientInfo{
			Name:    "agent-service",
			Version: "1.0.0",
		}),
	)
}

// NewFastRefreshMCPServerManager 创建快速刷新的MCP服务器管理器（用于开发/测试）
func NewFastRefreshMCPServerManager(
	namingClient naming_client.INamingClient,
	groupName string,
	logger logging.ILogger,
) MCPServerManager {
	return NewNacosMCPServerManager(
		namingClient,
		groupName,
		logger,
		WithHealthCheckTimeout(5*time.Second),
		WithHTTPClientTimeout(5*time.Minute), // 开发环境更长的超时时间
		WithClientInfo(model.ClientInfo{
			Name:    "agent-service-dev",
			Version: "1.0.0",
		}),
	)
}

// NewLongTimeoutMCPServerManager 创建支持长时间工具调用的MCP服务器管理器
func NewLongTimeoutMCPServerManager(
	namingClient naming_client.INamingClient,
	groupName string,
	logger logging.ILogger,
) MCPServerManager {
	return NewNacosMCPServerManager(
		namingClient,
		groupName,
		logger,
		WithHealthCheckTimeout(10*time.Second),
		WithHTTPClientTimeout(10*time.Minute), // 10分钟超时用于长时间运行的工具
		WithClientInfo(model.ClientInfo{
			Name:    "agent-service-long-timeout",
			Version: "1.0.0",
		}),
	)
}
