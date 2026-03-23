package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/tool/common"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"
)

// createOrUpdateConversationTool 创建或更新对话记录工具
type createOrUpdateConversationTool struct {
	logger          logging.ILogger
	managementRepo  repository.IManagementRepository
}

func NewCreateOrUpdateConversationTool(logger logging.ILogger, managementRepo repository.IManagementRepository) mcp.ITool {
	return &createOrUpdateConversationTool{
		logger:         logger,
		managementRepo: managementRepo,
	}
}

func (t *createOrUpdateConversationTool) Name() string {
	return "management.conversation.create_or_update"
}

func (t *createOrUpdateConversationTool) Description() string {
	return "Create or update a conversation record in the management service"
}

func (t *createOrUpdateConversationTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"conversationId": map[string]any{
				"type":        "string",
				"description": "Context-server conversation ID",
			},
			"orgId": map[string]any{
				"type":        "string",
				"description": "Organization ID",
			},
			"userId": map[string]any{
				"type":        "string",
				"description": "User ID",
			},
			"workflowType": map[string]any{
				"type":        "string",
				"description": "Workflow type (e.g., schedule.create, schedule.adjust)",
			},
			"lastMessageAt": map[string]any{
				"type":        "string",
				"description": "Last message timestamp (RFC3339 format)",
			},
			"messageCount": map[string]any{
				"type":        "integer",
				"description": "Total message count",
			},
			"meta": map[string]any{
				"type":        "object",
				"description": "Metadata from context-server (optional)",
			},
		},
		"required": []string{"conversationId", "orgId", "userId", "workflowType", "lastMessageAt", "messageCount"},
	}
}

func (t *createOrUpdateConversationTool) Execute(ctx context.Context, input map[string]any) (*mcp.CallToolResult, error) {
	// 解析输入
	conversationID, _ := input["conversationId"].(string)
	orgID, _ := input["orgId"].(string)
	userID, _ := input["userId"].(string)
	workflowType, _ := input["workflowType"].(string)
	lastMessageAtStr, _ := input["lastMessageAt"].(string)
	messageCount, _ := input["messageCount"].(float64)

	if conversationID == "" || orgID == "" || userID == "" || workflowType == "" {
		return common.NewExecuteError("Missing required fields", fmt.Errorf("conversationId, orgId, userId, and workflowType are required"))
	}

	// 解析时间
	var lastMessageAt time.Time
	if lastMessageAtStr != "" {
		var err error
		lastMessageAt, err = time.Parse(time.RFC3339, lastMessageAtStr)
		if err != nil {
			t.logger.Warn("Failed to parse lastMessageAt, using current time", "error", err, "input", lastMessageAtStr)
			lastMessageAt = time.Now()
		}
	} else {
		lastMessageAt = time.Now()
	}

	// 解析 meta
	meta := make(map[string]any)
	if metaRaw, ok := input["meta"]; ok {
		if metaMap, ok := metaRaw.(map[string]any); ok {
			meta = metaMap
		}
	}

	// 添加日志，追踪meta信息传递
	t.logger.Info("MCP tool received meta",
		"conversationID", conversationID,
		"metaKeys", func() []string {
			keys := make([]string, 0, len(meta))
			for k := range meta {
				keys = append(keys, k)
			}
			return keys
		}(),
		"scheduleStartDate", meta["scheduleStartDate"],
		"scheduleEndDate", meta["scheduleEndDate"],
		"scheduleStatus", meta["scheduleStatus"],
		"scheduleId", meta["scheduleId"])

	// 构建请求体
	reqBody := map[string]any{
		"conversationId": conversationID,
		"orgId":          orgID,
		"userId":         userID,
		"workflowType":   workflowType,
		"lastMessageAt":  lastMessageAt.Format(time.RFC3339),
		"messageCount":   int(messageCount),
		"meta":           meta,
	}

	// 调用管理服务的 HTTP API
	// 注意：baseURL 已经是 "api/v1"，所以这里只需要 "/conversations/create-or-update"
	var resp map[string]any
	if err := t.managementRepo.DoRequestWithData(ctx, "POST", "/conversations/create-or-update", reqBody, &resp); err != nil {
		t.logger.Error("Failed to call management service API", "error", err)
		return common.NewExecuteError("Failed to create or update conversation", err)
	}

	response := map[string]any{
		"status":  "success",
		"message": "Conversation record created or updated successfully",
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(data))},
	}, nil
}

var _ mcp.ITool = (*createOrUpdateConversationTool)(nil)
