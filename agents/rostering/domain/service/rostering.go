package service

import (
	"context"
	d_model "jusha/agent/rostering/domain/model"
)

// IDataService 数据服务领域接口
// 统一封装对data-server的MCP调用，提供排班业务所需的数据访问能力
type IRosteringService interface {
	// 排班数据服务
	QuerySchedules(ctx context.Context, filter d_model.ScheduleQueryFilter) (*d_model.ScheduleQueryResult, error)
	UpsertSchedule(ctx context.Context, req d_model.ScheduleUpsertRequest) (*d_model.ScheduleEntry, error)
	BatchUpsertSchedules(ctx context.Context, batch d_model.ScheduleBatch) (*d_model.BatchUpsertResult, error)
	DeleteSchedule(ctx context.Context, userID, date string) error

	// 人员数据服务
	GetStaffProfiles(ctx context.Context, orgID, department, modality string) ([]*d_model.Staff, error)
	ListStaff(ctx context.Context, filter d_model.StaffListFilter) (*d_model.StaffListResult, error)
	SearchStaff(ctx context.Context, filter d_model.StaffSearchFilter) (*d_model.StaffListResult, error)
	CheckStaffExists(ctx context.Context, orgID, name string) (bool, error)
	GetStaff(ctx context.Context, userID string) (*d_model.Staff, error)
	CreateStaff(ctx context.Context, req d_model.StaffCreateRequest) (string, error)

	// 配置数据服务
	ListGroups(ctx context.Context, orgID string) ([]*d_model.Group, error)
	CreateGroup(ctx context.Context, req d_model.CreateGroupRequest) (string, error)
	UpdateGroup(ctx context.Context, req d_model.UpdateGroupRequest) error
	AssignGroupMembers(ctx context.Context, req d_model.AddGroupMemberRequest) error
	ListShifts(ctx context.Context, orgID string, groupID string) ([]*d_model.Shift, error)
	GetGroupMembers(ctx context.Context, groupID string) ([]*d_model.Employee, error)

	// 请假数据服务
	CreateLeave(ctx context.Context, req d_model.CreateLeaveRequest) (string, error)
	GetLeaveRecords(ctx context.Context, orgID, staffID string, startDate, endDate string) ([]*d_model.LeaveRecord, error)
	// BatchGetLeaveRecords 批量获取请假记录（所有员工，按员工ID分组）
	// 优化：一次查询获取所有员工的请假记录，避免多次 RPC 调用
	BatchGetLeaveRecords(ctx context.Context, orgID string, startDate, endDate string) (map[string][]*d_model.LeaveRecord, error)

	// 排班范围数据服务（Shift Category Group）
	// 增强方法：支持向上递归查找父分类
	GetGroupsByShiftID(ctx context.Context, orgID, shiftID string, withFallback bool) ([]*d_model.Group, error)

	// 规则数据服务
	ListRules(ctx context.Context, req d_model.ListRulesRequest) ([]*d_model.Rule, error)
	GetRulesForEmployee(ctx context.Context, orgID, employeeID, date string) ([]*d_model.Rule, error)
	GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*d_model.Rule, error)
	GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*d_model.Rule, error)
	// 批量查询规则
	GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*d_model.Rule, error)
	GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*d_model.Rule, error)
	GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*d_model.Rule, error)

	// 班次分组服务
	AddShiftGroup(ctx context.Context, shiftID string, groupID string, priority int) error
	RemoveShiftGroup(ctx context.Context, shiftID string, groupID string) error
	GetShiftGroups(ctx context.Context, shiftID string) ([]*d_model.ShiftGroup, error)
	GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*d_model.Employee, error)

	// 班次人数配置服务
	GetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string) (*d_model.ShiftWeeklyStaffConfig, error)
	SetWeeklyStaffConfig(ctx context.Context, orgID, shiftID string, config []d_model.WeekdayStaffConfig) error
	CalculateStaffing(ctx context.Context, orgID, shiftID string) (*d_model.StaffingCalculationPreview, error)

	// 固定人员配置服务
	CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error)

	// 系统设置服务
	GetSystemSetting(ctx context.Context, orgID, key string) (string, error)
	SetSystemSetting(ctx context.Context, orgID, key, value string) error

	// 用户偏好服务
	GetUserWorkflowVersion(ctx context.Context, orgID, userID string) (string, error)
	SetUserWorkflowVersion(ctx context.Context, orgID, userID, version string) error

	// 数据库管理服务
	EnsureScheduleDBConfigured() error
	AutoMigrateStaffing(ctx context.Context) error
}
