package tool

type ToolName string

func (t ToolName) String() string {
	return string(t)
}

// Employee 员工管理工具
const (
	ToolEmployeeCreate ToolName = "rostering.employee.create"
	ToolEmployeeUpdate ToolName = "rostering.employee.update"
	ToolEmployeeList   ToolName = "rostering.employee.list"
	ToolEmployeeGet    ToolName = "rostering.employee.get"
	ToolEmployeeDelete ToolName = "rostering.employee.delete"
)

// Department 部门管理工具
const (
	ToolDepartmentCreate ToolName = "rostering.department.create"
	ToolDepartmentUpdate ToolName = "rostering.department.update"
	ToolDepartmentList   ToolName = "rostering.department.list"
)

// Group 分组管理工具
const (
	ToolGroupCreate       ToolName = "rostering.group.create"
	ToolGroupUpdate       ToolName = "rostering.group.update"
	ToolGroupList         ToolName = "rostering.group.list"
	ToolGroupGet          ToolName = "rostering.group.get"
	ToolGroupDelete       ToolName = "rostering.group.delete"
	ToolGroupAddMember    ToolName = "rostering.group.add_member"
	ToolGroupRemoveMember ToolName = "rostering.group.remove_member"
	ToolGroupGetMembers   ToolName = "rostering.group.get_members"
)

// Shift 班次管理工具
const (
	ToolShiftCreate            ToolName = "rostering.shift.create"
	ToolShiftUpdate            ToolName = "rostering.shift.update"
	ToolShiftList              ToolName = "rostering.shift.list"
	ToolShiftGet               ToolName = "rostering.shift.get"
	ToolShiftDelete            ToolName = "rostering.shift.delete"
	ToolShiftToggleStatus      ToolName = "rostering.shift.toggle_status"
	ToolShiftSetGroups         ToolName = "rostering.shift.set_groups"
	ToolShiftAddGroup          ToolName = "rostering.shift.add_group"
	ToolShiftRemoveGroup       ToolName = "rostering.shift.remove_group"
	ToolShiftGetGroups         ToolName = "rostering.shift.get_groups"
	ToolShiftGetGroupMembers   ToolName = "rostering.shift.get_group_members"
	ToolShiftGetWeeklyStaff    ToolName = "rostering.shift.get_weekly_staff"
	ToolShiftSetWeeklyStaff    ToolName = "rostering.shift.set_weekly_staff"
	ToolShiftCalculateStaffing ToolName = "rostering.shift.calculate_staffing"
)

// Rule 规则管理工具
const (
	ToolRuleCreate          ToolName = "rostering.rule.create"
	ToolRuleUpdate          ToolName = "rostering.rule.update"
	ToolRuleList            ToolName = "rostering.rule.list"
	ToolRuleGet             ToolName = "rostering.rule.get"
	ToolRuleDelete          ToolName = "rostering.rule.delete"
	ToolRuleAddAssociations ToolName = "rostering.rule.add_associations"
	ToolRuleGetForEmployee  ToolName = "rostering.rule.get_for_employee"
	ToolRuleGetForGroup     ToolName = "rostering.rule.get_for_group"
	ToolRuleGetForShift     ToolName = "rostering.rule.get_for_shift"
	// 批量查询规则工具
	ToolRuleGetForEmployees ToolName = "rostering.rule.get_for_employees"
	ToolRuleGetForShifts    ToolName = "rostering.rule.get_for_shifts"
	ToolRuleGetForGroups    ToolName = "rostering.rule.get_for_groups"
)

// Leave 请假管理工具
const (
	ToolLeaveCreate     ToolName = "rostering.leave.create"
	ToolLeaveUpdate     ToolName = "rostering.leave.update"
	ToolLeaveList       ToolName = "rostering.leave.list"
	ToolLeaveGet        ToolName = "rostering.leave.get"
	ToolLeaveDelete     ToolName = "rostering.leave.delete"
	ToolLeaveGetBalance ToolName = "rostering.leave.get_balance"
)

// SystemSetting 系统设置工具
const (
	ToolSystemSettingGet ToolName = "rostering.system_setting.get"
	ToolSystemSettingSet ToolName = "rostering.system_setting.set"
)

// Scheduling 排班管理工具
const (
	ToolSchedulingBatchAssign    ToolName = "rostering.scheduling.batch_assign"
	ToolSchedulingGetByDateRange ToolName = "rostering.scheduling.get_by_date_range"
	ToolSchedulingGetSummary     ToolName = "rostering.scheduling.get_summary"
	ToolSchedulingDelete         ToolName = "rostering.scheduling.delete"
)
