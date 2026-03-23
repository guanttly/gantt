package http

import (
	"encoding/json"
	"net/http"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateTimePeriodRequest 创建时间段请求
type CreateTimePeriodRequest struct {
	OrgID       string `json:"orgId" validate:"required"`
	Code        string `json:"code" validate:"required"`
	Name        string `json:"name" validate:"required"`
	StartTime   string `json:"startTime" validate:"required"` // HH:MM格式
	EndTime     string `json:"endTime" validate:"required"`   // HH:MM格式
	IsCrossDay  bool   `json:"isCrossDay"`                    // 是否跨日
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

// UpdateTimePeriodRequest 更新时间段请求
type UpdateTimePeriodRequest struct {
	Name        string `json:"name"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	IsCrossDay  bool   `json:"isCrossDay"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
	IsActive    *bool  `json:"isActive,omitempty"`
}

// CreateTimePeriod 创建时间段
func (h *HTTPHandler) CreateTimePeriod(w http.ResponseWriter, r *http.Request) {
	var req CreateTimePeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	timePeriod := &model.TimePeriod{
		OrgID:       req.OrgID,
		Code:        req.Code,
		Name:        req.Name,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		IsCrossDay:  req.IsCrossDay,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	if err := h.container.TimePeriodService().CreateTimePeriod(r.Context(), timePeriod); err != nil {
		h.logger.Error("Failed to create time period", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, timePeriod)
}

// GetTimePeriod 获取时间段详情
func (h *HTTPHandler) GetTimePeriod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	timePeriod, err := h.container.TimePeriodService().GetTimePeriod(r.Context(), orgID, id)
	if err != nil {
		h.logger.Error("Failed to get time period", "error", err)
		RespondNotFound(w, "时间段不存在")
		return
	}

	RespondSuccess(w, timePeriod)
}

// UpdateTimePeriod 更新时间段
func (h *HTTPHandler) UpdateTimePeriod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req UpdateTimePeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	// 先获取现有数据
	existing, err := h.container.TimePeriodService().GetTimePeriod(r.Context(), orgID, id)
	if err != nil {
		RespondNotFound(w, "时间段不存在")
		return
	}

	// 更新字段
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.StartTime != "" {
		existing.StartTime = req.StartTime
	}
	if req.EndTime != "" {
		existing.EndTime = req.EndTime
	}
	existing.IsCrossDay = req.IsCrossDay
	existing.Description = req.Description
	existing.SortOrder = req.SortOrder
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.container.TimePeriodService().UpdateTimePeriod(r.Context(), existing); err != nil {
		h.logger.Error("Failed to update time period", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, existing)
}

// DeleteTimePeriod 删除时间段
func (h *HTTPHandler) DeleteTimePeriod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.TimePeriodService().DeleteTimePeriod(r.Context(), orgID, id); err != nil {
		h.logger.Error("Failed to delete time period", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "删除成功"})
}

// ListTimePeriods 查询时间段列表
func (h *HTTPHandler) ListTimePeriods(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	filter := &model.TimePeriodFilter{
		OrgID:   orgID,
		Keyword: r.URL.Query().Get("keyword"),
	}

	if isActiveStr := r.URL.Query().Get("isActive"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	// 分页参数
	filter.Page, filter.PageSize = parsePageParams(r)

	result, err := h.container.TimePeriodService().ListTimePeriods(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list time periods", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, result)
}

// GetActiveTimePeriods 获取所有启用的时间段
func (h *HTTPHandler) GetActiveTimePeriods(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	timePeriods, err := h.container.TimePeriodService().GetActiveTimePeriods(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to get active time periods", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, timePeriods)
}

// ToggleTimePeriodStatus 切换时间段状态
func (h *HTTPHandler) ToggleTimePeriodStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req struct {
		IsActive bool `json:"isActive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if err := h.container.TimePeriodService().ToggleTimePeriodStatus(r.Context(), orgID, id, req.IsActive); err != nil {
		h.logger.Error("Failed to toggle time period status", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "状态更新成功"})
}
