package domain

type IClient interface {
	IDepartmentService
	IEmployeeService
	IGroupService
	ILeaveService
	IRuleService
	ISchedulingService
	IShiftService
	ISystemSettingService
}
