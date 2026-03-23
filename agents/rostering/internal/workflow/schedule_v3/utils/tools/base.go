package tools

import (
	"encoding/json"

	"jusha/mcp/pkg/ai/toolcalling"
	"jusha/mcp/pkg/logging"
)

// BaseRosteringTool 排班工具基类
type BaseRosteringTool struct {
	name        string
	description string
	logger      logging.ILogger
}

// Name 返回工具名称
func (t *BaseRosteringTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *BaseRosteringTool) Description() string {
	return t.description
}

// Logger 返回日志记录器
func (t *BaseRosteringTool) Logger() logging.ILogger {
	return t.logger
}

// NewTextResult 创建文本结果
func (t *BaseRosteringTool) NewTextResult(content string) *toolcalling.ToolResult {
	return toolcalling.NewTextResult(content)
}

// NewJSONResult 创建JSON结果
func (t *BaseRosteringTool) NewJSONResult(data interface{}) (*toolcalling.ToolResult, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return toolcalling.NewJSONResult(string(jsonBytes)), nil
}

// NewErrorResult 创建错误结果
func (t *BaseRosteringTool) NewErrorResult(err error) *toolcalling.ToolResult {
	return toolcalling.NewErrorResult(err)
}
