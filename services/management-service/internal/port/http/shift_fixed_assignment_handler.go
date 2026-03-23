package http

import (
	"encoding/json"
	"net/http"
	"time"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// BatchCreateFixedAssignmentsRequest 批量创建固定人员配置请求
type BatchCreateFixedAssignmentsRequest struct {
	Assignments []CreateFixedAssignmentRequest `json:"assignments"`
}

// CreateFixedAssignmentRequest 创建固定人员配置请求
type CreateFixedAssignmentRequest struct {
	StaffID       string     `json:"staffId" validate:"required"`
	PatternType   string     `json:"patternType" validate:"required"` // weekly, monthly, specific
	Weekdays      []int      `json:"weekdays"`                        // 周模式: 1-7
	WeekPattern   string     `json:"weekPattern"`                     // every, odd, even
	Monthdays     []int      `json:"monthdays"`                       // 月模式: 1-31
	SpecificDates []string   `json:"specificDates"`                   // 指定日期模式
	StartDate     *time.Time `json:"startDate"`                       // 生效开始日期
	EndDate       *time.Time `json:"endDate"`                         // 生效结束日期
	IsActive      bool       `json:"isActive"`
}

// ListFixedAssignmentsByShift 获取班次的所有固定人员配置
func (h *HTTPHandler) ListFixedAssignmentsByShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]

	if shiftID == "" {
		RespondBadRequest(w, "shiftId is required")
		return
	}

	assignments, err := h.container.GetShiftFixedAssignmentService().ListByShiftID(r.Context(), shiftID)
	if err != nil {
		h.logger.Error("Failed to list fixed assignments", "error", err, "shiftId", shiftID)
		RespondInternalError(w, "Failed to list fixed assignments")
		return
	}

	// 如果没有数据，返回空数组而不是 null
	if assignments == nil {
		assignments = []*model.ShiftFixedAssignment{}
	}

	RespondSuccess(w, assignments)
}

// BatchCreateFixedAssignments 批量创建固定人员配置
func (h *HTTPHandler) BatchCreateFixedAssignments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]

	if shiftID == "" {
		RespondBadRequest(w, "shiftId is required")
		return
	}

	var req BatchCreateFixedAssignmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	// 转换为领域模型
	assignments := make([]*model.ShiftFixedAssignment, 0, len(req.Assignments))
	for _, assignReq := range req.Assignments {
		assignment := &model.ShiftFixedAssignment{
			ShiftID:       shiftID,
			StaffID:       assignReq.StaffID,
			PatternType:   assignReq.PatternType,
			Weekdays:      assignReq.Weekdays,
			WeekPattern:   assignReq.WeekPattern,
			Monthdays:     assignReq.Monthdays,
			SpecificDates: assignReq.SpecificDates,
			StartDate:     assignReq.StartDate,
			EndDate:       assignReq.EndDate,
			IsActive:      assignReq.IsActive,
		}
		assignments = append(assignments, assignment)
	}

	// 批量创建
	if err := h.container.GetShiftFixedAssignmentService().BatchCreate(r.Context(), shiftID, assignments); err != nil {
		h.logger.Error("Failed to batch create fixed assignments", "error", err, "shiftId", shiftID)
		RespondInternalError(w, "Failed to batch create fixed assignments")
		return
	}

	RespondCreated(w, map[string]string{
		"message": "Fixed assignments created successfully",
	})
}

// DeleteFixedAssignment 删除固定人员配置
func (h *HTTPHandler) DeleteFixedAssignment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	assignmentID := vars["assignmentId"]

	if assignmentID == "" {
		RespondBadRequest(w, "assignmentId is required")
		return
	}

	if err := h.container.GetShiftFixedAssignmentService().Delete(r.Context(), assignmentID); err != nil {
		h.logger.Error("Failed to delete fixed assignment", "error", err, "assignmentId", assignmentID)
		RespondInternalError(w, "Failed to delete fixed assignment")
		return
	}

	RespondSuccess(w, map[string]string{
		"message": "Fixed assignment deleted successfully",
	})
}

// CalculateFixedScheduleRequest 计算固定排班请求
type CalculateFixedScheduleRequest struct {
	StartDate string `json:"startDate" validate:"required"`
	EndDate   string `json:"endDate" validate:"required"`
}

// CalculateFixedSchedule 计算固定班次在指定周期内的实际排班
func (h *HTTPHandler) CalculateFixedSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]

	if shiftID == "" {
		RespondBadRequest(w, "shiftId is required")
		return
	}

	var req CalculateFixedScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	schedule, err := h.container.GetShiftFixedAssignmentService().CalculateFixedSchedule(
		r.Context(),
		shiftID,
		req.StartDate,
		req.EndDate,
	)
	if err != nil {
		h.logger.Error("Failed to calculate fixed schedule", "error", err, "shiftId", shiftID)
		RespondInternalError(w, "Failed to calculate fixed schedule")
		return
	}

	RespondSuccess(w, schedule)
}

// CalculateMultipleFixedSchedulesRequest 批量计算固定排班请求
type CalculateMultipleFixedSchedulesRequest struct {
	ShiftIDs  []string `json:"shiftIds" validate:"required"`
	StartDate string   `json:"startDate" validate:"required"`
	EndDate   string   `json:"endDate" validate:"required"`
}

// CalculateMultipleFixedSchedules 批量计算多个班次的固定排班
func (h *HTTPHandler) CalculateMultipleFixedSchedules(w http.ResponseWriter, r *http.Request) {
	var req CalculateMultipleFixedSchedulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if len(req.ShiftIDs) == 0 {
		RespondBadRequest(w, "shiftIds is required and must not be empty")
		return
	}

	schedules, err := h.container.GetShiftFixedAssignmentService().CalculateMultipleFixedSchedules(
		r.Context(),
		req.ShiftIDs,
		req.StartDate,
		req.EndDate,
	)
	if err != nil {
		h.logger.Error("Failed to calculate multiple fixed schedules", "error", err, "shiftIds", req.ShiftIDs)
		RespondInternalError(w, "Failed to calculate multiple fixed schedules")
		return
	}

	RespondSuccess(w, schedules)
}

