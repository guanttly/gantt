package service

import (
	"jusha/gantt/mcp/rostering/config"
	"jusha/gantt/mcp/rostering/domain/repository"
	"jusha/gantt/mcp/rostering/domain/service"
	"jusha/mcp/pkg/logging"
)

type baseServiceProvider struct {
	logger logging.ILogger
	cfg    config.IRosteringConfigurator

	departmentService    service.IDepartmentService
	employeeService      service.IEmployeeService
	groupService         service.IGroupService
	shiftService         service.IShiftService
	schedulingService    service.ISchedulingService
	leaveService         service.ILeaveService
	ruleService          service.IRuleService
	fixedAssignmentService service.IFixedAssignmentService
	systemSettingService service.ISystemSettingService
}

func (s *baseServiceProvider) Department() service.IDepartmentService {
	return s.departmentService
}

func (s *baseServiceProvider) Employee() service.IEmployeeService {
	return s.employeeService
}

func (s *baseServiceProvider) Group() service.IGroupService {
	return s.groupService
}

func (s *baseServiceProvider) Shift() service.IShiftService {
	return s.shiftService
}

func (s *baseServiceProvider) Scheduling() service.ISchedulingService {
	return s.schedulingService
}

func (s *baseServiceProvider) Leave() service.ILeaveService {
	return s.leaveService
}

func (s *baseServiceProvider) Rule() service.IRuleService {
	return s.ruleService
}

func (s *baseServiceProvider) FixedAssignment() service.IFixedAssignmentService {
	return s.fixedAssignmentService
}

func (s *baseServiceProvider) SystemSetting() service.ISystemSettingService {
	return s.systemSettingService
}

type serviceBuilder struct {
	logger logging.ILogger
	cfg    config.IRosteringConfigurator

	repoProvider repository.IRepositoryProvider
}

func NewServiceProviderBuilder() service.IServiceBuilder {
	return &serviceBuilder{}
}

func (b *serviceBuilder) WithLogger(logger logging.ILogger) service.IServiceBuilder {
	b.logger = logger
	return b
}

func (b *serviceBuilder) WithConfigurator(cfg config.IRosteringConfigurator) service.IServiceBuilder {
	b.cfg = cfg
	return b
}

func (b *serviceBuilder) WithRepositoryProvider(repoProvider repository.IRepositoryProvider) service.IServiceBuilder {
	b.repoProvider = repoProvider
	return b
}

func (b *serviceBuilder) Build() service.IServiceProvider {
	managementRepo := b.repoProvider.GetManagementRepository()

	return &baseServiceProvider{
		logger:                b.logger,
		cfg:                   b.cfg,
		departmentService:     newDepartmentService(b.logger, b.cfg, managementRepo),
		employeeService:      newEmployeeService(b.logger, b.cfg, managementRepo),
		groupService:         newGroupService(b.logger, b.cfg, managementRepo),
		shiftService:         newShiftService(b.logger, b.cfg, managementRepo),
		schedulingService:    newSchedulingService(b.logger, b.cfg, managementRepo),
		leaveService:         newLeaveService(b.logger, b.cfg, managementRepo),
		ruleService:          newRuleService(b.logger, b.cfg, managementRepo),
		fixedAssignmentService: newFixedAssignmentService(b.logger, b.cfg, managementRepo),
		systemSettingService:  newSystemSettingService(b.logger, b.cfg, managementRepo),
	}
}
