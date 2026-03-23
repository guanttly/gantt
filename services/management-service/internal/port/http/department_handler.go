package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateDepartmentRequest 创建部门请求
type CreateDepartmentRequest struct {
	OrgID       string  `json:"orgId" validate:"required"`
	Code        string  `json:"code" validate:"required"`
	Name        string  `json:"name" validate:"required"`
	ParentID    *string `json:"parentId"`
	Description string  `json:"description"`
	ManagerID   *string `json:"managerId"`
	SortOrder   int     `json:"sortOrder"`
}

// UpdateDepartmentRequest 更新部门请求
type UpdateDepartmentRequest struct {
	OrgID       string  `json:"orgId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ManagerID   *string `json:"managerId"`
	SortOrder   int     `json:"sortOrder"`
	IsActive    *bool   `json:"isActive"`
}

// ListDepartmentsRequest 查询部门列表请求
type ListDepartmentsRequest struct {
	PageRequest
	OrgID    string  `json:"orgId" form:"orgId"`
	ParentID *string `json:"parentId" form:"parentId"` // 用于查询子部门
	Keyword  string  `json:"keyword" form:"keyword"`
	IsActive *bool   `json:"isActive" form:"isActive"`
}

// CreateDepartment 创建部门
func (h *HTTPHandler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	department := &model.Department{
		OrgID:       req.OrgID,
		Code:        req.Code,
		Name:        req.Name,
		ParentID:    req.ParentID,
		Description: req.Description,
		ManagerID:   req.ManagerID,
		SortOrder:   req.SortOrder,
	}

	if err := h.container.GetDepartmentService().CreateDepartment(r.Context(), department); err != nil {
		h.logger.Error("Failed to create department", "error", err)
		RespondInternalError(w, "Failed to create department")
		return
	}

	RespondCreated(w, department)
}

// GetDepartment 获取部门详情
func (h *HTTPHandler) GetDepartment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	departmentID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	department, err := h.container.GetDepartmentService().GetDepartment(r.Context(), orgID, departmentID)
	if err != nil {
		h.logger.Error("Failed to get department", "error", err)
		RespondNotFound(w, "Department not found")
		return
	}

	RespondSuccess(w, department)
}

// UpdateDepartment 更新部门信息
func (h *HTTPHandler) UpdateDepartment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	departmentID := vars["id"]

	var req UpdateDepartmentRequest
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

	// 先获取现有部门
	department, err := h.container.GetDepartmentService().GetDepartment(r.Context(), orgID, departmentID)
	if err != nil {
		RespondNotFound(w, "Department not found")
		return
	}

	// 更新字段
	if req.Name != "" {
		department.Name = req.Name
	}
	if req.Description != "" {
		department.Description = req.Description
	}
	if req.ManagerID != nil {
		department.ManagerID = req.ManagerID
	}
	if req.SortOrder > 0 {
		department.SortOrder = req.SortOrder
	}
	if req.IsActive != nil {
		department.IsActive = *req.IsActive
	}

	if err := h.container.GetDepartmentService().UpdateDepartment(r.Context(), department); err != nil {
		h.logger.Error("Failed to update department", "error", err)
		RespondInternalError(w, "Failed to update department")
		return
	}

	RespondSuccess(w, department)
}

// DeleteDepartment 删除部门
func (h *HTTPHandler) DeleteDepartment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	departmentID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.GetDepartmentService().DeleteDepartment(r.Context(), orgID, departmentID); err != nil {
		h.logger.Error("Failed to delete department", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "Department deleted successfully"})
}

// ListDepartments 查询部门列表
func (h *HTTPHandler) ListDepartments(w http.ResponseWriter, r *http.Request) {
	var req ListDepartmentsRequest

	// 解析查询参数
	req.OrgID = r.URL.Query().Get("orgId")
	req.Keyword = r.URL.Query().Get("keyword")

	if parentID := r.URL.Query().Get("parentId"); parentID != "" {
		req.ParentID = &parentID
	}

	if isActive := r.URL.Query().Get("isActive"); isActive != "" {
		if active, err := strconv.ParseBool(isActive); err == nil {
			req.IsActive = &active
		}
	}

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

	filter := &model.DepartmentFilter{
		OrgID:    req.OrgID,
		ParentID: req.ParentID,
		Keyword:  req.Keyword,
		IsActive: req.IsActive,
		Page:     req.Page,
		PageSize: req.Size,
	}

	result, err := h.container.GetDepartmentService().ListDepartments(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list departments", "error", err)
		RespondInternalError(w, "Failed to list departments")
		return
	}

	RespondSuccess(w, result)
}

// GetDepartmentTree 获取部门树
func (h *HTTPHandler) GetDepartmentTree(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	tree, err := h.container.GetDepartmentService().GetDepartmentTree(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to get department tree", "error", err)
		RespondInternalError(w, "Failed to get department tree")
		return
	}

	RespondSuccess(w, tree)
}

// GetActiveDepartments 获取所有启用的部门
func (h *HTTPHandler) GetActiveDepartments(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	departments, err := h.container.GetDepartmentService().GetActiveDepartments(r.Context(), orgID)
	if err != nil {
		h.logger.Error("Failed to get active departments", "error", err)
		RespondInternalError(w, "Failed to get active departments")
		return
	}

	RespondSuccess(w, departments)
}
