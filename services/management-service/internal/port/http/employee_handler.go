package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateEmployeeRequest 创建员工请求
type CreateEmployeeRequest struct {
	OrgID      string               `json:"orgId" validate:"required"`
	EmployeeID string               `json:"employeeId" validate:"required"` // 工号
	Name       string               `json:"name" validate:"required"`
	Phone      string               `json:"phone"`
	Email      string               `json:"email"`
	Department string               `json:"department"`
	Position   string               `json:"position"`
	Status     model.EmployeeStatus `json:"status"`
}

// UpdateEmployeeRequest 更新员工请求
type UpdateEmployeeRequest struct {
	OrgID      string `json:"orgId"` // 组织ID（可选，优先使用body中的值）
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Department string `json:"department"`
	Position   string `json:"position"`
}

// UpdateEmployeeStatusRequest 更新员工状态请求
type UpdateEmployeeStatusRequest struct {
	Status model.EmployeeStatus `json:"status" validate:"required"`
}

// EmployeeResponse 员工响应（包含关联信息）
type EmployeeResponse struct {
	ID           string               `json:"id"`
	OrgID        string               `json:"orgId"`
	EmployeeID   string               `json:"employeeId"`
	Name         string               `json:"name"`
	Phone        string               `json:"phone"`
	Email        string               `json:"email"`
	DepartmentID string               `json:"departmentId"`         // 部门ID
	Department   *DepartmentInfo      `json:"department,omitempty"` // 部门详情
	Position     string               `json:"position"`
	Status       model.EmployeeStatus `json:"status"`
	Groups       []*GroupInfo         `json:"groups,omitempty"` // 所属分组列表
	HireDate     *string              `json:"hireDate,omitempty"`
	CreatedAt    string               `json:"createdAt"`
	UpdatedAt    string               `json:"updatedAt"`
}

// DepartmentInfo 部门信息
type DepartmentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// GroupInfo 分组信息
type GroupInfo struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// SimpleEmployeeResponse 简单员工响应（不包含关联信息）
type SimpleEmployeeResponse struct {
	ID           string               `json:"id"`
	OrgID        string               `json:"orgId"`
	EmployeeID   string               `json:"employeeId"`
	Name         string               `json:"name"`
	Phone        string               `json:"phone"`
	Email        string               `json:"email"`
	DepartmentID string               `json:"departmentId"`
	Position     string               `json:"position"`
	Status       model.EmployeeStatus `json:"status"`
}

// ListEmployeesRequest 查询员工列表请求
type ListEmployeesRequest struct {
	PageRequest
	OrgID      string               `json:"orgId" form:"orgId"`
	Keyword    string               `json:"keyword" form:"keyword"`
	Department string               `json:"department" form:"department"`
	Status     model.EmployeeStatus `json:"status" form:"status"`
}

// CreateEmployee 创建员工
func (h *HTTPHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req CreateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	employee := &model.Employee{
		OrgID:        req.OrgID,
		EmployeeID:   req.EmployeeID,
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		DepartmentID: req.Department,
		Position:     req.Position,
		Status:       req.Status,
	}

	if employee.Status == "" {
		employee.Status = model.EmployeeStatusActive
	}

	if err := h.container.GetEmployeeService().CreateEmployee(r.Context(), employee); err != nil {
		h.logger.Error("Failed to create employee", "error", err)
		RespondInternalError(w, "Failed to create employee")
		return
	}

	// 构建包含部门信息的响应
	resp := h.buildEmployeeResponse(r.Context(), employee)
	RespondCreated(w, resp)
}

// GetEmployee 获取员工详情
func (h *HTTPHandler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	employeeID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	employee, err := h.container.GetEmployeeService().GetEmployee(r.Context(), orgID, employeeID)
	if err != nil {
		h.logger.Error("Failed to get employee", "error", err)
		RespondNotFound(w, "Employee not found")
		return
	}

	// 构建包含部门信息的响应
	resp := h.buildEmployeeResponse(r.Context(), employee)
	RespondSuccess(w, resp)
}

// UpdateEmployee 更新员工信息
func (h *HTTPHandler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	employeeID := vars["id"]

	var req UpdateEmployeeRequest
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

	// 先获取现有员工
	employee, err := h.container.GetEmployeeService().GetEmployee(r.Context(), orgID, employeeID)
	if err != nil {
		RespondNotFound(w, "Employee not found")
		return
	}

	// 记录接收到的请求数据
	h.logger.Info("Update employee request",
		"employeeID", employeeID,
		"req.Name", req.Name,
		"req.Phone", req.Phone,
		"req.Email", req.Email,
		"req.Department", req.Department,
		"req.Position", req.Position,
	)

	// 更新字段 - 只更新非空字段
	if req.Name != "" {
		employee.Name = req.Name
	}
	if req.Phone != "" {
		employee.Phone = req.Phone
	}
	if req.Email != "" {
		employee.Email = req.Email
	}
	if req.Department != "" {
		employee.DepartmentID = req.Department
	}
	if req.Position != "" {
		employee.Position = req.Position
	}

	if err := h.container.GetEmployeeService().UpdateEmployee(r.Context(), employee); err != nil {
		h.logger.Error("Failed to update employee", "error", err)
		RespondInternalError(w, "Failed to update employee")
		return
	}

	// 构建包含部门信息的响应
	resp := h.buildEmployeeResponse(r.Context(), employee)
	RespondSuccess(w, resp)
}

// DeleteEmployee 删除员工
func (h *HTTPHandler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	employeeID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.GetEmployeeService().DeleteEmployee(r.Context(), orgID, employeeID); err != nil {
		h.logger.Error("Failed to delete employee", "error", err)
		RespondInternalError(w, "Failed to delete employee")
		return
	}

	RespondSuccess(w, map[string]string{"message": "Employee deleted successfully"})
}

// ListEmployees 查询员工列表
func (h *HTTPHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	var req ListEmployeesRequest

	// 解析查询参数
	req.OrgID = r.URL.Query().Get("orgId")
	req.Keyword = r.URL.Query().Get("keyword")
	req.Department = r.URL.Query().Get("department")
	req.Status = model.EmployeeStatus(r.URL.Query().Get("status"))

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

	// 解析 includeGroups 参数（默认 false，仅在需要时设为 true）
	includeGroups := false
	if includeGroupsStr := r.URL.Query().Get("includeGroups"); includeGroupsStr != "" {
		if includeGroupsStr == "true" || includeGroupsStr == "1" {
			includeGroups = true
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

	filter := &model.EmployeeFilter{
		OrgID:         req.OrgID,
		Keyword:       req.Keyword,
		Department:    req.Department,
		Status:        req.Status,
		IncludeGroups: includeGroups, // 根据查询参数决定是否加载分组信息
		Page:          req.Page,
		PageSize:      req.Size,
	}

	result, err := h.container.GetEmployeeService().ListEmployees(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list employees", "error", err)
		RespondInternalError(w, "Failed to list employees")
		return
	}

	// 构建包含部门信息的响应列表
	employeeResponses := h.buildEmployeeListResponse(r.Context(), result.Items)

	RespondPage(w, result.Total, req.Page, req.Size, employeeResponses)
}

// ListSimpleEmployees 查询简单员工列表（不包含分组和部门详情）
func (h *HTTPHandler) ListSimpleEmployees(w http.ResponseWriter, r *http.Request) {
	var req ListEmployeesRequest

	// 解析查询参数
	req.OrgID = r.URL.Query().Get("orgId")
	req.Keyword = r.URL.Query().Get("keyword")
	req.Status = model.EmployeeStatus(r.URL.Query().Get("status"))

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
		req.Size = 50
	}
	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	filter := &model.EmployeeFilter{
		OrgID:    req.OrgID,
		Keyword:  req.Keyword,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.Size,
	}

	// 使用 ListSimpleEmployees 方法（不查询分组和部门详情）
	result, err := h.container.GetEmployeeService().ListSimpleEmployees(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list simple employees", "error", err)
		RespondInternalError(w, "Failed to list employees")
		return
	}

	// 构建简单响应列表（不包含分组和部门详情）
	employeeResponses := h.buildSimpleEmployeeListResponse(result.Items)

	RespondPage(w, result.Total, req.Page, req.Size, employeeResponses)
}

// buildEmployeeResponse 构建员工响应（包含部门信息）
func (h *HTTPHandler) buildEmployeeResponse(ctx context.Context, employee *model.Employee) *EmployeeResponse {
	resp := &EmployeeResponse{
		ID:           employee.ID,
		OrgID:        employee.OrgID,
		EmployeeID:   employee.EmployeeID,
		Name:         employee.Name,
		Phone:        employee.Phone,
		Email:        employee.Email,
		DepartmentID: employee.DepartmentID,
		Position:     employee.Position,
		Status:       employee.Status,
		CreatedAt:    employee.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    employee.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if employee.HireDate != nil {
		hireDate := employee.HireDate.Format("2006-01-02")
		resp.HireDate = &hireDate
	}

	// 如果有部门ID，获取部门信息
	if employee.DepartmentID != "" {
		if dept, err := h.container.GetDepartmentService().GetDepartment(ctx, employee.OrgID, employee.DepartmentID); err == nil {
			resp.Department = &DepartmentInfo{
				ID:   dept.ID,
				Name: dept.Name,
				Code: dept.Code,
			}
		}
	}

	// 添加分组信息
	if len(employee.Groups) > 0 {
		resp.Groups = make([]*GroupInfo, 0, len(employee.Groups))
		for _, group := range employee.Groups {
			resp.Groups = append(resp.Groups, &GroupInfo{
				ID:   group.ID,
				Code: group.Code,
				Name: group.Name,
				Type: string(group.Type),
			})
		}
	}

	return resp
}

// buildEmployeeListResponse 构建员工列表响应
func (h *HTTPHandler) buildEmployeeListResponse(ctx context.Context, employees []*model.Employee) []*EmployeeResponse {
	responses := make([]*EmployeeResponse, 0, len(employees))
	for _, emp := range employees {
		responses = append(responses, h.buildEmployeeResponse(ctx, emp))
	}
	return responses
}

// buildSimpleEmployeeListResponse 构建简单员工列表响应（不包含分组和部门详情）
func (h *HTTPHandler) buildSimpleEmployeeListResponse(employees []*model.Employee) []*SimpleEmployeeResponse {
	responses := make([]*SimpleEmployeeResponse, 0, len(employees))
	for _, emp := range employees {
		responses = append(responses, &SimpleEmployeeResponse{
			ID:           emp.ID,
			OrgID:        emp.OrgID,
			EmployeeID:   emp.EmployeeID,
			Name:         emp.Name,
			Phone:        emp.Phone,
			Email:        emp.Email,
			DepartmentID: emp.DepartmentID,
			Position:     emp.Position,
			Status:       emp.Status,
		})
	}
	return responses
}

// UpdateEmployeeStatus 更新员工状态
func (h *HTTPHandler) UpdateEmployeeStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	employeeID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req UpdateEmployeeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if err := h.container.GetEmployeeService().UpdateEmployeeStatus(r.Context(), orgID, employeeID, req.Status); err != nil {
		h.logger.Error("Failed to update employee status", "error", err)
		RespondInternalError(w, "Failed to update employee status")
		return
	}

	RespondSuccess(w, map[string]string{"message": "Employee status updated successfully"})
}
