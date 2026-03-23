package utils

import (
	"os"
	"strconv"
	"strings"
)

// --- Helper functions to override config from environment variables ---

// overrideFromEnv reads an environment variable and returns its value if set, otherwise returns the current value.
func OverrideFromEnv(key string, currentVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return currentVal
}

// overrideIntFromEnv reads an environment variable, parses it as an integer,
// and returns the parsed value if successful, otherwise returns the current value.
func OverrideIntFromEnv(key string, currentVal int) int {
	if valueStr, exists := os.LookupEnv(key); exists && valueStr != "" {
		if valueInt, err := strconv.Atoi(valueStr); err == nil {
			return valueInt
		}
		// Optionally log a warning here if parsing fails
	}
	return currentVal // 返回解析后的配置
}

// overrideBoolFromEnv reads an environment variable, parses it as a boolean,// and returns the parsed value if successful, otherwise returns the current value.
// Handles "true", "1", "false", "0". Case-insensitive.
func OverrideBoolFromEnv(key string, currentVal bool) bool {
	if valueStr, exists := os.LookupEnv(key); exists && valueStr != "" {
		lowerVal := strings.ToLower(valueStr)
		if lowerVal == "true" || lowerVal == "1" {
			return true
		}
		if lowerVal == "false" || lowerVal == "0" {
			return false
		}
		// Optionally log a warning here if parsing fails
	}
	return currentVal
}

// --- End Helper functions ---
