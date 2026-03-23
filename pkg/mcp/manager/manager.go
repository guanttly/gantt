package manager

import (
	"context"

	"jusha/mcp/pkg/mcp/model"
)

// MCPServerInfo MCP服务器信息
type MCPServerInfo struct {
	Name     string            `json:"name"`
	Endpoint string            `json:"endpoint"`
	Tools    []model.Tool      `json:"tools"`
	Healthy  bool              `json:"healthy"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// MCPToolInfo MCP工具信息，包含其所属服务器
type MCPToolInfo struct {
	Tool       model.Tool        `json:"tool"`
	ServerName string            `json:"serverName"`
	Endpoint   string            `json:"endpoint"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// MCPServerManager 管理Nacos中的所有MCP服务器及其工具
type MCPServerManager interface {
	// 服务器管理
	// ListServers 获取所有已发现的MCP服务器列表
	ListServers(ctx context.Context) ([]*MCPServerInfo, error)

	// GetServer 根据名称获取特定的MCP服务器信息
	GetServer(ctx context.Context, serverName string) (*MCPServerInfo, error)

	// RefreshServers 手动刷新服务器列表（从Nacos重新发现）
	RefreshServers(ctx context.Context) error

	// WatchServers 监听MCP服务器变化，当有服务器上线/下线时回调
	WatchServers(ctx context.Context, callback func([]*MCPServerInfo)) error

	// 工具管理
	// ListAllTools 获取所有MCP服务器中的所有工具
	ListAllTools(ctx context.Context) ([]*MCPToolInfo, error)

	// ListServerTools 获取特定MCP服务器中的工具列表
	ListServerTools(ctx context.Context, serverName string) ([]*MCPToolInfo, error)

	// GetTool 根据工具名称获取工具信息（如果多个服务器有同名工具，返回第一个找到的）
	GetTool(ctx context.Context, toolName string) (*MCPToolInfo, error)

	// GetToolFromServer 从指定服务器获取特定工具
	GetToolFromServer(ctx context.Context, serverName, toolName string) (*MCPToolInfo, error)

	// FindToolsByName 查找所有匹配名称的工具（可能来自不同服务器）
	FindToolsByName(ctx context.Context, toolName string) ([]*MCPToolInfo, error)

	// 工具调用
	// CallTool 调用工具（自动选择最佳服务器）
	CallTool(ctx context.Context, toolName string, arguments map[string]any) (*model.CallToolResult, error)

	// CallToolOnServer 在指定服务器上调用工具
	CallToolOnServer(ctx context.Context, serverName, toolName string, arguments map[string]any) (*model.CallToolResult, error)

	// 生命周期管理
	// Start 启动管理器，开始服务发现和监听
	Start(ctx context.Context) error

	// Stop 停止管理器，清理资源
	Stop(ctx context.Context) error

	// IsRunning 检查管理器是否正在运行
	IsRunning() bool

	// 健康检查
	// HealthCheck 检查特定MCP服务器的健康状态
	HealthCheck(ctx context.Context, serverName string) (bool, error)

	// HealthCheckAll 检查所有MCP服务器的健康状态
	HealthCheckAll(ctx context.Context) (map[string]bool, error)

	// 统计信息
	// GetStats 获取管理器统计信息
	GetStats() (*MCPManagerStats, error)
}

// MCPManagerStats MCP管理器统计信息
type MCPManagerStats struct {
	TotalServers     int            `json:"totalServers"`
	HealthyServers   int            `json:"healthyServers"`
	UnhealthyServers int            `json:"unhealthyServers"`
	TotalTools       int            `json:"totalTools"`
	ToolsByServer    map[string]int `json:"toolsByServer"`
	LastRefreshTime  string         `json:"lastRefreshTime"`
	IsRunning        bool           `json:"isRunning"`
}
