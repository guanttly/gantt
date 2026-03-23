package repository

import (
	"context"
	"jusha/gantt/mcp/rostering/domain/model"
)

type IManagementRepository interface {
	// 通用HTTP请求方法
	DoRequest(ctx context.Context, method, path string, body any) (error, *model.APIResponse)
	DoRequestWithData(ctx context.Context, method, path string, body any, data any) error

	// 员工相关
	GetEmployee(ctx context.Context, id string) (*model.Employee, error)
	ListEmployees(ctx context.Context, req *model.ListEmployeesRequest) (*model.PageData[*model.Employee], error)
	CreateEmployee(ctx context.Context, req *model.CreateEmployeeRequest) (*model.Employee, error)
	UpdateEmployee(ctx context.Context, id string, req *model.UpdateEmployeeRequest) (*model.Employee, error)
	DeleteEmployee(ctx context.Context, id string) error

	// 班次相关
	GetShift(ctx context.Context, id string) (*model.Shift, error)
	ListShifts(ctx context.Context, req *model.ListShiftsRequest) (*model.PageData[*model.Shift], error)
	CreateShift(ctx context.Context, req *model.CreateShiftRequest) (*model.Shift, error)
	UpdateShift(ctx context.Context, id string, req *model.UpdateShiftRequest) (*model.Shift, error)
	DeleteShift(ctx context.Context, id string) error
	SetShiftGroups(ctx context.Context, shiftID string, req *model.SetShiftGroupsRequest) error
	AddShiftGroup(ctx context.Context, shiftID string, req *model.AddShiftGroupRequest) error
	RemoveShiftGroup(ctx context.Context, shiftID string, groupID string) error
	GetShiftGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error)
	GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error)
	ToggleShiftStatus(ctx context.Context, id string, status string) error
	GetShiftWeeklyStaff(ctx context.Context, orgID, shiftID string) (*model.ShiftWeeklyStaffConfig, error)
	SetShiftWeeklyStaff(ctx context.Context, orgID, shiftID string, req *model.SetShiftWeeklyStaffRequest) error
	CalculateStaffing(ctx context.Context, orgID, shiftID string) (*model.StaffingCalculationPreview, error)

	// 固定人员配置相关
	ListFixedAssignmentsByShift(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error)
	CalculateFixedSchedule(ctx context.Context, shiftID string, startDate, endDate string) (map[string][]string, error)
	CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error)

	// 批量查询规则
	GetRulesForEmployees(ctx context.Context, orgID string, employeeIDs []string) (map[string][]*model.Rule, error)
	GetRulesForShifts(ctx context.Context, orgID string, shiftIDs []string) (map[string][]*model.Rule, error)
	GetRulesForGroups(ctx context.Context, orgID string, groupIDs []string) (map[string][]*model.Rule, error)

	// 规则相关
	GetRule(ctx context.Context, id string) (*model.Rule, error)
	ListRules(ctx context.Context, req *model.ListRulesRequest) (*model.PageData[*model.Rule], error)
	CreateRule(ctx context.Context, req *model.CreateRuleRequest) (*model.Rule, error)
	UpdateRule(ctx context.Context, id string, req *model.UpdateRuleRequest) (*model.Rule, error)
	DeleteRule(ctx context.Context, id string) error
	GetRulesForEmployee(ctx context.Context, orgID, employeeID string) ([]*model.Rule, error)
	GetRulesForGroup(ctx context.Context, orgID, groupID string) ([]*model.Rule, error)
	GetRulesForShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error)

	// V4.1 规则班次关系相关
	CreateRuleWithRelations(ctx context.Context, req *model.CreateRuleWithRelationsRequest) (*model.Rule, error)
	UpdateRuleWithRelations(ctx context.Context, id string, req *model.UpdateRuleWithRelationsRequest) (*model.Rule, error)
	GetRulesBySubjectShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error)
	GetRulesByObjectShift(ctx context.Context, orgID, shiftID string) ([]*model.Rule, error)

	// 分组相关
	GetGroup(ctx context.Context, id string) (*model.Group, error)
	ListGroups(ctx context.Context, req *model.ListGroupsRequest) (*model.PageData[*model.Group], error)
	CreateGroup(ctx context.Context, req *model.CreateGroupRequest) (*model.Group, error)
	UpdateGroup(ctx context.Context, id string, req *model.UpdateGroupRequest) (*model.Group, error)
	DeleteGroup(ctx context.Context, id string) error

	// 系统设置相关
	GetSystemSetting(ctx context.Context, orgID, key string) (string, error)
	SetSystemSetting(ctx context.Context, orgID, key, value string) error
}
