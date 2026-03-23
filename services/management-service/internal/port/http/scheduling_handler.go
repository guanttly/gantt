package http

import (
	"encoding/json"
	"net/http"
	"time"

	"jusha/gantt/service/management/domain/model"
)

// ======================== 请求类型定义 ========================

// BatchAssignRequest 批量分配排班请求
type BatchAssignRequest struct {
	OrgID       string                   `json:"orgId" validate:"required"`
	Assignments []ScheduleAssignmentItem `json:"assignments" validate:"required,min=1"`
}

// ScheduleAssignmentItem 单条排班分配记录
type ScheduleAssignmentItem struct {
	EmployeeID string `json:"employeeId" validate:"required"`
	ShiftID    string `json:"shiftId" validate:"required"`
	Date       string `json:"date" validate:"required"` // YYYY-MM-DD格式
	Notes      string `json:"notes"`
}

// GetScheduleByDateRangeRequest 按日期范围查询排班请求
type GetScheduleByDateRangeRequest struct {
	OrgID     string `json:"orgId" form:"orgId" validate:"required"`
	StartDate string `json:"startDate" form:"startDate" validate:"required"`
	EndDate   string `json:"endDate" form:"endDate" validate:"required"`
}

// GetEmployeeScheduleRequest 查询员工排班请求
type GetEmployeeScheduleRequest struct {
	OrgID      string `json:"orgId" form:"orgId" validate:"required"`
	EmployeeID string `json:"employeeId" form:"employeeId" validate:"required"`
	StartDate  string `json:"startDate" form:"startDate" validate:"required"`
	EndDate    string `json:"endDate" form:"endDate" validate:"required"`
}

// DeleteScheduleRequest 删除排班请求
type DeleteScheduleRequest struct {
	OrgID      string `json:"orgId" form:"orgId" validate:"required"`
	EmployeeID string `json:"employeeId" form:"employeeId" validate:"required"`
	Date       string `json:"date" form:"date" validate:"required"`
}

// BatchDeleteScheduleRequest 批量删除排班请求
type BatchDeleteScheduleRequest struct {
	OrgID       string   `json:"orgId" validate:"required"`
	EmployeeIDs []string `json:"employeeIds" validate:"required,min=1"`
	Dates       []string `json:"dates" validate:"required,min=1"` // YYYY-MM-DD格式
}

// ======================== HTTP 处理器 ========================

// BatchAssignSchedule 批量分配排班
func (h *HTTPHandler) BatchAssignSchedule(w http.ResponseWriter, r *http.Request) {
	var req BatchAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	// 转换为领域模型
	assignments := make([]*model.ShiftAssignment, 0, len(req.Assignments))
	for _, item := range req.Assignments {
		date, err := time.Parse("2006-01-02", item.Date)
		if err != nil {
			RespondBadRequest(w, "Invalid date format: "+item.Date)
			return
		}

		assignment := &model.ShiftAssignment{
			OrgID:      req.OrgID,
			EmployeeID: item.EmployeeID,
			ShiftID:    item.ShiftID,
			Date:       date,
			Notes:      item.Notes,
		}
		assignments = append(assignments, assignment)
	}

	// 调用服务层批量分配
	if err := h.container.GetSchedulingService().BatchAssignShifts(r.Context(), assignments); err != nil {
		h.logger.Error("Failed to batch assign schedule", "error", err)
		RespondInternalError(w, "Failed to batch assign schedule: "+err.Error())
		return
	}

	RespondSuccess(w, map[string]interface{}{
		"message": "Batch assignment completed successfully",
		"count":   len(assignments),
	})
}

// GetScheduleByDateRange 获取日期范围内的排班数据
func (h *HTTPHandler) GetScheduleByDateRange(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}
	if startDateStr == "" {
		RespondBadRequest(w, "startDate is required")
		return
	}
	if endDateStr == "" {
		RespondBadRequest(w, "endDate is required")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		RespondBadRequest(w, "Invalid startDate format, expected YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		RespondBadRequest(w, "Invalid endDate format, expected YYYY-MM-DD")
		return
	}

	assignments, err := h.container.GetSchedulingService().GetScheduleByDateRange(r.Context(), orgID, startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get schedule by date range", "error", err)
		RespondInternalError(w, "Failed to get schedule by date range")
		return
	}

	RespondSuccess(w, assignments)
}

// GetEmployeeSchedule 获取员工的排班数据
func (h *HTTPHandler) GetEmployeeSchedule(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	employeeID := r.URL.Query().Get("employeeId")
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}
	if employeeID == "" {
		RespondBadRequest(w, "employeeId is required")
		return
	}
	if startDateStr == "" {
		RespondBadRequest(w, "startDate is required")
		return
	}
	if endDateStr == "" {
		RespondBadRequest(w, "endDate is required")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		RespondBadRequest(w, "Invalid startDate format, expected YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		RespondBadRequest(w, "Invalid endDate format, expected YYYY-MM-DD")
		return
	}

	assignments, err := h.container.GetSchedulingService().GetEmployeeSchedule(r.Context(), orgID, employeeID, startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get employee schedule", "employeeId", employeeID, "error", err)
		RespondInternalError(w, "Failed to get employee schedule")
		return
	}

	RespondSuccess(w, assignments)
}

// DeleteScheduleAssignment 删除排班分配
func (h *HTTPHandler) DeleteScheduleAssignment(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	assignmentID := r.URL.Query().Get("id")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}
	if assignmentID == "" {
		RespondBadRequest(w, "id is required")
		return
	}

	if err := h.container.GetSchedulingService().DeleteScheduleAssignmentByID(r.Context(), orgID, assignmentID); err != nil {
		h.logger.Error("Failed to delete schedule assignment", "error", err)
		RespondInternalError(w, "Failed to delete schedule assignment")
		return
	}

	RespondSuccess(w, map[string]string{"message": "Schedule assignment deleted successfully"})
}

// BatchDeleteSchedule 批量删除排班
func (h *HTTPHandler) BatchDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if req.OrgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	// 转换日期字符串为time.Time
	dates := make([]time.Time, 0, len(req.Dates))
	for _, dateStr := range req.Dates {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			RespondBadRequest(w, "Invalid date format: "+dateStr)
			return
		}
		dates = append(dates, date)
	}

	if err := h.container.GetSchedulingService().BatchDeleteScheduleAssignments(r.Context(), req.OrgID, req.EmployeeIDs, dates); err != nil {
		h.logger.Error("Failed to batch delete schedule", "error", err)
		RespondInternalError(w, "Failed to batch delete schedule")
		return
	}

	RespondSuccess(w, map[string]interface{}{
		"message": "Batch delete completed successfully",
		"count":   len(req.EmployeeIDs) * len(dates),
	})
}

// GetScheduleSummary 获取排班汇总
func (h *HTTPHandler) GetScheduleSummary(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}
	if startDateStr == "" {
		RespondBadRequest(w, "startDate is required")
		return
	}
	if endDateStr == "" {
		RespondBadRequest(w, "endDate is required")
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		RespondBadRequest(w, "Invalid startDate format, expected YYYY-MM-DD")
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		RespondBadRequest(w, "Invalid endDate format, expected YYYY-MM-DD")
		return
	}

	summary, err := h.container.GetSchedulingService().GetScheduleSummary(r.Context(), orgID, startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get schedule summary", "error", err)
		RespondInternalError(w, "Failed to get schedule summary")
		return
	}

	RespondSuccess(w, summary)
}
