package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateGroupRequest 创建分组请求
type CreateGroupRequest struct {
	OrgID       string          `json:"orgId" validate:"required"`
	Code        string          `json:"code" validate:"required"`
	Name        string          `json:"name" validate:"required"`
	Type        model.GroupType `json:"type" validate:"required"`
	ParentID    string          `json:"parentId"`
	Description string          `json:"description"`
}

// UpdateGroupRequest 更新分组请求
type UpdateGroupRequest struct {
	OrgID       string `json:"orgId"` // 组织ID（可选，优先使用body中的值）
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListGroupsRequest 查询分组列表请求
type ListGroupsRequest struct {
	PageRequest
	OrgID    string          `json:"orgId" form:"orgId"`
	Type     model.GroupType `json:"type" form:"type"`
	ParentID string          `json:"parentId" form:"parentId"`
	Keyword  string          `json:"keyword" form:"keyword"`
}

// AddGroupMemberRequest 添加分组成员请求
type AddGroupMemberRequest struct {
	EmployeeID string `json:"employeeId" validate:"required"`
	Role       string `json:"role"`
}

// BatchAddGroupMembersRequest 批量添加分组成员请求
type BatchAddGroupMembersRequest struct {
	EmployeeIDs []string `json:"employeeIds" validate:"required"`
	Role        string   `json:"role"`
}

// ListGroupMembersRequest 查询分组成员列表请求
type ListGroupMembersRequest struct {
	PageRequest
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required"`
}

// CreateGroup 创建分组
func (h *HTTPHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	group := &model.Group{
		OrgID:       req.OrgID,
		Code:        req.Code,
		Name:        req.Name,
		Type:        req.Type,
		ParentID:    &req.ParentID,
		Description: req.Description,
	}

	if err := h.container.GetGroupService().CreateGroup(r.Context(), group); err != nil {
		h.logger.Error("Failed to create group", "error", err)
		// 检查是否为唯一键冲突错误
		errMsg := err.Error()
		if contains(errMsg, "duplicate") || contains(errMsg, "unique constraint") || contains(errMsg, "duplicated key") {
			RespondConflict(w, "分组编码已存在，请使用其他编码")
			return
		}
		RespondInternalError(w, "Failed to create group")
		return
	}

	RespondCreated(w, group)
}

// GetGroup 获取分组详情
func (h *HTTPHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	group, err := h.container.GetGroupService().GetGroup(r.Context(), orgID, groupID)
	if err != nil {
		h.logger.Error("Failed to get group", "error", err)
		RespondNotFound(w, "Group not found")
		return
	}

	RespondSuccess(w, group)
}

// UpdateGroup 更新分组信息
func (h *HTTPHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	// 优先使用 body 中的 orgId，如果没有则从 query 参数获取
	orgID := req.OrgID
	if orgID == "" {
		orgID = r.URL.Query().Get("orgId")
	}

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	// 先获取现有分组
	group, err := h.container.GetGroupService().GetGroup(r.Context(), orgID, groupID)
	if err != nil {
		RespondNotFound(w, "Group not found")
		return
	}

	// 更新字段
	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}

	if err := h.container.GetGroupService().UpdateGroup(r.Context(), group); err != nil {
		h.logger.Error("Failed to update group", "error", err)
		RespondInternalError(w, "Failed to update group")
		return
	}

	RespondSuccess(w, group)
}

// DeleteGroup 删除分组
func (h *HTTPHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.GetGroupService().DeleteGroup(r.Context(), orgID, groupID); err != nil {
		h.logger.Error("Failed to delete group", "error", err)
		RespondInternalError(w, "Failed to delete group")
		return
	}

	RespondSuccess(w, map[string]string{"message": "Group deleted successfully"})
}

// ListGroups 查询分组列表
func (h *HTTPHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	var req ListGroupsRequest

	// 解析查询参数
	req.OrgID = r.URL.Query().Get("orgId")
	req.Type = model.GroupType(r.URL.Query().Get("type"))
	req.ParentID = r.URL.Query().Get("parentId")
	req.Keyword = r.URL.Query().Get("keyword")

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = p
		}
	}
	if size := r.URL.Query().Get("size"); size != "" {
		if s, err := strconv.Atoi(size); err == nil {
			req.Size = s
		}
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 20
	}
	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	filter := &model.GroupFilter{
		OrgID:    req.OrgID,
		Type:     req.Type,
		ParentID: &req.ParentID,
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.Size,
	}

	result, err := h.container.GetGroupService().ListGroups(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list groups", "error", err)
		RespondInternalError(w, "Failed to list groups")
		return
	}

	RespondPage(w, result.Total, req.Page, req.Size, result.Items)
}

// GetGroupMembers 获取分组成员列表
func (h *HTTPHandler) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	// 解析分页参数
	var req ListGroupMembersRequest
	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = p
		}
	}
	if size := r.URL.Query().Get("size"); size != "" {
		if s, err := strconv.Atoi(size); err == nil {
			req.Size = s
		}
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 20
	}

	// 获取所有成员
	members, err := h.container.GetGroupService().GetMembers(r.Context(), groupID)
	if err != nil {
		h.logger.Error("Failed to get group members", "error", err)
		RespondInternalError(w, "Failed to get group members")
		return
	}

	// 手动分页
	total := int64(len(members))
	start := (req.Page - 1) * req.Size
	end := start + req.Size

	if start >= int(total) {
		RespondPage(w, total, req.Page, req.Size, []*model.Employee{})
		return
	}

	if end > int(total) {
		end = int(total)
	}

	pagedMembers := members[start:end]
	RespondPage(w, total, req.Page, req.Size, pagedMembers)
}

// AddGroupMember 添加分组成员
func (h *HTTPHandler) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	var req AddGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if err := h.container.GetGroupService().AddMember(r.Context(), groupID, req.EmployeeID, req.Role); err != nil {
		h.logger.Error("Failed to add group member", "error", err)
		RespondInternalError(w, "Failed to add group member")
		return
	}

	RespondCreated(w, map[string]string{"message": "Member added successfully"})
}

// BatchAddGroupMembers 批量添加分组成员
func (h *HTTPHandler) BatchAddGroupMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	var req BatchAddGroupMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if len(req.EmployeeIDs) == 0 {
		RespondBadRequest(w, "employeeIds is required")
		return
	}

	if err := h.container.GetGroupService().BatchAddMembers(r.Context(), groupID, req.EmployeeIDs, req.Role); err != nil {
		h.logger.Error("Failed to batch add group members", "error", err)
		RespondInternalError(w, "Failed to batch add group members")
		return
	}

	RespondCreated(w, map[string]string{"message": "Members added successfully"})
}

// RemoveGroupMember 移除分组成员
func (h *HTTPHandler) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]
	employeeID := vars["employeeId"]

	if err := h.container.GetGroupService().RemoveMember(r.Context(), groupID, employeeID); err != nil {
		h.logger.Error("Failed to remove group member", "error", err)
		RespondInternalError(w, "Failed to remove group member")
		return
	}

	RespondSuccess(w, map[string]string{"message": "Member removed successfully"})
}

// UpdateGroupMemberRole 更新分组成员角色
func (h *HTTPHandler) UpdateGroupMemberRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]
	employeeID := vars["employeeId"]

	var req UpdateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if err := h.container.GetGroupService().UpdateMemberRole(r.Context(), groupID, employeeID, req.Role); err != nil {
		h.logger.Error("Failed to update member role", "error", err)
		RespondInternalError(w, "Failed to update member role")
		return
	}

	RespondSuccess(w, map[string]string{"message": "Member role updated successfully"})
}
