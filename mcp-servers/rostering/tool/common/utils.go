package common

import (
	"strings"

	"jusha/mcp/pkg/errors"
	"jusha/mcp/pkg/mcp"
)

// Common helper functions shared across all tool submodules

// findKeyIgnoreCase 在 map 中查找键（大小写不敏感）
// 优先返回完全匹配的键，其次返回大小写不敏感匹配的键
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

// GetStringWithDefault extracts a string value from a map by key, or returns a default value if not found or empty
func GetStringWithDefault(m map[string]any, key, defaultValue string) string {
	if v := GetString(m, key); v != "" {
		return v
	}
	return defaultValue
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

// GetFloat extracts a float64 value from a map by key (case-insensitive)
func GetFloat(m map[string]any, key string) float64 {
	if actualKey, ok := findKeyIgnoreCase(m, key); ok {
		if v, ok := m[actualKey]; ok {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			}
		}
	}
	return 0
}

// GetStringArray extracts a string array from a map by key (case-insensitive)
func GetStringArray(m map[string]any, key string) []string {
	if actualKey, ok := findKeyIgnoreCase(m, key); ok {
		if v, ok := m[actualKey]; ok {
			if arr, ok := v.([]interface{}); ok {
				result := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						result = append(result, s)
					}
				}
				return result
			}
		}
	}
	return []string{}
}

// NewToolResultError creates a tool error result with appropriate error code
// This is a convenience function that wraps mcp.NewToolError with proper error codes
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
