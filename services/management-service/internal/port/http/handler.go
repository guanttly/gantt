package http

import (
	"net/http"

	"jusha/gantt/service/management/internal/wiring"
	"jusha/mcp/pkg/logging"

	"github.com/gorilla/mux"
)

// HTTPHandler HTTP 请求处理器
type HTTPHandler struct {
	container *wiring.Container
	logger    logging.ILogger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(container *wiring.Container, logger logging.ILogger) http.Handler {
	h := &HTTPHandler{
		container: container,
		logger:    logger.With("component", "HTTPHandler"),
	}

	router := mux.NewRouter()

	// 健康检查
	router.HandleFunc("/health", h.Health).Methods("GET")

	// 部门管理 API
	departmentRouter := router.PathPrefix("/api/v1/departments").Subrouter()
	departmentRouter.HandleFunc("", h.ListDepartments).Methods("GET")
	departmentRouter.HandleFunc("", h.CreateDepartment).Methods("POST")
	departmentRouter.HandleFunc("/tree", h.GetDepartmentTree).Methods("GET")
	departmentRouter.HandleFunc("/active", h.GetActiveDepartments).Methods("GET")
	departmentRouter.HandleFunc("/{id}", h.GetDepartment).Methods("GET")
	departmentRouter.HandleFunc("/{id}", h.UpdateDepartment).Methods("PUT")
	departmentRouter.HandleFunc("/{id}", h.DeleteDepartment).Methods("DELETE")

	// 员工管理 API
	employeeRouter := router.PathPrefix("/api/v1/employees").Subrouter()
	employeeRouter.HandleFunc("", h.ListEmployees).Methods("GET")
	employeeRouter.HandleFunc("", h.CreateEmployee).Methods("POST")
	employeeRouter.HandleFunc("/simple", h.ListSimpleEmployees).Methods("GET") // 简单查询端点，不包含分组信息
	employeeRouter.HandleFunc("/{id}", h.GetEmployee).Methods("GET")
	employeeRouter.HandleFunc("/{id}", h.UpdateEmployee).Methods("PUT")
	employeeRouter.HandleFunc("/{id}", h.DeleteEmployee).Methods("DELETE")
	employeeRouter.HandleFunc("/{id}/status", h.UpdateEmployeeStatus).Methods("PATCH")

	// 分组管理 API
	groupRouter := router.PathPrefix("/api/v1/groups").Subrouter()
	groupRouter.HandleFunc("", h.ListGroups).Methods("GET")
	groupRouter.HandleFunc("", h.CreateGroup).Methods("POST")
	groupRouter.HandleFunc("/{id}", h.GetGroup).Methods("GET")
	groupRouter.HandleFunc("/{id}", h.UpdateGroup).Methods("PUT")
	groupRouter.HandleFunc("/{id}", h.DeleteGroup).Methods("DELETE")
	groupRouter.HandleFunc("/{id}/members", h.GetGroupMembers).Methods("GET")
	groupRouter.HandleFunc("/{id}/members", h.AddGroupMember).Methods("POST")
	groupRouter.HandleFunc("/{id}/members/batch", h.BatchAddGroupMembers).Methods("POST")
	groupRouter.HandleFunc("/{id}/members/{employeeId}", h.RemoveGroupMember).Methods("DELETE")
	groupRouter.HandleFunc("/{id}/members/{employeeId}/role", h.UpdateGroupMemberRole).Methods("PATCH")

	// 班次管理 API
	shiftRouter := router.PathPrefix("/api/v1/shifts").Subrouter()
	shiftRouter.HandleFunc("", h.ListShifts).Methods("GET")
	shiftRouter.HandleFunc("", h.CreateShift).Methods("POST")
	shiftRouter.HandleFunc("/{id}", h.GetShift).Methods("GET")
	shiftRouter.HandleFunc("/{id}", h.UpdateShift).Methods("PUT")
	shiftRouter.HandleFunc("/{id}", h.DeleteShift).Methods("DELETE")
	shiftRouter.HandleFunc("/{id}/status", h.ToggleShiftStatus).Methods("PATCH")
	shiftRouter.HandleFunc("/{id}/assign", h.AssignShift).Methods("POST")
	shiftRouter.HandleFunc("/assignments", h.GetEmployeeShifts).Methods("GET")
	// 班次-分组关联 API
	shiftRouter.HandleFunc("/{id}/groups", h.GetShiftGroups).Methods("GET")
	shiftRouter.HandleFunc("/{id}/groups", h.SetShiftGroups).Methods("PUT")
	shiftRouter.HandleFunc("/{id}/groups/{groupId}", h.AddGroupToShift).Methods("POST")
	shiftRouter.HandleFunc("/{id}/groups/{groupId}", h.RemoveGroupFromShift).Methods("DELETE")
	shiftRouter.HandleFunc("/{id}/members", h.GetShiftGroupMembers).Methods("GET")
	// 班次固定人员配置 API
	shiftRouter.HandleFunc("/{id}/fixed-assignments", h.ListFixedAssignmentsByShift).Methods("GET")
	shiftRouter.HandleFunc("/{id}/fixed-assignments", h.BatchCreateFixedAssignments).Methods("POST")
	shiftRouter.HandleFunc("/{id}/fixed-assignments/{assignmentId}", h.DeleteFixedAssignment).Methods("DELETE")
	shiftRouter.HandleFunc("/{id}/fixed-assignments/calculate", h.CalculateFixedSchedule).Methods("POST")

	// 批量计算固定排班 API
	fixedAssignmentRouter := router.PathPrefix("/api/v1/fixed-assignments").Subrouter()
	fixedAssignmentRouter.HandleFunc("/calculate-multiple", h.CalculateMultipleFixedSchedules).Methods("POST")

	// 排班管理 API - 独立的排班服务
	schedulingRouter := router.PathPrefix("/api/v1/scheduling").Subrouter()
	schedulingRouter.HandleFunc("/assignments/batch", h.BatchAssignSchedule).Methods("POST")
	schedulingRouter.HandleFunc("/assignments", h.GetScheduleByDateRange).Methods("GET")
	schedulingRouter.HandleFunc("/assignments/employee", h.GetEmployeeSchedule).Methods("GET")
	schedulingRouter.HandleFunc("/assignments", h.DeleteScheduleAssignment).Methods("DELETE")
	schedulingRouter.HandleFunc("/assignments/batch/delete", h.BatchDeleteSchedule).Methods("POST")
	schedulingRouter.HandleFunc("/summary", h.GetScheduleSummary).Methods("GET")

	// 请假管理 API
	leaveRouter := router.PathPrefix("/api/v1/leaves").Subrouter()
	leaveRouter.HandleFunc("", h.ListLeaves).Methods("GET")
	leaveRouter.HandleFunc("", h.CreateLeave).Methods("POST")
	leaveRouter.HandleFunc("/{id}", h.GetLeave).Methods("GET")
	leaveRouter.HandleFunc("/{id}", h.UpdateLeave).Methods("PUT")
	leaveRouter.HandleFunc("/{id}", h.DeleteLeave).Methods("DELETE")
	leaveRouter.HandleFunc("/balance", h.GetLeaveBalance).Methods("GET")
	leaveRouter.HandleFunc("/balance/initialize", h.InitializeLeaveBalance).Methods("POST")

	// 排班规则管理 API
	ruleRouter := router.PathPrefix("/api/v1/scheduling-rules").Subrouter()
	ruleRouter.HandleFunc("", h.ListRules).Methods("GET")
	ruleRouter.HandleFunc("", h.CreateRule).Methods("POST")
	ruleRouter.HandleFunc("/{id}", h.GetRule).Methods("GET")
	ruleRouter.HandleFunc("/{id}", h.UpdateRule).Methods("PUT")
	ruleRouter.HandleFunc("/{id}", h.DeleteRule).Methods("DELETE")
	ruleRouter.HandleFunc("/{id}/status", h.ToggleSchedulingRuleStatus).Methods("PATCH")

	// V4新增API：规则解析和批量保存
	ruleRouter.HandleFunc("/parse", h.ParseRule).Methods("POST")
	ruleRouter.HandleFunc("/batch-parse", h.BatchParse).Methods("POST") // 批量解析
	ruleRouter.HandleFunc("/batch", h.BatchSaveRules).Methods("POST")
	ruleRouter.HandleFunc("/organize", h.OrganizeRules).Methods("POST")
	// V4迁移API
	ruleRouter.HandleFunc("/migration/preview", h.PreviewMigration).Methods("GET")
	ruleRouter.HandleFunc("/migration/execute", h.ExecuteMigration).Methods("POST")
	ruleRouter.HandleFunc("/migration/rollback", h.RollbackMigration).Methods("POST")
	ruleRouter.HandleFunc("/migration/{id}/status", h.GetMigrationStatus).Methods("GET")
	// V4依赖/冲突/统计API
	ruleRouter.HandleFunc("/dependencies", h.GetRuleDependencies).Methods("GET")
	ruleRouter.HandleFunc("/dependencies", h.CreateRuleDependency).Methods("POST")
	ruleRouter.HandleFunc("/dependencies/{id}", h.DeleteRuleDependency).Methods("DELETE")
	ruleRouter.HandleFunc("/conflicts", h.GetRuleConflicts).Methods("GET")
	ruleRouter.HandleFunc("/conflicts", h.CreateRuleConflict).Methods("POST")
	ruleRouter.HandleFunc("/conflicts/{id}", h.DeleteRuleConflict).Methods("DELETE")
	ruleRouter.HandleFunc("/statistics", h.GetRuleStatistics).Methods("GET")

	// 跨资源查询 - 获取员工/班次相关的规则
	router.HandleFunc("/api/v1/employees/{employeeId}/scheduling-rules", h.GetRulesForEmployee).Methods("GET")
	router.HandleFunc("/api/v1/shifts/{shiftId}/scheduling-rules", h.GetRulesForShift).Methods("GET")
	router.HandleFunc("/api/v1/groups/{groupId}/scheduling-rules", h.GetRulesForGroup).Methods("GET")
	// 批量查询规则接口
	router.HandleFunc("/api/v1/scheduling-rules/batch/employees", h.GetRulesForEmployees).Methods("POST")
	router.HandleFunc("/api/v1/scheduling-rules/batch/shifts", h.GetRulesForShifts).Methods("POST")
	router.HandleFunc("/api/v1/scheduling-rules/batch/groups", h.GetRulesForGroups).Methods("POST")

	// 机房管理 API
	modalityRoomRouter := router.PathPrefix("/api/v1/modality-rooms").Subrouter()
	modalityRoomRouter.HandleFunc("", h.ListModalityRooms).Methods("GET")
	modalityRoomRouter.HandleFunc("", h.CreateModalityRoom).Methods("POST")
	modalityRoomRouter.HandleFunc("/active", h.GetActiveModalityRooms).Methods("GET")
	modalityRoomRouter.HandleFunc("/{id}", h.GetModalityRoom).Methods("GET")
	modalityRoomRouter.HandleFunc("/{id}", h.UpdateModalityRoom).Methods("PUT")
	modalityRoomRouter.HandleFunc("/{id}", h.DeleteModalityRoom).Methods("DELETE")
	modalityRoomRouter.HandleFunc("/{id}/status", h.ToggleModalityRoomStatus).Methods("PATCH")
	// 机房周检查量配置 API（作为机房的子资源）
	modalityRoomRouter.HandleFunc("/{id}/weekly-volumes", h.GetModalityRoomWeeklyVolumes).Methods("GET")
	modalityRoomRouter.HandleFunc("/{id}/weekly-volumes", h.SaveModalityRoomWeeklyVolumes).Methods("PUT")
	modalityRoomRouter.HandleFunc("/{id}/weekly-volumes", h.DeleteModalityRoomWeeklyVolumes).Methods("DELETE")

	// 时间段管理 API
	timePeriodRouter := router.PathPrefix("/api/v1/time-periods").Subrouter()
	timePeriodRouter.HandleFunc("", h.ListTimePeriods).Methods("GET")
	timePeriodRouter.HandleFunc("", h.CreateTimePeriod).Methods("POST")
	timePeriodRouter.HandleFunc("/active", h.GetActiveTimePeriods).Methods("GET")
	timePeriodRouter.HandleFunc("/{id}", h.GetTimePeriod).Methods("GET")
	timePeriodRouter.HandleFunc("/{id}", h.UpdateTimePeriod).Methods("PUT")
	timePeriodRouter.HandleFunc("/{id}", h.DeleteTimePeriod).Methods("DELETE")
	timePeriodRouter.HandleFunc("/{id}/status", h.ToggleTimePeriodStatus).Methods("PATCH")

	// 检查类型管理 API
	scanTypeRouter := router.PathPrefix("/api/v1/scan-types").Subrouter()
	scanTypeRouter.HandleFunc("", h.ListScanTypes).Methods("GET")
	scanTypeRouter.HandleFunc("", h.CreateScanType).Methods("POST")
	scanTypeRouter.HandleFunc("/active", h.GetActiveScanTypes).Methods("GET")
	scanTypeRouter.HandleFunc("/{id}", h.GetScanType).Methods("GET")
	scanTypeRouter.HandleFunc("/{id}", h.UpdateScanType).Methods("PUT")
	scanTypeRouter.HandleFunc("/{id}", h.DeleteScanType).Methods("DELETE")
	scanTypeRouter.HandleFunc("/{id}/status", h.ToggleScanTypeStatus).Methods("PATCH")

	// 排班人数计算 API
	staffingRouter := router.PathPrefix("/api/v1/staffing").Subrouter()
	staffingRouter.HandleFunc("/calculate", h.CalculateStaffing).Methods("POST")
	staffingRouter.HandleFunc("/apply", h.ApplyStaffing).Methods("POST")
	staffingRouter.HandleFunc("/rules", h.ListStaffingRules).Methods("GET")
	staffingRouter.HandleFunc("/rules", h.CreateStaffingRule).Methods("POST")
	staffingRouter.HandleFunc("/rules/{id}", h.GetStaffingRule).Methods("GET")
	staffingRouter.HandleFunc("/rules/{id}", h.UpdateStaffingRule).Methods("PUT")
	staffingRouter.HandleFunc("/rules/{id}", h.DeleteStaffingRule).Methods("DELETE")

	// 班次周默认人数配置 API
	shiftRouter.HandleFunc("/{id}/weekly-staff", h.GetWeeklyStaffConfig).Methods("GET")
	shiftRouter.HandleFunc("/{id}/weekly-staff", h.SetWeeklyStaffConfig).Methods("PUT")
	shiftRouter.HandleFunc("/weekly-staff/batch", h.BatchGetWeeklyStaffConfig).Methods("GET")

	// 系统设置 API
	systemSettingRouter := router.PathPrefix("/api/v1/system-settings").Subrouter()
	systemSettingRouter.HandleFunc("", h.GetAllSettings).Methods("GET")
	systemSettingRouter.HandleFunc("/{key}", h.GetSetting).Methods("GET")
	systemSettingRouter.HandleFunc("/{key}", h.SetSetting).Methods("PUT")
	systemSettingRouter.HandleFunc("/{key}", h.DeleteSetting).Methods("DELETE")

	// 对话记录管理 API
	conversationRouter := router.PathPrefix("/api/v1/conversations").Subrouter()
	// 查询排班对话记录（支持按日期范围、状态等查询）
	conversationRouter.HandleFunc("/schedules", h.ListScheduleConversations).Methods("GET")
	// 创建或更新对话记录（由 MCP 工具调用）
	conversationRouter.HandleFunc("/create-or-update", h.CreateOrUpdateConversation).Methods("POST")

	// 用户偏好管理 API
	userRouter := router.PathPrefix("/api/v1/users/{userId}/preferences").Subrouter()
	userRouter.HandleFunc("/workflow-version", h.GetUserWorkflowVersion).Methods("GET")
	userRouter.HandleFunc("/workflow-version", h.SetUserWorkflowVersion).Methods("PUT")

	h.logger.Info("HTTP routes registered successfully")
	return router
}

// Health 健康检查
func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	RespondSuccess(w, map[string]string{
		"status":  "healthy",
		"service": "management-service",
	})
}
