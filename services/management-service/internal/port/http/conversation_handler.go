package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"jusha/gantt/service/management/domain/service"
	pkg_error "jusha/mcp/pkg/errors"
)

// ListScheduleConversations 查询排班对话记录
func (h *HTTPHandler) ListScheduleConversations(w http.ResponseWriter, r *http.Request) {
	if h.container.GetConversationService() == nil {
		RespondError(w, http.StatusServiceUnavailable, pkg_error.INFRASTRUCTURE_ERROR, "Conversation service is not available")
		return
	}

	// 解析查询参数
	filter := &service.ScheduleConversationFilter{
		OrgID:        r.URL.Query().Get("orgId"),
		UserID:       r.URL.Query().Get("userId"),
		StartDate:    r.URL.Query().Get("startDate"),
		EndDate:      r.URL.Query().Get("endDate"),
		WorkflowType: r.URL.Query().Get("workflowType"),
		Status:       r.URL.Query().Get("status"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := parseInt(limitStr, 20); err == nil {
			filter.Limit = limit
		}
	} else {
		filter.Limit = 20
	}

	// 调用服务
	conversations, err := h.container.GetConversationService().ListScheduleConversations(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list schedule conversations", "error", err)
		RespondError(w, http.StatusInternalServerError, pkg_error.INTERNAL, "Failed to list conversations")
		return
	}

	RespondSuccess(w, conversations)
}

// CreateOrUpdateConversation 创建或更新对话记录（由 MCP 工具调用）
func (h *HTTPHandler) CreateOrUpdateConversation(w http.ResponseWriter, r *http.Request) {
	if h.container.GetConversationService() == nil {
		RespondError(w, http.StatusServiceUnavailable, pkg_error.INFRASTRUCTURE_ERROR, "Conversation service is not available")
		return
	}

	// 解析请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		RespondError(w, http.StatusBadRequest, pkg_error.VALIDATION_ERROR, "Failed to read request body")
		return
	}

	var req service.CreateOrUpdateConversationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		RespondError(w, http.StatusBadRequest, pkg_error.VALIDATION_ERROR, "Invalid request body")
		return
	}

	// 验证必填字段
	if req.ConversationID == "" || req.OrgID == "" || req.UserID == "" || req.WorkflowType == "" {
		RespondError(w, http.StatusBadRequest, pkg_error.VALIDATION_ERROR, "Missing required fields: conversationId, orgId, userId, workflowType")
		return
	}

	// 解析时间
	if req.LastMessageAt.IsZero() {
		req.LastMessageAt = time.Now()
	}

	// 调用服务
	if err := h.container.GetConversationService().CreateOrUpdateConversation(r.Context(), &req); err != nil {
		h.logger.Error("Failed to create or update conversation", "error", err)
		RespondError(w, http.StatusInternalServerError, pkg_error.INTERNAL, "Failed to create or update conversation")
		return
	}

	RespondSuccess(w, map[string]string{
		"status":  "success",
		"message": "Conversation record created or updated successfully",
	})
}

// parseInt 解析整数，如果失败返回默认值
func parseInt(s string, defaultValue int) (int, error) {
	if s == "" {
		return defaultValue, nil
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue, err
	}
	return val, nil
}
