package common

import (
	"strings"

	"jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/mcp"
)

// findKeyIgnoreCase 在 map 中查找键（大小写不敏感）
func findKeyIgnoreCase(m map[string]any, key string) (string, bool) {
	// 1. 先尝试完全匹配（性能优化）
	if _, ok := m[key]; ok {
		return key, true
	}

	// 2. 大小写不敏感匹配
	lowerKey := strings.ToLower(key)
	for k := range m {
		if strings.ToLower(k) == lowerKey {
			return k, true
		}
	}

	return "", false
}

// GetString extracts a string value from a map by key (case-insensitive)
func GetString(m map[string]any, key string) string {
	if actualKey, ok := findKeyIgnoreCase(m, key); ok {
		if v, ok := m[actualKey]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// GetInt extracts an integer value from a map by key (case-insensitive)
func GetInt(m map[string]any, key string) int {
	if actualKey, ok := findKeyIgnoreCase(m, key); ok {
		if v, ok := m[actualKey]; ok {
			switch val := v.(type) {
			case int:
				return val
			case float64:
				return int(val)
			}
		}
	}
	return 0
}

// GetIntWithDefault extracts an integer value from a map by key, or returns a default value if not found or zero
func GetIntWithDefault(m map[string]any, key string, defaultValue int) int {
	if v := GetInt(m, key); v > 0 {
		return v
	}
	return defaultValue
}

// NewExecuteError creates a tool error result with appropriate error code
func NewExecuteError(message string, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return nil, mcp.WrapToolError(errors.MCP_TOOL_EXEC_ERROR, message, err)
	}
	return nil, mcp.NewToolError(errors.MCP_TOOL_EXEC_ERROR, message)
}

// NewValidationError creates a validation error result
func NewValidationError(message string) (*mcp.CallToolResult, error) {
	return nil, mcp.NewToolError(errors.VALIDATION_ERROR, message)
}

// NewNotFoundError creates a not found error result
func NewNotFoundError(message string) (*mcp.CallToolResult, error) {
	return nil, mcp.NewToolError(errors.NOT_FOUND, message)
}
