package leave

import (
	"context"
	"encoding/json"

	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// getLeaveTool 获取请假详情工具
type getLeaveTool struct {
	logger   logging.ILogger
	provider service.IServiceProvider
}

func NewGetLeaveTool(logger logging.ILogger, provider service.IServiceProvider) mcp.ITool {
	return &getLeaveTool{logger: logger, provider: provider}
}

func (t *getLeaveTool) Name() string {
	return "rostering.leave.get"
}

func (t *getLeaveTool) Description() string {
	return "Get detailed information of a specific leave request"
}

func (t *getLeaveTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "Leave request ID",
			},
		},
		"required": []string{"id"},
	}
}

func (t *getLeaveTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	id := common.GetString(input, "id")
	if id == "" {
		return common.NewValidationError("Leave request ID is required")
	}

	leave, err := t.provider.Leave().Get(ctx, id)
	if err != nil {
		t.logger.Error("Failed to get leave", "id", id, "error", err)
		return common.NewExecuteError("Failed to get leave", err)
	}

	data, _ := json.MarshalIndent(leave, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*getLeaveTool)(nil)
