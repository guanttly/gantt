package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// SetUserWorkflowVersionRequest 设置用户工作流版本请求
type SetUserWorkflowVersionRequest struct {
	OrgID   string `json:"orgId" validate:"required"`
	Version string `json:"version" validate:"required"`
}

// GetUserWorkflowVersion 获取用户工作流版本偏好
// GET /api/v1/users/{userId}/preferences/workflow-version
func (h *HTTPHandler) GetUserWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	if userID == "" {
		RespondBadRequest(w, "userId is required")
		return
	}

	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	// 使用 SystemSettingService 获取用户偏好
	// Key格式: user_{userID}_workflow_version
	key := fmt.Sprintf("user_%s_workflow_version", userID)
	value, err := h.container.GetSystemSettingService().GetSetting(r.Context(), orgID, key)
	if err != nil {
		// 如果获取失败或不存在，返回默认值 "v2"
		h.logger.Debug("User workflow version not found, using default v2", "orgID", orgID, "userID", userID, "error", err)
		RespondSuccess(w, map[string]string{
			"version": "v2",
		})
		return
	}

	// 验证版本值
	if value != "v2" && value != "v3" && value != "v4" {
		h.logger.Warn("Invalid workflow version, using default v2", "orgID", orgID, "userID", userID, "version", value)
		RespondSuccess(w, map[string]string{
			"version": "v2",
		})
		return
	}

	RespondSuccess(w, map[string]string{
		"version": value,
	})
}

// SetUserWorkflowVersion 设置用户工作流版本偏好
// PUT /api/v1/users/{userId}/preferences/workflow-version
func (h *HTTPHandler) SetUserWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userId"]
	if userID == "" {
		RespondBadRequest(w, "userId is required")
		return
	}

	var req SetUserWorkflowVersionRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", "error", err)
		RespondBadRequest(w, "Invalid request")
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to unmarshal request", "error", err)
		RespondBadRequest(w, "Invalid JSON")
		return
	}

	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	// 验证版本值
	if req.Version != "v2" && req.Version != "v3" && req.Version != "v4" {
		RespondBadRequest(w, "Invalid version, must be 'v2', 'v3' or 'v4'")
		return
	}

	// 使用 SystemSettingService 设置用户偏好
	// Key格式: user_{userID}_workflow_version
	key := fmt.Sprintf("user_%s_workflow_version", userID)
	description := fmt.Sprintf("User workflow version preference for user %s", userID)
	err = h.container.GetSystemSettingService().SetSetting(r.Context(), req.OrgID, key, req.Version, description)
	if err != nil {
		h.logger.Error("Failed to set user workflow version", "error", err, "orgID", req.OrgID, "userID", userID, "version", req.Version)
		RespondInternalError(w, "Failed to set user workflow version")
		return
	}

	RespondSuccess(w, map[string]string{
		"status": "ok",
	})
}
