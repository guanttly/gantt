package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// CreateShiftRequest 创建班次请求
type CreateShiftRequest struct {
	OrgID              string          `json:"orgId" validate:"required"`
	Code               string          `json:"code" validate:"required"`
	Name               string          `json:"name" validate:"required"`
	Type               model.ShiftType `json:"type" validate:"required"`
	StartTime          string          `json:"startTime" validate:"required"` // HH:MM格式
	EndTime            string          `json:"endTime" validate:"required"`   // HH:MM格式
	Duration           int             `json:"duration" validate:"required"`  // 时长（分钟）
	IsOvernight        bool            `json:"isOvernight"`
	Color              string          `json:"color"`
	Description        string          `json:"description"`
	Priority           int             `json:"priority"`
	SchedulingPriority int             `json:"schedulingPriority"` // 排班优先级
	IsActive           bool            `json:"isActive"`           // 是否启用
}

// UpdateShiftRequest 更新班次请求
type UpdateShiftRequest struct {
	OrgID              string `json:"orgId"` // 组织ID（可选，优先使用body中的值）
	Name               string `json:"name"`
	StartTime          string `json:"startTime"` // HH:MM格式
	EndTime            string `json:"endTime"`   // HH:MM格式
	Duration           int    `json:"duration"`  // 时长（分钟）
	IsOvernight        bool   `json:"isOvernight"`
	Color              string `json:"color"`
	Description        string `json:"description"`
	Priority           int    `json:"priority"`
	SchedulingPriority int    `json:"schedulingPriority"` // 排班优先级
	IsActive           *bool  `json:"isActive,omitempty"` // 是否启用
}

// ListShiftsRequest 查询班次列表请求
type ListShiftsRequest struct {
	PageRequest
	OrgID    string           `json:"orgId" form:"orgId"`
	Type     *model.ShiftType `json:"type" form:"type"`
	IsActive *bool            `json:"isActive" form:"isActive"`
	Keyword  string           `json:"keyword" form:"keyword"`
}

// AssignShiftRequest 分配班次请求
type AssignShiftRequest struct {
	EmployeeID string    `json:"employeeId" validate:"required"`
	Date       time.Time `json:"date" validate:"required"`
}

// BatchAssignShiftRequest 批量分配班次请求
type BatchAssignShiftRequest struct {
	ShiftID     string   `json:"shiftId" validate:"required"`
	EmployeeIDs []string `json:"employeeIds" validate:"required,min=1"`
	Dates       []string `json:"dates" validate:"required,min=1"` // YYYY-MM-DD格式
}

// CreateShift 创建班次
func (h *HTTPHandler) CreateShift(w http.ResponseWriter, r *http.Request) {
	var req CreateShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	shift := &model.Shift{
		OrgID:              req.OrgID,
		Code:               req.Code,
		Name:               req.Name,
		Type:               req.Type,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		Duration:           req.Duration,
		IsOvernight:        req.IsOvernight,
		Color:              req.Color,
		Description:        req.Description,
		Priority:           req.Priority,
		SchedulingPriority: req.SchedulingPriority,
		IsActive:           req.IsActive,
	}

	// 验证班次时间
	if err := shift.ValidateShiftTime(); err != nil {
		RespondBadRequest(w, err.Error())
		return
	}

	if err := h.container.GetShiftService().CreateShift(r.Context(), shift); err != nil {
		h.logger.Error("Failed to create shift", "error", err)
		RespondInternalError(w, "Failed to create shift")
		return
	}

	RespondCreated(w, shift)
}

// GetShift 获取班次详情
func (h *HTTPHandler) GetShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	shift, err := h.container.GetShiftService().GetShift(r.Context(), orgID, shiftID)
	if err != nil {
		h.logger.Error("Failed to get shift", "error", err)
		RespondNotFound(w, "Shift not found")
		return
	}

	RespondSuccess(w, shift)
}

// UpdateShift 更新班次信息
func (h *HTTPHandler) UpdateShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]

	var req UpdateShiftRequest
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

	// 先获取现有班次
	shift, err := h.container.GetShiftService().GetShift(r.Context(), orgID, shiftID)
	if err != nil {
		RespondNotFound(w, "Shift not found")
		return
	}

	// 更新字段
	if req.Name != "" {
		shift.Name = req.Name
	}
	if req.StartTime != "" {
		shift.StartTime = req.StartTime
	}
	if req.EndTime != "" {
		shift.EndTime = req.EndTime
	}
	if req.Duration > 0 {
		shift.Duration = req.Duration
	}
	shift.IsOvernight = req.IsOvernight
	if req.Color != "" {
		shift.Color = req.Color
	}
	if req.Description != "" {
		shift.Description = req.Description
	}
	if req.Priority >= 0 {
		shift.Priority = req.Priority
	}
	if req.SchedulingPriority >= 0 {
		shift.SchedulingPriority = req.SchedulingPriority
	}

	// 验证班次时间
	if err := shift.ValidateShiftTime(); err != nil {
		RespondBadRequest(w, err.Error())
		return
	}

	if err := h.container.GetShiftService().UpdateShift(r.Context(), shift); err != nil {
		h.logger.Error("Failed to update shift", "error", err)
		RespondInternalError(w, "Failed to update shift")
		return
	}

	RespondSuccess(w, shift)
}

// DeleteShift 删除班次
func (h *HTTPHandler) DeleteShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.GetShiftService().DeleteShift(r.Context(), orgID, shiftID); err != nil {
		h.logger.Error("Failed to delete shift", "error", err)
		RespondInternalError(w, "Failed to delete shift")
		return
	}

	RespondSuccess(w, map[string]string{"message": "Shift deleted successfully"})
}

// ListShifts 查询班次列表
func (h *HTTPHandler) ListShifts(w http.ResponseWriter, r *http.Request) {
	var req ListShiftsRequest

	// 解析查询参数
	req.OrgID = r.URL.Query().Get("orgId")

	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		shiftType := model.ShiftType(typeStr)
		req.Type = &shiftType
	}

	if isActiveStr := r.URL.Query().Get("isActive"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			req.IsActive = &isActive
		}
	}

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

	filter := &model.ShiftFilter{
		OrgID:    req.OrgID,
		Type:     req.Type,
		IsActive: req.IsActive,
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.Size,
	}

	result, err := h.container.GetShiftService().ListShifts(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list shifts", "error", err)
		RespondInternalError(w, "Failed to list shifts")
		return
	}

	// 批量获取周人数配置并设置摘要
	if len(result.Items) > 0 {
		shiftIDs := make([]string, len(result.Items))
		for i, shift := range result.Items {
			shiftIDs[i] = shift.ID
		}

		weeklyStaffMap, err := h.container.ShiftWeeklyStaffRepository().GetByShiftIDs(r.Context(), shiftIDs)
		if err != nil {
			h.logger.Warn("Failed to get weekly staff configs", "error", err)
			// 继续返回班次列表，只是没有摘要
		} else {
			for _, shift := range result.Items {
				weeklyStaffs := weeklyStaffMap[shift.ID]
				// 转换为 WeekdayStaff 格式用于生成摘要
				weekdayStaffs := make([]model.WeekdayStaff, 0, 7)
				weekdayMap := make(map[int]int)
				for _, ws := range weeklyStaffs {
					weekdayMap[ws.Weekday] = ws.StaffCount
				}
				for i := 0; i < 7; i++ {
					count := 0
					if c, ok := weekdayMap[i]; ok {
						count = c
					}
					weekdayStaffs = append(weekdayStaffs, model.WeekdayStaff{
						Weekday:    i,
						StaffCount: count,
					})
				}
				shift.WeeklyStaffSummary = model.FormatWeeklyStaffSummary(weekdayStaffs)
			}
		}
	}

	RespondPage(w, result.Total, req.Page, req.Size, result.Items)
}

// AssignShift 分配班次给员工
func (h *HTTPHandler) AssignShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var req AssignShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	assignment := &model.ShiftAssignment{
		OrgID:      orgID,
		ShiftID:    shiftID,
		EmployeeID: req.EmployeeID,
		Date:       req.Date,
	}

	if err := h.container.GetShiftService().AssignShift(r.Context(), assignment); err != nil {
		h.logger.Error("Failed to assign shift", "error", err)
		RespondInternalError(w, "Failed to assign shift")
		return
	}

	RespondCreated(w, map[string]string{"message": "Shift assigned successfully"})
}

// GetEmployeeShifts 获取员工的班次分配
func (h *HTTPHandler) GetEmployeeShifts(w http.ResponseWriter, r *http.Request) {
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

	shifts, err := h.container.GetShiftService().GetEmployeeShifts(r.Context(), orgID, employeeID, startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get employee shifts", "error", err)
		RespondInternalError(w, "Failed to get employee shifts")
		return
	}

	RespondSuccess(w, shifts)
}

// SetShiftGroupsRequest 设置班次关联分组请求
type SetShiftGroupsRequest struct {
	GroupIDs []string `json:"groupIds" validate:"required"`
}

// AddGroupToShiftRequest 添加分组到班次请求
type AddGroupToShiftRequest struct {
	Priority int `json:"priority"`
}

// GetShiftGroups 获取班次关联的所有分组
func (h *HTTPHandler) GetShiftGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]

	shiftGroups, err := h.container.GetShiftService().GetShiftGroups(r.Context(), shiftID)
	if err != nil {
		h.logger.Error("Failed to get shift groups", "error", err)
		RespondInternalError(w, "Failed to get shift groups")
		return
	}

	RespondSuccess(w, shiftGroups)
}

// SetShiftGroups 批量设置班次的关联分组
func (h *HTTPHandler) SetShiftGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]

	var req SetShiftGroupsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	if err := h.container.GetShiftService().SetShiftGroups(r.Context(), shiftID, req.GroupIDs); err != nil {
		h.logger.Error("Failed to set shift groups", "error", err)
		RespondInternalError(w, "Failed to set shift groups")
		return
	}

	RespondSuccess(w, map[string]interface{}{
		"message":    "Shift groups set successfully",
		"shiftId":    shiftID,
		"groupCount": len(req.GroupIDs),
	})
}

// AddGroupToShift 为班次添加关联分组
func (h *HTTPHandler) AddGroupToShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]
	groupID := vars["groupId"]

	var req AddGroupToShiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// 如果body为空或解析失败，使用默认优先级0
		req.Priority = 0
	}

	if err := h.container.GetShiftService().AddGroupToShift(r.Context(), shiftID, groupID, req.Priority); err != nil {
		h.logger.Error("Failed to add group to shift", "error", err)
		RespondInternalError(w, "Failed to add group to shift")
		return
	}

	RespondSuccess(w, map[string]interface{}{
		"message": "Group added to shift successfully",
		"shiftId": shiftID,
		"groupId": groupID,
	})
}

// RemoveGroupFromShift 从班次移除关联分组
func (h *HTTPHandler) RemoveGroupFromShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]
	groupID := vars["groupId"]

	if err := h.container.GetShiftService().RemoveGroupFromShift(r.Context(), shiftID, groupID); err != nil {
		h.logger.Error("Failed to remove group from shift", "error", err)
		RespondInternalError(w, "Failed to remove group from shift")
		return
	}

	RespondSuccess(w, map[string]interface{}{
		"message": "Group removed from shift successfully",
		"shiftId": shiftID,
		"groupId": groupID,
	})
}

// ToggleShiftStatus 切换班次启用/禁用状态
func (h *HTTPHandler) ToggleShiftStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]
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

	if err := h.container.GetShiftService().ToggleShiftStatus(r.Context(), orgID, shiftID, req.IsActive); err != nil {
		h.logger.Error("Failed to toggle shift status", "error", err)
		RespondInternalError(w, "Failed to toggle shift status")
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Shift status updated successfully",
		"shiftId":  shiftID,
		"isActive": req.IsActive,
	})
}

// GetShiftGroupMembers 获取班次关联的所有分组的成员
func (h *HTTPHandler) GetShiftGroupMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shiftID := vars["id"]
	if shiftID == "" {
		RespondBadRequest(w, "Shift ID is required")
		return
	}

	members, err := h.container.GetShiftService().GetShiftGroupMembers(r.Context(), shiftID)
	if err != nil {
		h.logger.Error("Failed to get shift group members", "shiftId", shiftID, "error", err)
		RespondInternalError(w, "Failed to get shift group members")
		return
	}

	RespondSuccess(w, members)
}
