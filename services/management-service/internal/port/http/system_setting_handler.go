package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// SetSystemSettingRequest 设置系统设置请求
type SetSystemSettingRequest struct {
	Value       string `json:"value" validate:"required"`
	Description string `json:"description"`
}

// SystemSettingResponse 系统设置响应
type SystemSettingResponse struct {
	ID          string `json:"id"`
	OrgID       string `json:"orgId"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// GetAllSettings 获取所有设置
// GET /api/v1/system-settings
func (h *HTTPHandler) GetAllSettings(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	settings, err := h.container.GetSystemSettingService().GetAllSettings(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to get all settings", "error", err)
		RespondInternalError(w, "Failed to get settings")
		return
	}

	responses := make([]*SystemSettingResponse, 0, len(settings))
	for _, setting := range settings {
		responses = append(responses, h.toSystemSettingResponse(setting))
	}

	RespondSuccess(w, responses)
}

// GetSetting 获取单个设置
// GET /api/v1/system-settings/{key}
func (h *HTTPHandler) GetSetting(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		RespondBadRequest(w, "key is required")
		return
	}

	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	value, err := h.container.GetSystemSettingService().GetSetting(r.Context(), orgID, key)
	if err != nil {
		h.logger.Error("Failed to get setting", "error", err, "key", key, "orgId", orgID)
		// 如果获取设置失败，尝试返回默认值（用于表还未创建的情况）
		defaultValue := model.GetSystemSettingDefaultValue(key)
		if defaultValue != "" {
			h.logger.Warn("Returning default value due to error", "key", key, "defaultValue", defaultValue, "error", err)
			RespondSuccess(w, map[string]string{
				"key":   key,
				"value": defaultValue,
			})
			return
		}
		RespondInternalError(w, fmt.Sprintf("Failed to get setting: %v", err))
		return
	}

	RespondSuccess(w, map[string]string{
		"key":   key,
		"value": value,
	})
}

// SetSetting 设置或更新系统设置
// PUT /api/v1/system-settings/{key}
func (h *HTTPHandler) SetSetting(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		RespondBadRequest(w, "key is required")
		return
	}

	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req SetSystemSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.Value == "" {
		RespondBadRequest(w, "value is required")
		return
	}

	err := h.container.GetSystemSettingService().SetSetting(r.Context(), orgID, key, req.Value, req.Description)
	if err != nil {
		h.logger.Error("Failed to set setting", "error", err, "key", key, "orgId", orgID, "value", req.Value)
		// 检查是否是表不存在的错误
		if errMsg := err.Error(); contains(errMsg, "Table") && contains(errMsg, "doesn't exist") {
			RespondInternalError(w, "Database table not found. Please restart the management service to create the system_settings table.")
			return
		}
		RespondInternalError(w, fmt.Sprintf("Failed to set setting: %v", err))
		return
	}

	// 返回更新后的设置
	value, err := h.container.GetSystemSettingService().GetSetting(r.Context(), orgID, key)
	if err != nil {
		h.logger.Error("Failed to get setting after update", "error", err, "key", key)
		RespondInternalError(w, "Failed to get setting after update")
		return
	}

	RespondSuccess(w, map[string]string{
		"key":   key,
		"value": value,
	})
}

// DeleteSetting 删除系统设置
// DELETE /api/v1/system-settings/{key}
func (h *HTTPHandler) DeleteSetting(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		RespondBadRequest(w, "key is required")
		return
	}

	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	err := h.container.GetSystemSettingService().DeleteSetting(r.Context(), orgID, key)
	if err != nil {
		h.logger.Error("Failed to delete setting", "error", err, "key", key)
		RespondInternalError(w, "Failed to delete setting")
		return
	}

	RespondSuccess(w, map[string]string{
		"message": "setting deleted",
	})
}

// toSystemSettingResponse 转换为响应格式
func (h *HTTPHandler) toSystemSettingResponse(setting *model.SystemSetting) *SystemSettingResponse {
	return &SystemSettingResponse{
		ID:          setting.ID,
		OrgID:       setting.OrgID,
		Key:         setting.Key,
		Value:       setting.Value,
		Description: setting.Description,
		CreatedAt:   setting.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   setting.UpdatedAt.Format(time.RFC3339),
	}
}

