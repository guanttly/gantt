package service

import (
	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/mcp/pkg/logging"
)

// IServiceProvider 聚合所有服务接口的提供者
type IServiceProvider interface {
	Employee() IEmployeeService
	Department() IDepartmentService
	Group() IGroupService
	Shift() IShiftService
	Scheduling() ISchedulingService
	Leave() ILeaveService
	Rule() IRuleService
	FixedAssignment() IFixedAssignmentService
	SystemSetting() ISystemSettingService
}

type IServiceBuilder interface {
	WithLogger(logger logging.ILogger) IServiceBuilder
	WithConfigurator(cfg config.IRosteringConfigurator) IServiceBuilder
	WithRepositoryProvider(repoProvider repository.IRepositoryProvider) IServiceBuilder
	Build() IServiceProvider
}
