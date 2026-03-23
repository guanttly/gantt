package http

import (
	"encoding/json"
	"net/http"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateScanTypeRequest 创建检查类型请求
type CreateScanTypeRequest struct {
	OrgID       string `json:"orgId" validate:"required"`
	Code        string `json:"code" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

// UpdateScanTypeRequest 更新检查类型请求
type UpdateScanTypeRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
	IsActive    *bool  `json:"isActive,omitempty"`
}

// CreateScanType 创建检查类型
func (h *HTTPHandler) CreateScanType(w http.ResponseWriter, r *http.Request) {
	var req CreateScanTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	scanType := &model.ScanType{
		OrgID:       req.OrgID,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	if err := h.container.ScanTypeService().CreateScanType(r.Context(), scanType); err != nil {
		h.logger.Error("Failed to create scan type", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, scanType)
}

// GetScanType 获取检查类型详情
func (h *HTTPHandler) GetScanType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	scanType, err := h.container.ScanTypeService().GetScanType(r.Context(), orgID, id)
	if err != nil {
		h.logger.Error("Failed to get scan type", "error", err)
		RespondNotFound(w, "检查类型不存在")
		return
	}

	RespondSuccess(w, scanType)
}

// UpdateScanType 更新检查类型
func (h *HTTPHandler) UpdateScanType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req UpdateScanTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	// 先获取现有数据
	existing, err := h.container.ScanTypeService().GetScanType(r.Context(), orgID, id)
	if err != nil {
		RespondNotFound(w, "检查类型不存在")
		return
	}

	// 更新字段
	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.Description = req.Description
	existing.SortOrder = req.SortOrder
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.container.ScanTypeService().UpdateScanType(r.Context(), existing); err != nil {
		h.logger.Error("Failed to update scan type", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, existing)
}

// DeleteScanType 删除检查类型
func (h *HTTPHandler) DeleteScanType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.ScanTypeService().DeleteScanType(r.Context(), orgID, id); err != nil {
		h.logger.Error("Failed to delete scan type", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "删除成功"})
}

// ListScanTypes 查询检查类型列表
func (h *HTTPHandler) ListScanTypes(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	filter := &model.ScanTypeFilter{
		OrgID:   orgID,
		Keyword: r.URL.Query().Get("keyword"),
	}

	if isActiveStr := r.URL.Query().Get("isActive"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	// 分页参数
	filter.Page, filter.PageSize = parsePageParams(r)

	result, err := h.container.ScanTypeService().ListScanTypes(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list scan types", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, result)
}

// GetActiveScanTypes 获取所有启用的检查类型
func (h *HTTPHandler) GetActiveScanTypes(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	scanTypes, err := h.container.ScanTypeService().GetActiveScanTypes(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to get active scan types", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, scanTypes)
}

// ToggleScanTypeStatus 切换检查类型状态
func (h *HTTPHandler) ToggleScanTypeStatus(w http.ResponseWriter, r *http.Request) {
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

	if err := h.container.ScanTypeService().ToggleScanTypeStatus(r.Context(), orgID, id, req.IsActive); err != nil {
		h.logger.Error("Failed to toggle scan type status", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "状态更新成功"})
}
