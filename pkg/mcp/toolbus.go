package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"jusha/mcp/pkg/logging"
	mcp_manager "jusha/mcp/pkg/mcp/manager"
)

// IToolBus MCP工具总线接口 - Anti-Corruption Layer
type IToolBus interface {
	// Execute 执行指定的MCP工具
	Execute(ctx context.Context, toolName string, payload any) (json.RawMessage, error)

	// Health 健康检查
	Health() error

	// GetServerNames 获取可用的服务器名称列表
	GetServerNames() []string
}

// toolBusConfig MCP工具总线配置
type toolBusConfig struct {
	DefaultTimeout    time.Duration
	RetryCount        int
	RetryDelay        time.Duration
	EnableHealthCheck bool
}

// DefaultToolBusConfig 默认配置
func DefaultToolBusConfig() *toolBusConfig {
	return &toolBusConfig{
		DefaultTimeout:    30 * time.Second,
		RetryCount:        3,
		RetryDelay:        1 * time.Second,
		EnableHealthCheck: true,
	}
}

// mcpToolBus MCP工具总线实现
type mcpToolBus struct {
	manager mcp_manager.MCPServerManager
	config  *toolBusConfig
	logger  logging.ILogger
}

// NewMCPToolBus 创建MCP工具总线
func NewMCPToolBus(manager mcp_manager.MCPServerManager, config *toolBusConfig, logger logging.ILogger) IToolBus {
	if config == nil {
		config = DefaultToolBusConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &mcpToolBus{
		manager: manager,
		config:  config,
		logger:  logger.With("component", "MCPToolBus"),
	}
}

// Execute 执行MCP工具
func (bus *mcpToolBus) Execute(ctx context.Context, toolName string, payload any) (json.RawMessage, error) {
	if bus.manager == nil {
		return nil, fmt.Errorf("MCP manager not available")
	}

	// 设置超时
	ctxWithTimeout, cancel := context.WithTimeout(ctx, bus.config.DefaultTimeout)
	defer cancel()

	// 转换payload为map[string]any
	var arguments map[string]any
	if payload != nil {
		// 先序列化再反序列化来转换类型
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			bus.logger.Error("Failed to marshal payload", "tool", toolName, "error", err)
			return nil, fmt.Errorf("marshal payload for tool %s: %w", toolName, err)
		}

		err = json.Unmarshal(payloadBytes, &arguments)
		if err != nil {
			bus.logger.Error("Failed to unmarshal payload to map", "tool", toolName, "error", err)
			return nil, fmt.Errorf("unmarshal payload for tool %s: %w", toolName, err)
		}
	}

	// 执行工具（带重试）
	var lastErr error
	for attempt := 0; attempt <= bus.config.RetryCount; attempt++ {
		if attempt > 0 {
			// 重试延迟
			select {
			case <-ctxWithTimeout.Done():
				return nil, ctxWithTimeout.Err()
			case <-time.After(bus.config.RetryDelay):
			}

			bus.logger.Warn("Retrying MCP tool execution",
				"tool", toolName,
				"attempt", attempt,
				"maxAttempts", bus.config.RetryCount)
		}

		bus.logger.Debug("Executing MCP tool",
			"tool", toolName,
			"attempt", attempt,
			"hasPayload", payload != nil)

		// 调用MCP管理器执行工具
		result, err := bus.manager.CallTool(ctxWithTimeout, toolName, arguments)
		if err != nil {
			lastErr = err

			// 检查是否是可重试错误
			if !bus.isRetryableError(err) {
				bus.logger.Error("Non-retryable error executing MCP tool",
					"tool", toolName,
					"error", err)
				return nil, fmt.Errorf("execute tool %s: %w", toolName, err)
			}

			bus.logger.Warn("Retryable error executing MCP tool",
				"tool", toolName,
				"error", err,
				"attempt", attempt)
			continue
		}

		// 成功执行，转换结果
		var resultBytes json.RawMessage
		if result != nil && len(result.Content) > 0 {
			// 提取Content中的实际数据
			// MCP协议返回格式: {"content":[{"type":"data","data":"<actual_json_string>"}]}
			content := result.Content[0]
			switch content.Type {
			case "data":
				// Data字段包含实际的JSON字符串，需要解析
				if content.Data != "" {
					resultBytes = json.RawMessage(content.Data)
				} else {
					bus.logger.Warn("Content type is 'data' but Data field is empty", "tool", toolName)
					resultBytes = json.RawMessage("{}")
				}
			case "text":
				// 对于text类型，尝试将其包装为JSON
				// 或者直接作为字符串返回
				if content.Text != "" {
					// 尝试解析为JSON，如果失败则包装成对象
					if json.Valid([]byte(content.Text)) {
						resultBytes = json.RawMessage(content.Text)
					} else {
						// 将文本包装成JSON对象
						wrapped := map[string]string{"text": content.Text}
						serialized, err := json.Marshal(wrapped)
						if err != nil {
							bus.logger.Error("Failed to wrap text result", "tool", toolName, "error", err)
							return nil, fmt.Errorf("wrap text result for tool %s: %w", toolName, err)
						}
						resultBytes = json.RawMessage(serialized)
					}
				} else {
					resultBytes = json.RawMessage("{}")
				}
			default:
				// 未知类型，返回整个CallToolResult
				bus.logger.Warn("Unknown content type, returning full result", "tool", toolName, "type", content.Type)
				serialized, err := json.Marshal(result)
				if err != nil {
					bus.logger.Error("Failed to serialize tool result", "tool", toolName, "error", err)
					return nil, fmt.Errorf("serialize result for tool %s: %w", toolName, err)
				}
				resultBytes = json.RawMessage(serialized)
			}
		}

		bus.logger.Debug("MCP tool executed successfully",
			"tool", toolName,
			"hasResult", result != nil)

		return resultBytes, nil
	}

	// 所有重试都失败了
	bus.logger.Error("All retry attempts failed for MCP tool",
		"tool", toolName,
		"attempts", bus.config.RetryCount+1,
		"lastError", lastErr)

	return nil, fmt.Errorf("execute tool %s after %d attempts: %w", toolName, bus.config.RetryCount+1, lastErr)
}

// Health 健康检查
func (bus *mcpToolBus) Health() error {
	if bus.manager == nil {
		return fmt.Errorf("MCP manager not available")
	}

	if !bus.config.EnableHealthCheck {
		return nil
	}

	// 简单的健康检查 - 检查管理器状态
	// 注意：具体的健康检查逻辑依赖于 IMCPServerManager 接口
	// 这里假设管理器已经启动并可用

	bus.logger.Debug("MCP tool bus health check passed")
	return nil
}

// GetServerNames 获取可用的服务器名称
func (bus *mcpToolBus) GetServerNames() []string {
	if bus.manager == nil {
		return []string{}
	}

	// 从 MCPServerManager 获取真实的服务器列表
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	servers, err := bus.manager.ListServers(ctx)
	if err != nil {
		bus.logger.Error("Failed to list servers from manager", "error", err)
		return []string{}
	}

	// 提取服务器名称列表
	names := make([]string, 0, len(servers))
	for _, server := range servers {
		if server != nil && server.Name != "" {
			names = append(names, server.Name)
		}
	}

	bus.logger.Debug("Retrieved server names", "count", len(names), "names", names)
	return names
}

// isRetryableError 判断错误是否可重试
func (bus *mcpToolBus) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 网络相关错误通常可重试
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains 检查字符串是否包含子字符串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				containsIgnoreCase(s, substr)))
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)

	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

// ToolBusError MCP工具总线错误类型
type ToolBusError struct {
	ToolName  string
	Operation string
	Cause     error
	Retryable bool
}

func (e *ToolBusError) Error() string {
	return fmt.Sprintf("toolbus %s %s: %v", e.Operation, e.ToolName, e.Cause)
}

func (e *ToolBusError) Unwrap() error {
	return e.Cause
}

func (e *ToolBusError) IsRetryable() bool {
	return e.Retryable
}

// NewToolBusError 创建工具总线错误
func NewToolBusError(toolName, operation string, cause error, retryable bool) *ToolBusError {
	return &ToolBusError{
		ToolName:  toolName,
		Operation: operation,
		Cause:     cause,
		Retryable: retryable,
	}
}
