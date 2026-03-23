package manager

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp/client"
	"jusha/mcp/pkg/mcp/manager/discovery"
	"jusha/mcp/pkg/mcp/model"
)

// NacosMCPServerManager 基于Nacos的MCP服务器管理器实现
type NacosMCPServerManager struct {
	namingClient naming_client.INamingClient
	groupName    string
	logger       logging.ILogger
	httpClient   *http.Client

	// 内部状态
	mu             sync.RWMutex
	runningLock    sync.RWMutex
	servers        map[string]*MCPServerInfo
	toolsCache     map[string][]*MCPToolInfo // serverName -> tools
	isRunning      bool
	lastUpdateTime time.Time

	// 服务发现 - 不再自己创建，而是接收外部传入的
	agentDiscovery discovery.AgentDiscovery
	stopChan       chan struct{}
	wg             sync.WaitGroup

	// 配置
	healthCheckTimeout time.Duration
	httpClientTimeout  time.Duration // HTTP客户端超时配置
	clientInfo         model.ClientInfo
}

// NewNacosMCPServerManager 创建新的Nacos MCP服务器管理器
func NewNacosMCPServerManager(
	namingClient naming_client.INamingClient,
	groupName string,
	logger logging.ILogger,
	options ...MCPManagerOption,
) *NacosMCPServerManager {
	// 内部创建AgentDiscovery
	agentDiscovery := discovery.NewNacosAgentDiscovery(namingClient, logger, groupName)

	manager := &NacosMCPServerManager{
		agentDiscovery:     agentDiscovery,
		logger:             logger.With("component", "NacosMCPServerManager"),
		httpClient:         &http.Client{Timeout: 30 * time.Second}, // httpClient仅用于健康检查，超时30s
		servers:            make(map[string]*MCPServerInfo),
		toolsCache:         make(map[string][]*MCPToolInfo),
		stopChan:           make(chan struct{}),
		healthCheckTimeout: 10 * time.Second,
		httpClientTimeout:  120 * time.Second, // 默认HTTP客户端超时2分钟
		clientInfo: model.ClientInfo{
			Name:    "agent-service",
			Version: "1.0.0",
		},
	}

	// 应用选项
	for _, opt := range options {
		opt(manager)
	}

	return manager
}

// MCPManagerOption 管理器配置选项
type MCPManagerOption func(*NacosMCPServerManager)

// WithHealthCheckTimeout 设置健康检查超时
func WithHealthCheckTimeout(timeout time.Duration) MCPManagerOption {
	return func(m *NacosMCPServerManager) {
		m.healthCheckTimeout = timeout
	}
}

// WithClientInfo 设置客户端信息
func WithClientInfo(clientInfo model.ClientInfo) MCPManagerOption {
	return func(m *NacosMCPServerManager) {
		m.clientInfo = clientInfo
	}
}

// WithHTTPClientTimeout 设置HTTP客户端超时
func WithHTTPClientTimeout(timeout time.Duration) MCPManagerOption {
	return func(m *NacosMCPServerManager) {
		m.httpClientTimeout = timeout
	}
}

// Start 启动管理器
func (m *NacosMCPServerManager) Start(ctx context.Context) error {
	m.runningLock.Lock()
	defer m.runningLock.Unlock()

	if m.isRunning {
		return fmt.Errorf("manager is already running")
	}

	// 启动内部的AgentDiscovery
	if err := m.agentDiscovery.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent discovery: %w", err)
	}
	// 初始化服务器列表（同步获取当前状态）
	if err := m.syncServersFromDiscovery(ctx); err != nil {
		m.logger.Warn("Failed to sync servers on start", "error", err)
	}

	// 启动服务变化监听
	m.isRunning = true
	m.stopChan = make(chan struct{})
	m.wg.Add(1)
	go m.watchServiceChanges(ctx)

	m.logger.Info("NacosMCPServerManager started")

	return nil
}

// Stop 停止管理器
func (m *NacosMCPServerManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	m.isRunning = false
	close(m.stopChan)

	// 停止内部的AgentDiscovery
	if err := m.agentDiscovery.Stop(ctx); err != nil {
		m.logger.Error("Failed to stop agent discovery", "error", err)
	} else {
		m.logger.Info("Agent discovery stopped within MCP manager")
	}

	// 等待后台任务结束
	m.wg.Wait()

	// 清理缓存
	m.servers = make(map[string]*MCPServerInfo)
	m.toolsCache = make(map[string][]*MCPToolInfo)

	m.logger.Info("NacosMCPServerManager stopped")
	return nil
}

// IsRunning 检查是否正在运行
func (m *NacosMCPServerManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// watchServiceChanges 监听服务变化
func (m *NacosMCPServerManager) watchServiceChanges(ctx context.Context) {
	defer m.wg.Done()

	// 监听服务变化
	err := m.agentDiscovery.WatchAgents(ctx, func(agents []*discovery.AgentService) {
		m.logger.Debug("Received service change notification", "agentCount", len(agents))
		if updateErr := m.updateServersFromAgents(ctx, agents); updateErr != nil {
			m.logger.Error("Failed to update servers from agents", "error", updateErr)
		}
	})

	if err != nil {
		m.logger.Error("Failed to watch agents", "error", err)
		return
	}

	// 保持监听直到停止信号
	<-m.stopChan
	m.logger.Debug("Service change watching stopped")
}

// syncServersFromDiscovery 同步获取当前服务状态
func (m *NacosMCPServerManager) syncServersFromDiscovery(ctx context.Context) error {
	agents, err := m.agentDiscovery.DiscoverAgents(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover agents: %w", err)
	}

	return m.updateServersFromAgents(ctx, agents)
}

// RefreshServers 手动刷新服务器列表（保持接口兼容性）
func (m *NacosMCPServerManager) RefreshServers(ctx context.Context) error {
	return m.syncServersFromDiscovery(ctx)
}

// updateServersFromAgents 从智能体列表更新服务器信息
func (m *NacosMCPServerManager) updateServersFromAgents(ctx context.Context, agents []*discovery.AgentService) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newServers := make(map[string]*MCPServerInfo)
	newToolsCache := make(map[string][]*MCPToolInfo)

	for _, agent := range agents {
		serverInfo := &MCPServerInfo{
			Name:     agent.Name,
			Endpoint: agent.Endpoint,
			Healthy:  agent.Healthy,
			Metadata: make(map[string]string),
			Tools:    []model.Tool{}, // 初始化为空数组而不是nil
		}

		// 只对健康的服务器获取工具列表
		if agent.Healthy {
			m.logger.Debug("Fetching tools for healthy server", "server", agent.Name, "endpoint", agent.Endpoint)
			tools, err := m.fetchServerTools(ctx, agent)
			if err != nil {
				m.logger.Warn("Failed to fetch tools for server",
					"server", agent.Name, "error", err)
				serverInfo.Healthy = false
				// 即使失败也要确保toolsCache中有条目，避免空指针
				newToolsCache[agent.Name] = []*MCPToolInfo{}
			} else {
				m.logger.Debug("Successfully fetched tools", "server", agent.Name, "toolCount", len(tools))
				serverInfo.Tools = make([]model.Tool, len(tools))
				toolInfos := make([]*MCPToolInfo, len(tools))

				for i, tool := range tools {
					serverInfo.Tools[i] = tool
					toolInfos[i] = &MCPToolInfo{
						Tool:       tool,
						ServerName: agent.Name,
						Endpoint:   agent.Endpoint,
						Metadata:   make(map[string]string),
					}
				}

				newToolsCache[agent.Name] = toolInfos
			}
		} else {
			m.logger.Debug("Skipping unhealthy server", "server", agent.Name)
			// 对于不健康的服务器，确保toolsCache中有空条目
			newToolsCache[agent.Name] = []*MCPToolInfo{}
		}

		newServers[agent.Name] = serverInfo
	}

	// 更新缓存
	m.servers = newServers
	m.toolsCache = newToolsCache
	m.lastUpdateTime = time.Now()

	// 记录详细的更新信息
	healthyCount := m.countHealthyServersLocked()
	totalToolsCount := 0
	for serverName, tools := range m.toolsCache {
		toolCount := len(tools)
		totalToolsCount += toolCount
		m.logger.Debug("Server tools cached", "server", serverName, "toolCount", toolCount)
	}

	m.logger.Debug("Servers updated",
		"total", len(newServers),
		"healthy", healthyCount,
		"totalTools", totalToolsCount)

	return nil
}

// fetchServerTools 获取服务器的工具列表
func (m *NacosMCPServerManager) fetchServerTools(ctx context.Context, agent *discovery.AgentService) ([]model.Tool, error) {
	endpoint := agent.Endpoint + "/mcp"
	c := client.NewHTTPMCPClientWithTimeout(endpoint, m.logger, m.httpClientTimeout)

	// 初始化客户端
	if err := c.Initialize(ctx, m.clientInfo); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}
	defer c.Close()

	// 获取工具列表
	tools, err := c.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return tools, nil
}

// ListServers 获取所有服务器列表
func (m *NacosMCPServerManager) ListServers(ctx context.Context) ([]*MCPServerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]*MCPServerInfo, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}

	return servers, nil
}

// GetServer 获取特定服务器信息
func (m *NacosMCPServerManager) GetServer(ctx context.Context, serverName string) (*MCPServerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, exists := m.servers[serverName]
	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	return server, nil
}

// WatchServers 监听服务器变化
func (m *NacosMCPServerManager) WatchServers(ctx context.Context, callback func([]*MCPServerInfo)) error {
	return m.agentDiscovery.WatchAgents(ctx, func(agents []*discovery.AgentService) {
		servers := make([]*MCPServerInfo, len(agents))
		for i, agent := range agents {
			servers[i] = &MCPServerInfo{
				Name:     agent.Name,
				Endpoint: agent.Endpoint,
				Healthy:  agent.Healthy,
				Metadata: make(map[string]string),
			}
		}
		callback(servers)
	})
}

// ListAllTools 获取所有工具
func (m *NacosMCPServerManager) ListAllTools(ctx context.Context) ([]*MCPToolInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 初始化为空数组，确保返回 [] 而不是 null
	allTools := make([]*MCPToolInfo, 0)
	for _, tools := range m.toolsCache {
		allTools = append(allTools, tools...)
	}

	return allTools, nil
}

// ListServerTools 获取特定服务器的工具
func (m *NacosMCPServerManager) ListServerTools(ctx context.Context, serverName string) ([]*MCPToolInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools, exists := m.toolsCache[serverName]
	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	return tools, nil
}

// GetTool 获取工具信息（返回第一个找到的）
func (m *NacosMCPServerManager) GetTool(ctx context.Context, toolName string) (*MCPToolInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tools := range m.toolsCache {
		for _, tool := range tools {
			if tool.Tool.Name == toolName {
				return tool, nil
			}
		}
	}

	return nil, fmt.Errorf("tool %s not found", toolName)
}

// GetToolFromServer 从指定服务器获取工具
func (m *NacosMCPServerManager) GetToolFromServer(ctx context.Context, serverName, toolName string) (*MCPToolInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools, exists := m.toolsCache[serverName]
	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	for _, tool := range tools {
		if tool.Tool.Name == toolName {
			return tool, nil
		}
	}

	return nil, fmt.Errorf("tool %s not found in server %s", toolName, serverName)
}

// FindToolsByName 查找所有匹配的工具
func (m *NacosMCPServerManager) FindToolsByName(ctx context.Context, toolName string) ([]*MCPToolInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 初始化为空数组，确保返回 [] 而不是 null
	matchedTools := make([]*MCPToolInfo, 0)
	for _, tools := range m.toolsCache {
		for _, tool := range tools {
			if tool.Tool.Name == toolName {
				matchedTools = append(matchedTools, tool)
			}
		}
	}

	return matchedTools, nil
}

// CallTool 调用工具（自动选择服务器）
func (m *NacosMCPServerManager) CallTool(ctx context.Context, toolName string, arguments map[string]any) (*model.CallToolResult, error) {
	toolInfo, err := m.GetTool(ctx, toolName)
	if err != nil {
		return nil, err
	}

	return m.CallToolOnServer(ctx, toolInfo.ServerName, toolName, arguments)
}

// CallToolOnServer 在指定服务器上调用工具
func (m *NacosMCPServerManager) CallToolOnServer(ctx context.Context, serverName, toolName string, arguments map[string]any) (*model.CallToolResult, error) {
	m.mu.RLock()
	server, exists := m.servers[serverName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	if !server.Healthy {
		return nil, fmt.Errorf("server %s is not healthy", serverName)
	}

	endpoint := server.Endpoint + "/mcp"
	c := client.NewHTTPMCPClientWithTimeout(endpoint, m.logger, m.httpClientTimeout)

	// 初始化客户端
	if err := c.Initialize(ctx, m.clientInfo); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}
	defer c.Close()

	// 调用工具
	result, err := c.CallTool(ctx, toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s on server %s: %w", toolName, serverName, err)
	}

	return result, nil
}

// HealthCheck 检查特定服务器健康状态
func (m *NacosMCPServerManager) HealthCheck(ctx context.Context, serverName string) (bool, error) {
	m.mu.RLock()
	server, exists := m.servers[serverName]
	m.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("server %s not found", serverName)
	}

	// 创建健康检查上下文
	healthCtx, cancel := context.WithTimeout(ctx, m.healthCheckTimeout)
	defer cancel()

	// 尝试调用健康检查端点
	healthURL := server.Endpoint + "/health"
	req, err := http.NewRequestWithContext(healthCtx, "GET", healthURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false, nil // 不返回错误，只是不健康
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == http.StatusOK

	// 更新服务器状态
	m.mu.Lock()
	if currentServer, exists := m.servers[serverName]; exists {
		currentServer.Healthy = healthy
	}
	m.mu.Unlock()

	return healthy, nil
}

// HealthCheckAll 检查所有服务器健康状态
func (m *NacosMCPServerManager) HealthCheckAll(ctx context.Context) (map[string]bool, error) {
	m.mu.RLock()
	serverNames := make([]string, 0, len(m.servers))
	for name := range m.servers {
		serverNames = append(serverNames, name)
	}
	m.mu.RUnlock()

	results := make(map[string]bool)
	for _, serverName := range serverNames {
		healthy, err := m.HealthCheck(ctx, serverName)
		if err != nil {
			m.logger.Warn("Health check failed", "server", serverName, "error", err)
			results[serverName] = false
		} else {
			results[serverName] = healthy
		}
	}

	return results, nil
}

// GetStats 获取统计信息
func (m *NacosMCPServerManager) GetStats() (*MCPManagerStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalServers := len(m.servers)
	healthyServers := m.countHealthyServersLocked()
	unhealthyServers := totalServers - healthyServers

	totalTools := 0
	toolsByServer := make(map[string]int)
	for serverName, tools := range m.toolsCache {
		count := len(tools)
		totalTools += count
		toolsByServer[serverName] = count
	}

	return &MCPManagerStats{
		TotalServers:     totalServers,
		HealthyServers:   healthyServers,
		UnhealthyServers: unhealthyServers,
		TotalTools:       totalTools,
		ToolsByServer:    toolsByServer,
		LastRefreshTime:  m.lastUpdateTime.Format(time.RFC3339),
		IsRunning:        m.isRunning,
	}, nil
}

// countHealthyServersLocked 计算健康服务器数量（需要持有读锁）
func (m *NacosMCPServerManager) countHealthyServersLocked() int {
	count := 0
	for _, server := range m.servers {
		if server.Healthy {
			count++
		}
	}
	return count
}
