package wiring

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"jusha/gantt/service/management/config"
	"jusha/gantt/service/management/domain/repository"
	"jusha/gantt/service/management/internal/entity"
	"jusha/mcp/pkg/adapter"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/discovery"
	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/mcp"

	domain_service "jusha/gantt/service/management/domain/service"
	repo_impl "jusha/gantt/service/management/internal/repository"
	service_impl "jusha/gantt/service/management/internal/service"
)

// Container 依赖注入容器
type Container struct {
	ctx    context.Context
	logger logging.ILogger
	config config.IManagementServiceConfigurator

	// 基础设施
	db *gorm.DB

	// 仓储层
	employeeRepo        repository.IEmployeeRepository
	groupRepo           repository.IGroupRepository
	shiftRepo           repository.IShiftRepository
	shiftAssignmentRepo repository.IShiftAssignmentRepository
	shiftGroupRepo      repository.IShiftGroupRepository // 班次-分组关联仓储
	schedulingRepo      repository.ISchedulingRepository // 排班仓储
	leaveRepo           repository.ILeaveRepository
	leaveBalanceRepo    repository.ILeaveBalanceRepository
	holidayRepo         repository.IHolidayRepository
	schedulingRuleRepo  repository.ISchedulingRuleRepository
	departmentRepo      repository.IDepartmentRepository

	// 机房与报告量相关仓储
	modalityRoomRepo             repository.IModalityRoomRepository
	timePeriodRepo               repository.ITimePeriodRepository
	scanTypeRepo                 repository.IScanTypeRepository
	modalityRoomWeeklyVolumeRepo repository.IModalityRoomWeeklyVolumeRepository
	shiftWeeklyStaffRepo         repository.IShiftWeeklyStaffRepository
	shiftStaffingRuleRepo        repository.IShiftStaffingRuleRepository
	shiftFixedAssignmentRepo     repository.IShiftFixedAssignmentRepository
	systemSettingRepo            repository.ISystemSettingRepository

	// V4.1新增：规则范围仓储
	ruleApplyScopeRepo repository.IRuleApplyScopeRepository

	// 服务层
	employeeService       domain_service.IEmployeeService
	groupService          domain_service.IGroupService
	shiftService          domain_service.IShiftService
	schedulingService     domain_service.ISchedulingService // 排班服务
	leaveService          domain_service.ILeaveService
	schedulingRuleService domain_service.ISchedulingRuleService
	departmentService     domain_service.IDepartmentService
	systemSettingService  domain_service.ISystemSettingService

	// 机房与报告量相关服务
	modalityRoomService             domain_service.IModalityRoomService
	timePeriodService               domain_service.ITimePeriodService
	scanTypeService                 domain_service.IScanTypeService
	modalityRoomWeeklyVolumeService domain_service.IModalityRoomWeeklyVolumeService
	staffingCalculationService      domain_service.IStaffingCalculationService
	shiftFixedAssignmentService     domain_service.IShiftFixedAssignmentService

	// MCP 基础设施（用于提供工具给其他服务调用）
	mcpManager mcp.MCPServerManager
	toolBus    mcp.IToolBus

	// 对话记录仓储
	conversationRepo repository.IConversationRepository

	// V4新增：依赖关系仓储
	ruleDependencyRepo  repository.IRuleDependencyRepository
	ruleConflictRepo    repository.IRuleConflictRepository
	shiftDependencyRepo repository.IShiftDependencyRepository

	// 对话记录服务
	conversationService domain_service.IConversationService

	// V4新增：规则解析服务
	ruleParserService domain_service.IRuleParserService

	// V4新增：规则组织服务
	ruleOrganizerService domain_service.IRuleOrganizerService

	// V4新增：规则迁移服务
	ruleMigrationService domain_service.IRuleMigrationService

	// V4新增：规则统计服务
	ruleStatisticsService domain_service.IRuleStatisticsService
}

// NewContainer 创建依赖注入容器
func NewContainer(ctx context.Context, logger logging.ILogger, cfg config.IManagementServiceConfigurator) (*Container, error) {
	if logger == nil {
		logger = slog.Default()
	}

	container := &Container{
		ctx:    ctx,
		logger: logger.With("component", "Container"),
		config: cfg,
	}

	// 初始化数据库
	if err := container.initDatabase(); err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// 初始化 MCP 基础设施（用于提供工具给其他服务调用）
	if err := container.initMCP(); err != nil {
		return nil, fmt.Errorf("init MCP: %w", err)
	}

	// 初始化仓储层
	container.initRepositories()

	// 初始化服务层
	container.initServices()

	container.logger.Info("Dependency injection container initialized successfully")
	return container, nil
}

// initDatabase 初始化数据库连接
func (c *Container) initDatabase() error {
	cfg := c.config.GetConfig()

	// 检查数据库配置
	if cfg.Config == nil || cfg.Config.Database == nil || cfg.Config.Database.MySQL == nil {
		return fmt.Errorf("database config is required")
	}

	mysqlCfg := cfg.Config.Database.MySQL
	if mysqlCfg.Host == "" {
		return fmt.Errorf("database host is required")
	}

	// 使用 adapter.NewMySQLConnection 初始化数据库连接
	dbCtx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	db, err := adapter.NewMySQLConnection(dbCtx, mysqlCfg, c.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	c.db = db
	c.logger.Info("Database connected successfully")

	// 自动迁移表结构
	if err := c.autoMigrate(); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	return nil
}

// initMCP 初始化 MCP 基础设施
func (c *Container) initMCP() error {
	c.logger.Info("Initializing MCP infrastructure")

	cfg := c.config.GetConfig()

	// 检查 MCP 配置
	if cfg.Config == nil || cfg.Config.MCP == nil {
		c.logger.Info("MCP config not found, skipping MCP initialization")
		return nil
	}

	// 1. 创建命名服务客户端
	if cfg.Config.Discovery != nil && cfg.Config.Discovery.Nacos != nil {
		namingClient, err := discovery.NewNacosNamingClient(cfg.Config.Discovery.Nacos, c.logger)
		if err != nil {
			c.logger.Warn("Failed to create naming client", "error", err)
			return nil // 不阻断启动
		}

		// 2. 创建 MCP Server Manager
		manager := mcp.NewDefaultMCPServerManager(
			namingClient,
			"mcp-server", // 服务组名
			c.logger,
		)

		// 启动 MCP 管理器
		if err := manager.Start(c.ctx); err != nil {
			c.logger.Warn("Failed to start MCP server manager", "error", err)
			return nil // 不阻断启动
		}

		c.mcpManager = manager
		c.logger.Info("MCP server manager started successfully")

		// 3. 创建工具总线
		toolBusConfig := mcp.DefaultToolBusConfig()

		// 从配置中读取超时参数
		if cfg.Config.MCP != nil && cfg.Config.MCP.ClientTimeout > 0 {
			toolBusConfig.DefaultTimeout = time.Duration(cfg.Config.MCP.ClientTimeout) * time.Second
		}

		c.toolBus = mcp.NewMCPToolBus(c.mcpManager, toolBusConfig, c.logger)
		c.logger.Info("MCP tool bus created successfully")
	} else {
		c.logger.Info("Discovery not configured, MCP manager will not be created")
	}

	return nil
}

// autoMigrate 自动迁移数据库表结构
func (c *Container) autoMigrate() error {
	return c.db.AutoMigrate(
		&entity.EmployeeEntity{},
		&entity.GroupEntity{},
		&entity.GroupMemberEntity{},
		&entity.ShiftEntity{},
		&entity.ShiftAssignmentEntity{},
		&entity.ShiftGroupEntity{}, // 班次-分组关联表
		&entity.LeaveRecordEntity{},
		&entity.LeaveBalanceEntity{},
		&entity.HolidayEntity{},
		&entity.SchedulingRuleEntity{},
		&entity.SchedulingRuleAssociationEntity{},
		&entity.DepartmentEntity{},
		// 机房与报告量相关表
		&entity.ModalityRoomEntity{},
		&entity.TimePeriodEntity{},
		&entity.ScanTypeEntity{},
		&entity.ModalityRoomWeeklyVolumeEntity{},
		&entity.ShiftWeeklyStaffEntity{},
		&entity.ShiftStaffingRuleEntity{},
		&entity.ShiftFixedAssignmentEntity{},
		&entity.SystemSettingEntity{},
		// 对话记录表
		&entity.ConversationEntity{},
		// V4新增表
		&entity.RuleDependencyEntity{},
		&entity.RuleConflictEntity{},
		&entity.ShiftDependencyEntity{},
		// V4.1新增表
		&entity.RuleApplyScopeEntity{},
	)
}

// initRepositories 初始化仓储层
func (c *Container) initRepositories() {
	c.employeeRepo = repo_impl.NewEmployeeRepository(c.db)
	c.groupRepo = repo_impl.NewGroupRepository(c.db)
	c.shiftRepo = repo_impl.NewShiftRepository(c.db)
	c.shiftAssignmentRepo = repo_impl.NewShiftAssignmentRepository(c.db)
	c.shiftGroupRepo = repo_impl.NewShiftGroupRepository(c.db) // 初始化班次-分组关联仓储
	c.schedulingRepo = repo_impl.NewSchedulingRepository(c.db) // 初始化排班仓储
	c.leaveRepo = repo_impl.NewLeaveRepository(c.db)
	c.leaveBalanceRepo = repo_impl.NewLeaveBalanceRepository(c.db)
	c.holidayRepo = repo_impl.NewHolidayRepository(c.db)
	c.schedulingRuleRepo = repo_impl.NewSchedulingRuleRepository(c.db)
	c.departmentRepo = repo_impl.NewDepartmentRepository(c.db)

	// 机房与报告量相关仓储
	c.modalityRoomRepo = repo_impl.NewModalityRoomRepository(c.db)
	c.timePeriodRepo = repo_impl.NewTimePeriodRepository(c.db)
	c.scanTypeRepo = repo_impl.NewScanTypeRepository(c.db)
	c.modalityRoomWeeklyVolumeRepo = repo_impl.NewModalityRoomWeeklyVolumeRepository(c.db)
	c.shiftWeeklyStaffRepo = repo_impl.NewShiftWeeklyStaffRepository(c.db)
	c.shiftStaffingRuleRepo = repo_impl.NewShiftStaffingRuleRepository(c.db)
	c.shiftFixedAssignmentRepo = repo_impl.NewShiftFixedAssignmentRepository(c.db)
	c.systemSettingRepo = repo_impl.NewSystemSettingRepository(c.db)
	c.conversationRepo = repo_impl.NewConversationRepository(c.db)

	// V4新增：依赖关系仓储
	c.ruleDependencyRepo = repo_impl.NewRuleDependencyRepository(c.db)
	c.ruleConflictRepo = repo_impl.NewRuleConflictRepository(c.db)
	c.shiftDependencyRepo = repo_impl.NewShiftDependencyRepository(c.db)

	// V4.1新增：规则范围仓储
	c.ruleApplyScopeRepo = repo_impl.NewRuleApplyScopeRepository(c.db)

	c.logger.Info("All repositories initialized successfully")
}

// initServices 初始化服务层
func (c *Container) initServices() {
	c.employeeService = service_impl.NewEmployeeService(c.employeeRepo, c.groupRepo, c.logger)
	c.groupService = service_impl.NewGroupService(c.groupRepo, c.employeeRepo, c.logger)
	c.shiftService = service_impl.NewShiftService(c.shiftRepo, c.shiftAssignmentRepo, c.shiftGroupRepo, c.groupRepo, c.employeeRepo, c.logger)
	c.schedulingService = service_impl.NewSchedulingService(c.schedulingRepo, c.shiftRepo, c.employeeRepo, c.logger)
	c.leaveService = service_impl.NewLeaveService(c.leaveRepo, c.leaveBalanceRepo, c.employeeRepo, c.holidayRepo, c.logger)
	c.schedulingRuleService = service_impl.NewSchedulingRuleService(c.schedulingRuleRepo, c.employeeRepo, c.shiftRepo, c.groupRepo, c.ruleApplyScopeRepo, c.logger)
	c.departmentService = service_impl.NewDepartmentService(c.departmentRepo, c.logger)

	// 机房与报告量相关服务
	c.modalityRoomService = service_impl.NewModalityRoomService(c.modalityRoomRepo, c.logger)
	c.timePeriodService = service_impl.NewTimePeriodService(c.timePeriodRepo, c.logger)
	c.scanTypeService = service_impl.NewScanTypeService(c.scanTypeRepo, c.logger)
	c.modalityRoomWeeklyVolumeService = service_impl.NewModalityRoomWeeklyVolumeService(
		c.modalityRoomWeeklyVolumeRepo,
		c.modalityRoomRepo,
		c.timePeriodRepo,
		c.scanTypeRepo,
		c.logger,
	)
	c.staffingCalculationService = service_impl.NewStaffingCalculationService(
		c.shiftRepo,
		c.shiftStaffingRuleRepo,
		c.shiftWeeklyStaffRepo,
		c.modalityRoomWeeklyVolumeRepo,
		c.modalityRoomRepo,
		c.timePeriodRepo,
		c.config,
		c.logger,
	)
	c.shiftFixedAssignmentService = service_impl.NewShiftFixedAssignmentService(
		c.shiftFixedAssignmentRepo,
		c.logger,
	)
	c.systemSettingService = service_impl.NewSystemSettingService(c.systemSettingRepo, c.logger)

	// 对话记录服务
	c.conversationService = service_impl.NewConversationService(c.conversationRepo, c.logger)
	c.logger.Info("ConversationService created successfully")

	// V4新增：规则解析服务
	// 初始化AI工厂
	aiFactory := ai.NewAIModelFactory(c.ctx, c.config, c.logger)
	c.ruleParserService = service_impl.NewRuleParserService(
		c.logger,
		aiFactory,
		c.schedulingRuleRepo,
		c.employeeRepo,
		c.shiftRepo,
		c.groupRepo,
		c.ruleApplyScopeRepo,
		c.ruleDependencyRepo,
		c.ruleConflictRepo,
	)
	c.logger.Info("RuleParserService created successfully")

	// V4新增：规则组织服务
	c.ruleOrganizerService = service_impl.NewRuleOrganizerService(
		c.logger,
		c.schedulingRuleRepo,
		c.ruleDependencyRepo,
		c.ruleConflictRepo,
		c.shiftDependencyRepo,
		c.shiftRepo,
	)
	c.logger.Info("RuleOrganizerService created successfully")

	// V4新增：规则迁移服务
	c.ruleMigrationService = service_impl.NewRuleMigrationService(
		c.logger,
		c.schedulingRuleRepo,
		c.ruleDependencyRepo,
		c.ruleConflictRepo,
	)
	c.logger.Info("RuleMigrationService created successfully")

	// V4新增：规则统计服务
	c.ruleStatisticsService = service_impl.NewRuleStatisticsService(
		c.logger,
		c.schedulingRuleRepo,
	)
	c.logger.Info("RuleStatisticsService created successfully")

	c.logger.Info("All services initialized successfully")
}

// GetEmployeeService 获取员工服务
func (c *Container) GetEmployeeService() domain_service.IEmployeeService {
	return c.employeeService
}

// GetGroupService 获取分组服务
func (c *Container) GetGroupService() domain_service.IGroupService {
	return c.groupService
}

// GetShiftService 获取班次服务
func (c *Container) GetShiftService() domain_service.IShiftService {
	return c.shiftService
}

// GetSchedulingService 获取排班服务
func (c *Container) GetSchedulingService() domain_service.ISchedulingService {
	return c.schedulingService
}

// GetLeaveService 获取假期服务
func (c *Container) GetLeaveService() domain_service.ILeaveService {
	return c.leaveService
}

// GetSchedulingRuleService 获取排班规则服务
func (c *Container) GetSchedulingRuleService() domain_service.ISchedulingRuleService {
	return c.schedulingRuleService
}

// GetDepartmentService 获取部门服务
func (c *Container) GetDepartmentService() domain_service.IDepartmentService {
	return c.departmentService
}

// GetShiftGroupRepository 获取班次-分组关联仓储
func (c *Container) GetShiftGroupRepository() repository.IShiftGroupRepository {
	return c.shiftGroupRepo
}

// ModalityRoomService 获取机房服务
func (c *Container) ModalityRoomService() domain_service.IModalityRoomService {
	return c.modalityRoomService
}

// TimePeriodService 获取时间段服务
func (c *Container) TimePeriodService() domain_service.ITimePeriodService {
	return c.timePeriodService
}

// ScanTypeService 获取检查类型服务
func (c *Container) ScanTypeService() domain_service.IScanTypeService {
	return c.scanTypeService
}

// ModalityRoomWeeklyVolumeService 获取机房周检查量服务
func (c *Container) ModalityRoomWeeklyVolumeService() domain_service.IModalityRoomWeeklyVolumeService {
	return c.modalityRoomWeeklyVolumeService
}

// StaffingCalculationService 获取排班人数计算服务
func (c *Container) StaffingCalculationService() domain_service.IStaffingCalculationService {
	return c.staffingCalculationService
}

// ShiftWeeklyStaffRepository 获取班次周人数仓储
func (c *Container) ShiftWeeklyStaffRepository() repository.IShiftWeeklyStaffRepository {
	return c.shiftWeeklyStaffRepo
}

// GetShiftFixedAssignmentService 获取班次固定人员配置服务
func (c *Container) GetShiftFixedAssignmentService() domain_service.IShiftFixedAssignmentService {
	return c.shiftFixedAssignmentService
}

// GetSystemSettingService 获取系统设置服务
func (c *Container) GetSystemSettingService() domain_service.ISystemSettingService {
	return c.systemSettingService
}

// GetConversationService 获取对话记录服务
func (c *Container) GetConversationService() domain_service.IConversationService {
	return c.conversationService
}

// GetRuleParserService 获取规则解析服务
func (c *Container) GetRuleParserService() domain_service.IRuleParserService {
	return c.ruleParserService
}

// GetRuleOrganizerService 获取规则组织服务
func (c *Container) GetRuleOrganizerService() domain_service.IRuleOrganizerService {
	return c.ruleOrganizerService
}

// GetRuleMigrationService 获取规则迁移服务
func (c *Container) GetRuleMigrationService() domain_service.IRuleMigrationService {
	return c.ruleMigrationService
}

// GetRuleStatisticsService 获取规则统计服务
func (c *Container) GetRuleStatisticsService() domain_service.IRuleStatisticsService {
	return c.ruleStatisticsService
}

// GetRuleDependencyRepo 获取规则依赖仓储
func (c *Container) GetRuleDependencyRepo() repository.IRuleDependencyRepository {
	return c.ruleDependencyRepo
}

// GetRuleConflictRepo 获取规则冲突仓储
func (c *Container) GetRuleConflictRepo() repository.IRuleConflictRepository {
	return c.ruleConflictRepo
}

// Close 关闭资源
func (c *Container) Close() error {
	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
