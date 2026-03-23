package http

import (
	"encoding/json"
	"net/http"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateModalityRoomRequest 创建机房请求
type CreateModalityRoomRequest struct {
	OrgID       string `json:"orgId" validate:"required"`
	Code        string `json:"code" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Location    string `json:"location"`
	SortOrder   int    `json:"sortOrder"`
}

// UpdateModalityRoomRequest 更新机房请求
type UpdateModalityRoomRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"`
	SortOrder   int    `json:"sortOrder"`
	IsActive    *bool  `json:"isActive,omitempty"`
}

// CreateModalityRoom 创建机房
func (h *HTTPHandler) CreateModalityRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateModalityRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	modalityRoom := &model.ModalityRoom{
		OrgID:       req.OrgID,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Location:    req.Location,
		SortOrder:   req.SortOrder,
	}

	if err := h.container.ModalityRoomService().CreateModalityRoom(r.Context(), modalityRoom); err != nil {
		h.logger.Error("Failed to create modality room", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, modalityRoom)
}

// GetModalityRoom 获取机房详情
func (h *HTTPHandler) GetModalityRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	modalityRoom, err := h.container.ModalityRoomService().GetModalityRoom(r.Context(), orgID, id)
	if err != nil {
		h.logger.Error("Failed to get modality room", "error", err)
		RespondNotFound(w, "机房不存在")
		return
	}

	RespondSuccess(w, modalityRoom)
}

// UpdateModalityRoom 更新机房
func (h *HTTPHandler) UpdateModalityRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req UpdateModalityRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	// 先获取现有数据
	existing, err := h.container.ModalityRoomService().GetModalityRoom(r.Context(), orgID, id)
	if err != nil {
		RespondNotFound(w, "机房不存在")
		return
	}

	// 更新字段
	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.Description = req.Description
	existing.Location = req.Location
	existing.SortOrder = req.SortOrder
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.container.ModalityRoomService().UpdateModalityRoom(r.Context(), existing); err != nil {
		h.logger.Error("Failed to update modality room", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, existing)
}

// DeleteModalityRoom 删除机房
func (h *HTTPHandler) DeleteModalityRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.ModalityRoomService().DeleteModalityRoom(r.Context(), orgID, id); err != nil {
		h.logger.Error("Failed to delete modality room", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "删除成功"})
}

// ListModalityRooms 查询机房列表
func (h *HTTPHandler) ListModalityRooms(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	filter := &model.ModalityRoomFilter{
		OrgID:   orgID,
		Keyword: r.URL.Query().Get("keyword"),
	}

	if isActiveStr := r.URL.Query().Get("isActive"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	// 分页参数
	filter.Page, filter.PageSize = parsePageParams(r)

	result, err := h.container.ModalityRoomService().ListModalityRooms(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list modality rooms", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, result)
}

// GetActiveModalityRooms 获取所有启用的机房
func (h *HTTPHandler) GetActiveModalityRooms(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	modalityRooms, err := h.container.ModalityRoomService().GetActiveModalityRooms(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to get active modality rooms", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, modalityRooms)
}

// ToggleModalityRoomStatus 切换机房状态
func (h *HTTPHandler) ToggleModalityRoomStatus(w http.ResponseWriter, r *http.Request) {
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

	if err := h.container.ModalityRoomService().ToggleModalityRoomStatus(r.Context(), orgID, id, req.IsActive); err != nil {
		h.logger.Error("Failed to toggle modality room status", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "状态更新成功"})
}
