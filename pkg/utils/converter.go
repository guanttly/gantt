package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/model"
)

// ParseIntWithDefault 尝试将字符串转为int，失败则返回默认值
func ParseIntWithDefault(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def, err
	}
	return i, nil
}

// GetFileMediaType 根据文件名或路径返回文件的 MIME 类型（mediaType），找不到时返回 "application/octet-stream"
func GetFileMediaType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}
	mtype := mime.TypeByExtension(ext)
	if mtype == "" {
		return "application/octet-stream"
	}
	return mtype
}

func FormatCollectionName(collectionName string) string {
	// 去除-
	collectionName = strings.ReplaceAll(collectionName, "-", "")

	return collectionName
}

func FormatJsonStr(logger logging.ILogger, content string) (string, error) {
	// AI返回的时候是md格式的，去掉md格式【去除```json前所有的内容和```后所有的内容，如果内容不包含```json则直接返回error】
	startMarker := "```json"
	endMarker := "```"

	startIndex := strings.Index(content, startMarker)
	if startIndex == -1 {
		// 如果没有 ```json，尝试查找普通的 ```
		startMarker = "```"
		startIndex = strings.Index(content, startMarker)
		if startIndex == -1 {
			// 如果连 ``` 都没有，可能模型直接返回了 JSON 或无效内容
			// 尝试直接解析，或者认为格式无效
			trimmedContent := strings.TrimSpace(content)
			if (strings.HasPrefix(trimmedContent, "{") && strings.HasSuffix(trimmedContent, "}")) ||
				(strings.HasPrefix(trimmedContent, "[") && strings.HasSuffix(trimmedContent, "]")) {
				logger.WarnContext(context.Background(), "未找到Markdown JSON标记，但内容看起来像JSON，尝试直接使用", "content", content)
				return trimmedContent, nil
			}
			logger.ErrorContext(context.Background(), "响应内容中未找到 ```json 或 ``` 标记", "content", content)
			return "", fmt.Errorf("响应内容中未找到 ```json 或 ``` 标记")
		}
		// 找到了普通的 ```
		// 查找从 startIndex + len(startMarker) 开始的第一个 endMarker
		endIndex := strings.Index(content[startIndex+len(startMarker):], endMarker)
		if endIndex == -1 {
			logger.ErrorContext(context.Background(), "找到起始 ``` 但未找到结束 ``` 标记", "content", content)
			return "", fmt.Errorf("找到起始 ``` 但未找到结束 ``` 标记")
		}
		// 计算真实的结束索引
		endIndex += startIndex + len(startMarker)
		jsonContent := strings.TrimSpace(content[startIndex+len(startMarker) : endIndex])
		return jsonContent, nil

	} else {
		// 找到了 ```json
		// 查找从 startIndex + len(startMarker) 开始的第一个 endMarker
		endIndex := strings.Index(content[startIndex+len(startMarker):], endMarker)
		if endIndex == -1 {
			logger.ErrorContext(context.Background(), "找到起始 ```json 但未找到结束 ``` 标记", "content", content)
			return "", fmt.Errorf("找到起始 ```json 但未找到结束 ``` 标记")
		}
		// 计算真实的结束索引
		endIndex += startIndex + len(startMarker)
		jsonContent := strings.TrimSpace(content[startIndex+len(startMarker) : endIndex])
		return jsonContent, nil
	}
}

func FormatAIResponse(logger logging.ILogger, content string) (*model.AIThinkResponse, error) {
	var aiThinkResponse model.AIThinkResponse
	startMarker := "<think>"
	endMarker := "</think>"

	startIndex := strings.Index(content, startMarker)
	if startIndex == -1 {
		// 没有找到 <think> 标签，可能AI没有提供思考过程
		logger.WarnContext(context.Background(), "未在响应内容中找到 <think> 标签", "content", content)
		// 返回空的响应，但不认为是错误
		return &aiThinkResponse, nil
	}

	// 查找从 startIndex + len(startMarker) 开始的第一个 endMarker
	endIndex := strings.Index(content[startIndex+len(startMarker):], endMarker)
	if endIndex == -1 {
		logger.ErrorContext(context.Background(), "找到起始 <think> 但未找到结束 </think> 标记", "content", content)
		return nil, fmt.Errorf("找到起始 <think> 但未找到结束 </think> 标记")
	}

	// 计算真实的结束索引
	endIndex += startIndex + len(startMarker)

	// 提取 <think> 和 </think> 之间的内容
	thinkingContent := strings.TrimSpace(content[startIndex+len(startMarker) : endIndex])
	aiThinkResponse.Thinking = thinkingContent
	// 提取 </think> 之后的内容作为响应
	responseContent := strings.TrimSpace(content[endIndex+len(endMarker):])
	if responseContent == "" {
		logger.ErrorContext(context.Background(), "找到 <think> 但没有响应内容", "content", content)
		return nil, fmt.Errorf("找到 <think> 但没有响应内容")
	}
	aiThinkResponse.Response = responseContent
	return &aiThinkResponse, nil
}

func QueryParamInt(r *http.Request, key string, defaultValue int) int {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

// PropertiesToJSON 将 map[string]string 类型的属性序列化为 JSON 字符串
// 用于 Neo4j 存储，因为 Neo4j 不能直接存储复杂的 map 类型
func PropertiesToJSON(properties map[string]string) (string, error) {
	if properties == nil {
		return "{}", nil
	}

	propertiesBytes, err := json.Marshal(properties)
	if err != nil {
		return "", fmt.Errorf("failed to marshal properties: %w", err)
	}

	return string(propertiesBytes), nil
}

// PropertiesFromJSON 从 JSON 字符串反序列化为 map[string]string 类型的属性
// 用于从 Neo4j 读取数据时的反序列化
func PropertiesFromJSON(propertiesJSON string) (map[string]string, error) {
	var properties map[string]string

	if propertiesJSON == "" || propertiesJSON == "{}" {
		return make(map[string]string), nil
	}

	if err := json.Unmarshal([]byte(propertiesJSON), &properties); err != nil {
		log.Printf("Warning: failed to unmarshal properties: %v", err)
		return make(map[string]string), fmt.Errorf("failed to unmarshal properties: %w", err)
	}

	return properties, nil
}

// PropertiesFromJSONSafe 从 JSON 字符串安全地反序列化为 map[string]string 类型的属性
// 如果反序列化失败，返回空 map 而不是错误，用于容错处理
func PropertiesFromJSONSafe(propertiesJSON string) map[string]string {
	properties, err := PropertiesFromJSON(propertiesJSON)
	if err != nil {
		log.Printf("Warning: failed to unmarshal properties, returning empty map: %v", err)
		return make(map[string]string)
	}
	return properties
}
