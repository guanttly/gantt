package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateLeaveRequest 创建假期申请请求
type CreateLeaveRequest struct {
	OrgID      string          `json:"orgId" validate:"required"`
	EmployeeID string          `json:"employeeId" validate:"required"`
	Type       model.LeaveType `json:"type" validate:"required"`
	StartDate  string          `json:"startDate" validate:"required"` // YYYY-MM-DD
	EndDate    string          `json:"endDate" validate:"required"`   // YYYY-MM-DD
	StartTime  *string         `json:"startTime"`                     // HH:MM（可选，用于小时级请假）
	EndTime    *string         `json:"endTime"`                       // HH:MM（可选，用于小时级请假）
	Reason     string          `json:"reason"`
}

// UpdateLeaveRequest 更新假期申请请求
type UpdateLeaveRequest struct {
	OrgID     string  `json:"orgId"`     // 组织ID（可选，优先使用body中的值）
	StartDate string  `json:"startDate"` // YYYY-MM-DD
	EndDate   string  `json:"endDate"`   // YYYY-MM-DD
	StartTime *string `json:"startTime"` // HH:MM
	EndTime   *string `json:"endTime"`   // HH:MM
	Reason    string  `json:"reason"`
}

// ListLeavesRequest 查询假期列表请求
type ListLeavesRequest struct {
	PageRequest
	OrgID      string          `json:"orgId" form:"orgId"`
	EmployeeID string          `json:"employeeId" form:"employeeId"`
	Keyword    string          `json:"keyword" form:"keyword"` // 员工姓名或工号搜索
	Type       model.LeaveType `json:"type" form:"type"`
	StartDate  string          `json:"startDate" form:"startDate"` // YYYY-MM-DD
	EndDate    string          `json:"endDate" form:"endDate"`     // YYYY-MM-DD
}

// InitializeLeaveBalanceRequest 初始化假期余额请求
type InitializeLeaveBalanceRequest struct {
	OrgID      string          `json:"orgId" validate:"required"`
	EmployeeID string          `json:"employeeId" validate:"required"`
	Type       model.LeaveType `json:"type" validate:"required"`
	Year       int             `json:"year" validate:"required"`
	TotalDays  float64         `json:"totalDays" validate:"required,min=0"`
}

// CreateLeave 创建假期申请
func (h *HTTPHandler) CreateLeave(w http.ResponseWriter, r *http.Request) {
	var req CreateLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		RespondBadRequest(w, "Invalid startDate format")
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		RespondBadRequest(w, "Invalid endDate format")
		return
	}

	// 使用服务层计算请假天数（考虑工作日、节假日、小时级请假）
	days, err := h.container.GetLeaveService().CalculateLeaveDays(r.Context(), req.OrgID, startDate, endDate, req.StartTime, req.EndTime)
	if err != nil {
		RespondBadRequest(w, "Failed to calculate leave days: "+err.Error())
		return
	}

	leave := &model.LeaveRecord{
		OrgID:      req.OrgID,
		EmployeeID: req.EmployeeID,
		Type:       req.Type,
		StartDate:  startDate,
		EndDate:    endDate,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Days:       days,
		Reason:     req.Reason,
	}

	if err := h.container.GetLeaveService().CreateLeave(r.Context(), leave); err != nil {
		h.logger.Error("Failed to create leave", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondCreated(w, leave)
}

// GetLeave 获取假期详情
func (h *HTTPHandler) GetLeave(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	leaveID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	leave, err := h.container.GetLeaveService().GetLeave(r.Context(), orgID, leaveID)
	if err != nil {
		h.logger.Error("Failed to get leave", "error", err)
		RespondNotFound(w, "Leave record not found")
		return
	}

	RespondSuccess(w, leave)
}

// UpdateLeave 更新假期申请
func (h *HTTPHandler) UpdateLeave(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	leaveID := vars["id"]

	var req UpdateLeaveRequest
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

	// 先获取现有假期
	leave, err := h.container.GetLeaveService().GetLeave(r.Context(), orgID, leaveID)
	if err != nil {
		RespondNotFound(w, "Leave record not found")
		return
	}
	// 更新字段
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			RespondBadRequest(w, "Invalid startDate format")
			return
		}
		leave.StartDate = startDate
	}
	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			RespondBadRequest(w, "Invalid endDate format")
			return
		}
		leave.EndDate = endDate
	}
	if req.StartTime != nil {
		leave.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		leave.EndTime = req.EndTime
	}
	if req.Reason != "" {
		leave.Reason = req.Reason
	}

	// 重新计算天数（使用真实计算逻辑）
	days, err := h.container.GetLeaveService().CalculateLeaveDays(r.Context(), orgID, leave.StartDate, leave.EndDate, leave.StartTime, leave.EndTime)
	if err != nil {
		RespondBadRequest(w, "Failed to calculate leave days: "+err.Error())
		return
	}
	leave.Days = days

	if err := h.container.GetLeaveService().UpdateLeave(r.Context(), leave); err != nil {
		h.logger.Error("Failed to update leave", "error", err)
		RespondInternalError(w, "Failed to update leave")
		return
	}

	RespondSuccess(w, leave)
}

// DeleteLeave 删除假期申请
func (h *HTTPHandler) DeleteLeave(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	leaveID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.GetLeaveService().DeleteLeave(r.Context(), orgID, leaveID); err != nil {
		h.logger.Error("Failed to delete leave", "error", err)
		RespondInternalError(w, "Failed to delete leave")
		return
	}

	RespondSuccess(w, map[string]string{"message": "Leave record deleted successfully"})
}

// ListLeaves 查询假期列表
func (h *HTTPHandler) ListLeaves(w http.ResponseWriter, r *http.Request) {
	var req ListLeavesRequest

	// 解析查询参数
	req.OrgID = r.URL.Query().Get("orgId")
	req.EmployeeID = r.URL.Query().Get("employeeId")
	req.Keyword = r.URL.Query().Get("keyword")
	req.Type = model.LeaveType(r.URL.Query().Get("type"))
	req.StartDate = r.URL.Query().Get("startDate")
	req.EndDate = r.URL.Query().Get("endDate")

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

	filter := &model.LeaveFilter{
		OrgID:    req.OrgID,
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.Size,
	}

	// 只有非空时才设置过滤条件
	if req.EmployeeID != "" {
		filter.EmployeeID = &req.EmployeeID
	}
	if req.Type != "" {
		filter.Type = &req.Type
	}

	// 解析日期范围
	if req.StartDate != "" {
		if startDate, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			filter.StartDate = &startDate
		}
	}
	if req.EndDate != "" {
		if endDate, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			filter.EndDate = &endDate
		}
	}

	result, err := h.container.GetLeaveService().ListLeaves(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list leaves", "error", err)
		RespondInternalError(w, "Failed to list leaves")
		return
	}

	RespondPage(w, result.Total, req.Page, req.Size, result.Items)
}

// GetLeaveBalance 获取员工假期余额
func (h *HTTPHandler) GetLeaveBalance(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	employeeID := r.URL.Query().Get("employeeId")
	yearStr := r.URL.Query().Get("year")

	if orgID == "" || employeeID == "" {
		RespondBadRequest(w, "orgId and employeeId are required")
		return
	}

	year := time.Now().Year()
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	// 注意：当前Service接口只支持按类型查询单个余额
	// 如果需要查询所有类型，需要循环查询或修改Service接口
	// 这里简化处理，返回提示信息
	RespondSuccess(w, map[string]interface{}{
		"message": "Please specify leave_type parameter to query specific balance",
		"year":    year,
	})
}

// InitializeLeaveBalance 初始化假期余额
func (h *HTTPHandler) InitializeLeaveBalance(w http.ResponseWriter, r *http.Request) {
	var req InitializeLeaveBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if err := h.container.GetLeaveService().InitializeLeaveBalance(r.Context(), req.OrgID, req.EmployeeID, req.Type, req.Year, req.TotalDays); err != nil {
		h.logger.Error("Failed to initialize leave balance", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondCreated(w, map[string]string{"message": "Leave balance initialized successfully"})
}
